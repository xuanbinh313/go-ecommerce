package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/promotion/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const campaignCols = `id,name,description,campaign_identifier,starts_at,ends_at,created_at,updated_at,deleted_at`

func scanCampaign(row pgx.Row) (models.Campaign, error) {
	var v models.Campaign
	err := row.Scan(&v.ID, &v.Name, &v.Description, &v.CampaignIdentifier, &v.StartsAt, &v.EndsAt, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}
func (r *Repository) campaignWithBudget(ctx context.Context, v models.Campaign) (models.Campaign, error) {
	var b models.CampaignBudget
	err := r.db.QueryRow(ctx, `SELECT id,type,currency_code,raw_limit,raw_used,attribute FROM promotion_campaign_budget WHERE campaign_id=$1 AND deleted_at IS NULL`, v.ID).Scan(&b.ID, &b.Type, &b.CurrencyCode, &b.Limit, &b.Used, &b.Attribute)
	if err == nil {
		v.Budget = &b
		if b.CurrencyCode != nil {
			var currency map[string]any
			if e := r.db.QueryRow(ctx, `SELECT jsonb_build_object('code',code,'symbol',symbol,'symbol_native',symbol_native,'name',name,'decimal_digits',decimal_digits,'rounding',rounding) FROM currency WHERE code=$1`, *b.CurrencyCode).Scan(&currency); e == nil {
				v.Currency = currency
			} else if e != pgx.ErrNoRows {
				return v, e
			}
		}
	} else if err != pgx.ErrNoRows {
		return v, err
	}
	return v, nil
}
func campaignLimitValue(raw json.RawMessage) (any, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var n json.Number
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}
	num, ok := value.(json.Number)
	if !ok {
		return nil, fmt.Errorf("budget.limit must be a number or null")
	}
	n = num
	return n.String(), nil
}
func (r *Repository) ListCampaigns(ctx context.Context, f models.CampaignFilter) ([]models.Campaign, int, error) {
	where, args := []string{"c.deleted_at IS NULL"}, []any{}
	if f.Identifier != "" {
		args = append(args, f.Identifier)
		where = append(where, fmt.Sprintf("c.campaign_identifier=$%d", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(c.name ILIKE $%d OR c.description ILIKE $%d OR c.campaign_identifier ILIKE $%d)", len(args), len(args), len(args)))
	}
	if f.BudgetCurrency != "" {
		args = append(args, f.BudgetCurrency)
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM promotion_campaign_budget b WHERE b.campaign_id=c.id AND b.deleted_at IS NULL AND b.currency_code=$%d)", len(args)))
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM promotion_campaign c"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM promotion_campaign c%s ORDER BY c.created_at DESC LIMIT $%d OFFSET $%d", campaignCols, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.Campaign{}
	for rows.Next() {
		v, e := scanCampaign(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	for i := range out {
		out[i], err = r.campaignWithBudget(ctx, out[i])
		if err != nil {
			return nil, 0, err
		}
	}
	return out, count, nil
}
func (r *Repository) GetCampaign(ctx context.Context, id string) (models.Campaign, error) {
	v, err := scanCampaign(r.db.QueryRow(ctx, "SELECT "+campaignCols+" FROM promotion_campaign WHERE id=$1 AND deleted_at IS NULL", id))
	if err != nil {
		return v, err
	}
	return r.campaignWithBudget(ctx, v)
}
func (r *Repository) CreateCampaign(ctx context.Context, in models.CampaignInput) (models.Campaign, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Campaign{}, err
	}
	defer tx.Rollback(ctx)
	id := entityid.New("procamp")
	_, err = tx.Exec(ctx, `INSERT INTO promotion_campaign(id,name,description,campaign_identifier,starts_at,ends_at) VALUES($1,$2,$3,$4,$5,$6)`, id, in.Name, in.Description, in.CampaignIdentifier, in.StartsAt, in.EndsAt)
	if err != nil {
		return models.Campaign{}, err
	}
	if b := in.Budget; b != nil {
		limit, limitErr := campaignLimitValue(b.Limit)
		if limitErr != nil {
			return models.Campaign{}, limitErr
		}
		_, err = tx.Exec(ctx, `INSERT INTO promotion_campaign_budget(id,campaign_id,type,currency_code,"limit",raw_limit,attribute) VALUES($1,$2,$3,$4,$5,CASE WHEN $5::numeric IS NULL THEN NULL ELSE jsonb_build_object('value',$5::text,'precision',20) END,$6)`, entityid.New("probudg"), id, b.Type, b.CurrencyCode, limit, b.Attribute)
		if err != nil {
			return models.Campaign{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Campaign{}, err
	}
	return r.GetCampaign(ctx, id)
}
func (r *Repository) UpdateCampaign(ctx context.Context, id string, in models.CampaignUpdate) (models.Campaign, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Campaign{}, err
	}
	defer tx.Rollback(ctx)
	sets, args := []string{}, []any{}
	put := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	if in.Name != nil {
		put("name", *in.Name)
	}
	if in.Description != nil {
		put("description", *in.Description)
	}
	if in.CampaignIdentifier != nil {
		put("campaign_identifier", *in.CampaignIdentifier)
	}
	if in.StartsAt != nil {
		put("starts_at", *in.StartsAt)
	}
	if in.EndsAt != nil {
		put("ends_at", *in.EndsAt)
	}
	if len(sets) > 0 {
		args = append(args, id)
		tag, e := tx.Exec(ctx, "UPDATE promotion_campaign SET "+strings.Join(sets, ",")+",updated_at=now() WHERE id=$"+fmt.Sprint(len(args))+" AND deleted_at IS NULL", args...)
		if e != nil {
			return models.Campaign{}, e
		}
		if tag.RowsAffected() == 0 {
			return models.Campaign{}, pgx.ErrNoRows
		}
	}
	if in.Budget != nil {
		b := in.Budget
		if len(b.Limit) > 0 {
			limit, limitErr := campaignLimitValue(b.Limit)
			if limitErr != nil {
				return models.Campaign{}, limitErr
			}
			tag, e := tx.Exec(ctx, `UPDATE promotion_campaign_budget SET "limit"=$2,raw_limit=CASE WHEN $2::numeric IS NULL THEN NULL ELSE jsonb_build_object('value',$2::text,'precision',20) END,updated_at=now() WHERE campaign_id=$1 AND deleted_at IS NULL`, id, limit)
			if e != nil {
				return models.Campaign{}, e
			}
			if tag.RowsAffected() == 0 {
				return models.Campaign{}, pgx.ErrNoRows
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Campaign{}, err
	}
	return r.GetCampaign(ctx, id)
}
func (r *Repository) DeleteCampaign(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE promotion_campaign SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) LinkCampaignPromotions(ctx context.Context, id string, add, remove []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM promotion_campaign WHERE id=$1 AND deleted_at IS NULL)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	if len(add) > 0 {
		tag, e := tx.Exec(ctx, `UPDATE promotion SET campaign_id=$1,updated_at=now() WHERE id=ANY($2) AND deleted_at IS NULL`, id, add)
		if e != nil {
			return e
		}
		if int(tag.RowsAffected()) != len(add) {
			return pgx.ErrNoRows
		}
	}
	if len(remove) > 0 {
		_, err = tx.Exec(ctx, `UPDATE promotion SET campaign_id=NULL,updated_at=now() WHERE campaign_id=$1 AND id=ANY($2) AND deleted_at IS NULL`, id, remove)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
