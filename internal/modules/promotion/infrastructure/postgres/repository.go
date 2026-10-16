package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/promotion/domain"
	"github.com/example/medusa-go-clone/internal/modules/promotion/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const promotionCols = `id,code,is_automatic,is_tax_inclusive,"limit",used,type,status,campaign_id,metadata,created_at,updated_at`

func scanPromotion(row pgx.Row) (models.Promotion, error) {
	var p models.Promotion
	err := row.Scan(&p.ID, &p.Code, &p.IsAutomatic, &p.IsTaxInclusive, &p.Limit, &p.Used, &p.Type, &p.Status, &p.CampaignID, &p.Metadata, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
func (r *Repository) Get(ctx context.Context, idOrCode string) (models.Promotion, error) {
	p, err := scanPromotion(r.db.QueryRow(ctx, "SELECT "+promotionCols+" FROM promotion WHERE (id=$1 OR code=$1) AND deleted_at IS NULL LIMIT 1", idOrCode))
	if err != nil {
		return p, err
	}
	var m models.ApplicationMethod
	err = r.db.QueryRow(ctx, `SELECT id,promotion_id,value,currency_code,max_quantity,apply_to_quantity,buy_rules_min_quantity,type,target_type,allocation FROM promotion_application_method WHERE promotion_id=$1 AND deleted_at IS NULL`, p.ID).Scan(&m.ID, &m.PromotionID, &m.Value, &m.CurrencyCode, &m.MaxQuantity, &m.ApplyToQuantity, &m.BuyRulesMinQuantity, &m.Type, &m.TargetType, &m.Allocation)
	if err == nil {
		p.ApplicationMethod = &m
		p.ApplicationMethod.TargetRules, err = r.loadRules(ctx, `SELECT r.id,r.description,r.attribute,r.operator FROM promotion_rule r JOIN application_method_target_rules l ON l.rule_id=r.id WHERE l.application_method_id=$1 AND r.deleted_at IS NULL`, m.ID)
		if err != nil {
			return p, err
		}
		p.ApplicationMethod.BuyRules, err = r.loadRules(ctx, `SELECT r.id,r.description,r.attribute,r.operator FROM promotion_rule r JOIN application_method_buy_rules l ON l.rule_id=r.id WHERE l.application_method_id=$1 AND r.deleted_at IS NULL`, m.ID)
		if err != nil {
			return p, err
		}
	} else if err != pgx.ErrNoRows {
		return p, err
	}
	p.Rules, err = r.loadRules(ctx, `SELECT r.id,r.description,r.attribute,r.operator FROM promotion_rule r JOIN promotion_promotion_rule l ON l.rule_id=r.id WHERE l.promotion_id=$1 AND r.deleted_at IS NULL`, p.ID)
	return p, err
}
func (r *Repository) loadRules(ctx context.Context, query string, id string) ([]models.Rule, error) {
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Rule{}
	for rows.Next() {
		var v models.Rule
		if err = rows.Scan(&v.ID, &v.Description, &v.Attribute, &v.Operator); err != nil {
			return nil, err
		}
		values, err := r.db.Query(ctx, "SELECT value FROM promotion_rule_value WHERE promotion_rule_id=$1 AND deleted_at IS NULL ORDER BY created_at", v.ID)
		if err != nil {
			return nil, err
		}
		for values.Next() {
			var x string
			if err = values.Scan(&x); err != nil {
				values.Close()
				return nil, err
			}
			v.Values = append(v.Values, x)
		}
		values.Close()
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) List(ctx context.Context, f models.Filter) ([]models.Promotion, int, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	for _, x := range []struct {
		col  string
		vals []string
	}{{"id", f.IDs}, {"code", f.Codes}, {"campaign_id", f.CampaignIDs}, {"status", f.Statuses}} {
		if len(x.vals) > 0 {
			args = append(args, x.vals)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", x.col, len(args)))
		}
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM promotion"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT id FROM promotion%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	out := make([]models.Promotion, 0, len(ids))
	for _, id := range ids {
		v, e := r.Get(ctx, id)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, nil
}
func (r *Repository) Create(ctx context.Context, in models.CreateInput) (models.Promotion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Promotion{}, err
	}
	defer tx.Rollback(ctx)
	id := entityid.New("promo")
	automatic, taxInclusive := false, false
	if in.IsAutomatic != nil {
		automatic = *in.IsAutomatic
	}
	if in.IsTaxInclusive != nil {
		taxInclusive = *in.IsTaxInclusive
	}
	status := in.Status
	if status == "" {
		status = "draft"
	}
	meta := in.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage("null")
	}
	_, err = tx.Exec(ctx, `INSERT INTO promotion(id,code,is_automatic,is_tax_inclusive,"limit",used,type,status,campaign_id,metadata) VALUES($1,$2,$3,$4,$5,0,$6,$7,$8,$9)`, id, in.Code, automatic, taxInclusive, in.Limit, in.Type, status, in.CampaignID, meta)
	if err != nil {
		return models.Promotion{}, err
	}
	if err = r.insertMethod(ctx, tx, id, in.ApplicationMethod); err != nil {
		return models.Promotion{}, err
	}
	for _, rule := range in.Rules {
		ruleID, e := insertRule(ctx, tx, rule)
		if e != nil {
			return models.Promotion{}, e
		}
		if _, err = tx.Exec(ctx, "INSERT INTO promotion_promotion_rule(promotion_id,rule_id) VALUES($1,$2)", id, ruleID); err != nil {
			return models.Promotion{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Promotion{}, err
	}
	return r.Get(ctx, id)
}
func rawNumber(v float64) []byte {
	b, _ := json.Marshal(map[string]any{"value": strconv.FormatFloat(v, 'f', -1, 64), "precision": 20})
	return b
}
func (r *Repository) insertMethod(ctx context.Context, tx pgx.Tx, promotionID string, in models.ApplicationMethodInput) error {
	id := entityid.New("proappmet")
	var value any
	var raw []byte
	if in.Value != nil {
		value = *in.Value
		raw = rawNumber(*in.Value)
	} else {
		value = nil
		raw = []byte("null")
	}
	_, err := tx.Exec(ctx, `INSERT INTO promotion_application_method(id,promotion_id,value,raw_value,currency_code,max_quantity,apply_to_quantity,buy_rules_min_quantity,type,target_type,allocation) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, promotionID, value, raw, in.CurrencyCode, in.MaxQuantity, in.ApplyToQuantity, in.BuyRulesMinQuantity, in.Type, in.TargetType, in.Allocation)
	if err != nil {
		return err
	}
	for _, x := range in.TargetRules {
		ruleID, e := insertRule(ctx, tx, x)
		if e != nil {
			return e
		}
		if _, err = tx.Exec(ctx, "INSERT INTO application_method_target_rules(application_method_id,rule_id) VALUES($1,$2)", id, ruleID); err != nil {
			return err
		}
	}
	for _, x := range in.BuyRules {
		ruleID, e := insertRule(ctx, tx, x)
		if e != nil {
			return e
		}
		if _, err = tx.Exec(ctx, "INSERT INTO application_method_buy_rules(application_method_id,rule_id) VALUES($1,$2)", id, ruleID); err != nil {
			return err
		}
	}
	return nil
}
func insertRule(ctx context.Context, tx pgx.Tx, in models.RuleInput) (string, error) {
	id := entityid.New("prorul")
	if _, err := tx.Exec(ctx, `INSERT INTO promotion_rule(id,description,attribute,operator) VALUES($1,$2,$3,$4)`, id, in.Description, in.Attribute, in.Operator); err != nil {
		return "", err
	}
	for _, v := range in.Values {
		if _, err := tx.Exec(ctx, `INSERT INTO promotion_rule_value(id,promotion_rule_id,value) VALUES($1,$2,$3)`, entityid.New("prorulval"), id, v); err != nil {
			return "", err
		}
	}
	return id, nil
}
func (r *Repository) Update(ctx context.Context, id string, in models.UpdateInput) (models.Promotion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Promotion{}, err
	}
	defer tx.Rollback(ctx)
	sets, args := []string{}, []any{id}
	put := func(c string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", c, len(args)))
	}
	if in.Code != nil {
		put("code", *in.Code)
	}
	if in.IsAutomatic != nil {
		put("is_automatic", *in.IsAutomatic)
	}
	if in.IsTaxInclusive != nil {
		put("is_tax_inclusive", *in.IsTaxInclusive)
	}
	if in.Limit != nil {
		put(`"limit"`, *in.Limit)
	}
	if in.Type != nil {
		put("type", *in.Type)
	}
	if in.Status != nil {
		put("status", *in.Status)
	}
	if in.CampaignID != nil {
		put("campaign_id", *in.CampaignID)
	}
	if in.Metadata != nil {
		put("metadata", in.Metadata)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := tx.Exec(ctx, "UPDATE promotion SET "+strings.Join(sets, ",")+" WHERE id=$1 AND deleted_at IS NULL", args...)
		if err != nil {
			return models.Promotion{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.Promotion{}, pgx.ErrNoRows
		}
	}
	if in.ApplicationMethod != nil {
		if err := r.updateMethod(ctx, tx, id, *in.ApplicationMethod); err != nil {
			return models.Promotion{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Promotion{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repository) updateMethod(ctx context.Context, tx pgx.Tx, promotionID string, in models.ApplicationMethodInput) error {
	sets, args := []string{}, []any{promotionID}
	put := func(c string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", c, len(args)))
	}
	if in.Value != nil {
		put("value", *in.Value)
		put("raw_value", rawNumber(*in.Value))
	}
	if in.CurrencyCode != nil {
		put("currency_code", *in.CurrencyCode)
	}
	if in.MaxQuantity != nil {
		put("max_quantity", *in.MaxQuantity)
	}
	if in.Type != "" {
		put("type", in.Type)
	}
	if in.TargetType != "" {
		put("target_type", in.TargetType)
	}
	if in.Allocation != nil {
		put("allocation", *in.Allocation)
	}
	if in.ApplyToQuantity != nil {
		put("apply_to_quantity", *in.ApplyToQuantity)
	}
	if in.BuyRulesMinQuantity != nil {
		put("buy_rules_min_quantity", *in.BuyRulesMinQuantity)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at=now()")
	_, err := tx.Exec(ctx, "UPDATE promotion_application_method SET "+strings.Join(sets, ",")+" WHERE promotion_id=$1 AND deleted_at IS NULL", args...)
	return err
}
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, "UPDATE promotion SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, "UPDATE promotion_application_method SET deleted_at=now(),updated_at=now() WHERE promotion_id=$1 AND deleted_at IS NULL", id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "UPDATE application_method_target_rules SET deleted_at=now(),updated_at=now() WHERE application_method_id IN (SELECT id FROM promotion_application_method WHERE promotion_id=$1)", id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "UPDATE application_method_buy_rules SET deleted_at=now(),updated_at=now() WHERE application_method_id IN (SELECT id FROM promotion_application_method WHERE promotion_id=$1)", id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "UPDATE promotion_promotion_rule SET deleted_at=now(),updated_at=now() WHERE promotion_id=$1 AND deleted_at IS NULL", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ domain.Repository = (*Repository)(nil)
