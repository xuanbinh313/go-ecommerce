package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/tax/domain"
	"github.com/example/medusa-go-clone/internal/modules/tax/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) ListProviders(ctx context.Context, ids []string, enabled *bool, limit, offset int) ([]models.TaxProvider, int, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	if len(ids) > 0 {
		args = append(args, ids)
		where = append(where, fmt.Sprintf("id=ANY($%d)", len(args)))
	}
	if enabled != nil {
		args = append(args, *enabled)
		where = append(where, fmt.Sprintf("is_enabled=$%d", len(args)))
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM tax_provider"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT id,is_enabled FROM tax_provider%s ORDER BY id LIMIT $%d OFFSET $%d", w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.TaxProvider{}
	for rows.Next() {
		var item models.TaxProvider
		if err := rows.Scan(&item.ID, &item.IsEnabled); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, count, rows.Err()
}

const regionCols = `id,country_code,province_code,provider_id,parent_id,metadata,created_by,created_at,updated_at`
const rateCols = `id,tax_region_id,rate,code,name,is_default,is_combinable,metadata,created_by,created_at,updated_at`

func scanRegion(row pgx.Row) (models.TaxRegion, error) {
	var v models.TaxRegion
	err := row.Scan(&v.ID, &v.CountryCode, &v.ProvinceCode, &v.ProviderID, &v.ParentID, &v.Metadata, &v.CreatedBy, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func scanRate(row pgx.Row) (models.TaxRate, error) {
	var v models.TaxRate
	err := row.Scan(&v.ID, &v.TaxRegionID, &v.Rate, &v.Code, &v.Name, &v.IsDefault, &v.IsCombinable, &v.Metadata, &v.CreatedBy, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func (r *Repository) ListRegions(ctx context.Context, ids, countries, parents []string, limit, offset int) ([]models.TaxRegion, int, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	for _, f := range []struct {
		c string
		v []string
	}{{"id", ids}, {"country_code", countries}, {"parent_id", parents}} {
		if len(f.v) > 0 {
			args = append(args, f.v)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", f.c, len(args)))
		}
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM tax_region"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM tax_region%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", regionCols, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.TaxRegion{}
	for rows.Next() {
		v, e := scanRegion(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetRegion(ctx context.Context, id string) (models.TaxRegion, error) {
	v, err := scanRegion(r.db.QueryRow(ctx, "SELECT "+regionCols+" FROM tax_region WHERE id=$1 AND deleted_at IS NULL", id))
	if err != nil {
		return v, err
	}
	v.TaxRates, _, err = r.ListRates(ctx, nil, []string{id}, nil, 100, 0)
	return v, err
}
func (r *Repository) CreateRegion(ctx context.Context, in models.RegionInput) (models.TaxRegion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.TaxRegion{}, err
	}
	defer tx.Rollback(ctx)
	id := entityid.New("txreg")
	meta := in.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage("null")
	}
	_, err = tx.Exec(ctx, `INSERT INTO tax_region(id,country_code,province_code,provider_id,parent_id,metadata,created_by) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, in.CountryCode, in.ProvinceCode, in.ProviderID, in.ParentID, meta, in.CreatedBy)
	if err != nil {
		return models.TaxRegion{}, err
	}
	if in.DefaultTaxRate != nil {
		v := *in.DefaultTaxRate
		v.IsDefault = boolptr(true)
		if _, err = insertRate(ctx, tx, id, v, in.CreatedBy); err != nil {
			return models.TaxRegion{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.TaxRegion{}, err
	}
	return r.GetRegion(ctx, id)
}
func insertRate(ctx context.Context, tx pgx.Tx, regionID string, in models.RateInput, actor *string) (models.TaxRate, error) {
	d, c := false, false
	if in.IsDefault != nil {
		d = *in.IsDefault
	}
	if in.IsCombinable != nil {
		c = *in.IsCombinable
	}
	m := in.Metadata
	if len(m) == 0 {
		m = json.RawMessage("null")
	}
	id := entityid.New("txr")
	v, err := scanRate(tx.QueryRow(ctx, `INSERT INTO tax_rate(id,tax_region_id,rate,code,name,is_default,is_combinable,metadata,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+rateCols, id, regionID, in.Rate, in.Code, in.Name, d, c, m, actor))
	if err != nil {
		return v, err
	}
	for _, rule := range in.Rules {
		if _, err = tx.Exec(ctx, `INSERT INTO tax_rate_rule(id,tax_rate_id,reference,reference_id,created_by) VALUES($1,$2,$3,$4,$5)`, entityid.New("txrule"), id, rule.Reference, rule.ReferenceID, actor); err != nil {
			return v, err
		}
	}
	return v, nil
}
func boolptr(v bool) *bool { return &v }
func (r *Repository) UpdateRegion(ctx context.Context, id string, in models.RegionUpdate) (models.TaxRegion, error) {
	sets, args := []string{}, []any{id}
	put := func(c string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", c, len(args)))
	}
	if in.ProvinceCode != nil {
		put("province_code", *in.ProvinceCode)
	}
	if in.ProviderID != nil {
		put("provider_id", *in.ProviderID)
	}
	if in.Metadata != nil {
		put("metadata", in.Metadata)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := r.db.Exec(ctx, "UPDATE tax_region SET "+strings.Join(sets, ",")+" WHERE id=$1 AND deleted_at IS NULL", args...)
		if err != nil {
			return models.TaxRegion{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.TaxRegion{}, pgx.ErrNoRows
		}
	}
	return r.GetRegion(ctx, id)
}
func (r *Repository) DeleteRegion(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `WITH RECURSIVE descendants(id) AS (SELECT id FROM tax_region WHERE id=$1 UNION ALL SELECT t.id FROM tax_region t JOIN descendants d ON t.parent_id=d.id), rates(id) AS (SELECT id FROM tax_rate WHERE tax_region_id IN (SELECT id FROM descendants)) UPDATE tax_rate_rule SET deleted_at=now(),updated_at=now() WHERE tax_rate_id IN (SELECT id FROM rates) AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `WITH RECURSIVE descendants(id) AS (SELECT id FROM tax_region WHERE id=$1 UNION ALL SELECT t.id FROM tax_region t JOIN descendants d ON t.parent_id=d.id) UPDATE tax_rate SET deleted_at=now(),updated_at=now() WHERE tax_region_id IN (SELECT id FROM descendants) AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `WITH RECURSIVE descendants(id) AS (SELECT id FROM tax_region WHERE id=$1 UNION ALL SELECT t.id FROM tax_region t JOIN descendants d ON t.parent_id=d.id) UPDATE tax_region SET deleted_at=now(),updated_at=now() WHERE id IN (SELECT id FROM descendants) AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) ListRates(ctx context.Context, ids, regionIDs, codes []string, limit, offset int) ([]models.TaxRate, int, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	for _, f := range []struct {
		c string
		v []string
	}{{"id", ids}, {"tax_region_id", regionIDs}, {"code", codes}} {
		if len(f.v) > 0 {
			args = append(args, f.v)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", f.c, len(args)))
		}
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM tax_rate"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM tax_rate%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", rateCols, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.TaxRate{}
	for rows.Next() {
		v, e := scanRate(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetRate(ctx context.Context, id string) (models.TaxRate, error) {
	v, err := scanRate(r.db.QueryRow(ctx, "SELECT "+rateCols+" FROM tax_rate WHERE id=$1 AND deleted_at IS NULL", id))
	if err != nil {
		return v, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,tax_rate_id,reference,reference_id,metadata,created_by FROM tax_rate_rule WHERE tax_rate_id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	v.Rules = []models.TaxRateRule{}
	for rows.Next() {
		var x models.TaxRateRule
		if err = rows.Scan(&x.ID, &x.TaxRateID, &x.Reference, &x.ReferenceID, &x.Metadata, &x.CreatedBy); err != nil {
			return v, err
		}
		v.Rules = append(v.Rules, x)
	}
	return v, rows.Err()
}
func (r *Repository) CreateRate(ctx context.Context, in models.RateInput, regionID string, actor *string) (models.TaxRate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.TaxRate{}, err
	}
	defer tx.Rollback(ctx)
	v, err := insertRate(ctx, tx, regionID, in, actor)
	if err != nil {
		return v, err
	}
	if err = tx.Commit(ctx); err != nil {
		return v, err
	}
	return r.GetRate(ctx, v.ID)
}
func (r *Repository) UpdateRate(ctx context.Context, id string, in models.RateUpdate, actor *string) (models.TaxRate, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.TaxRate{}, err
	}
	defer tx.Rollback(ctx)
	sets, args := []string{}, []any{id}
	put := func(c string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", c, len(args)))
	}
	if in.Rate != nil {
		put("rate", *in.Rate)
	}
	if in.Code != nil {
		put("code", *in.Code)
	}
	if in.Name != nil {
		put("name", *in.Name)
	}
	if in.IsDefault != nil {
		put("is_default", *in.IsDefault)
	}
	if in.IsCombinable != nil {
		put("is_combinable", *in.IsCombinable)
	}
	if in.Metadata != nil {
		put("metadata", in.Metadata)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := tx.Exec(ctx, "UPDATE tax_rate SET "+strings.Join(sets, ",")+" WHERE id=$1 AND deleted_at IS NULL", args...)
		if err != nil {
			return models.TaxRate{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.TaxRate{}, pgx.ErrNoRows
		}
	}
	if in.Rules != nil {
		if _, err := tx.Exec(ctx, "UPDATE tax_rate_rule SET deleted_at=now(),updated_at=now() WHERE tax_rate_id=$1 AND deleted_at IS NULL", id); err != nil {
			return models.TaxRate{}, err
		}
		for _, rule := range *in.Rules {
			if _, err := tx.Exec(ctx, `INSERT INTO tax_rate_rule(id,tax_rate_id,reference,reference_id,created_by) VALUES($1,$2,$3,$4,$5)`, entityid.New("txrule"), id, rule.Reference, rule.ReferenceID, actor); err != nil {
				return models.TaxRate{}, err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.TaxRate{}, err
	}
	return r.GetRate(ctx, id)
}
func (r *Repository) DeleteRate(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, "UPDATE tax_rate SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, "UPDATE tax_rate_rule SET deleted_at=now(),updated_at=now() WHERE tax_rate_id=$1 AND deleted_at IS NULL", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) CreateRateRules(ctx context.Context, rules []models.RuleCreateInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, rule := range rules {
		metadata := rule.Metadata
		if len(metadata) == 0 {
			metadata = []byte(`{}`)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO tax_rate_rule(id,tax_rate_id,reference,reference_id,metadata,created_by) VALUES($1,$2,$3,$4,$5,$6)`, entityid.New("txrule"), rule.TaxRateID, rule.Reference, rule.ReferenceID, metadata, rule.CreatedBy); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteRateRules(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `UPDATE tax_rate_rule SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	return err
}

var _ domain.Repository = (*Repository)(nil)
