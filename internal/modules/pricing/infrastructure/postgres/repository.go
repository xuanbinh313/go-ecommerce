package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/pricing/domain"
	"github.com/example/medusa-go-clone/internal/modules/pricing/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const preferenceCols = `id,attribute,value,is_tax_inclusive,created_at,updated_at`

func scanPreference(row pgx.Row) (models.PricePreference, error) {
	var v models.PricePreference
	err := row.Scan(&v.ID, &v.Attribute, &v.Value, &v.IsTaxInclusive, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func (r *Repository) ListPreferences(ctx context.Context, attrs, values []string, limit, offset int) ([]models.PricePreference, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	if len(attrs) > 0 {
		args = append(args, attrs)
		where = append(where, fmt.Sprintf("attribute=ANY($%d)", len(args)))
	}
	if len(values) > 0 {
		args = append(args, values)
		where = append(where, fmt.Sprintf("value=ANY($%d)", len(args)))
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM price_preference"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM price_preference%s ORDER BY created_at LIMIT $%d OFFSET $%d", preferenceCols, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.PricePreference{}
	for rows.Next() {
		v, e := scanPreference(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetPreference(ctx context.Context, id string) (models.PricePreference, error) {
	return scanPreference(r.db.QueryRow(ctx, "SELECT "+preferenceCols+" FROM price_preference WHERE id=$1 AND deleted_at IS NULL", id))
}
func (r *Repository) CreatePreference(ctx context.Context, in models.PreferenceInput) (models.PricePreference, error) {
	inclusive := false
	if in.IsTaxInclusive != nil {
		inclusive = *in.IsTaxInclusive
	}
	return scanPreference(r.db.QueryRow(ctx, "INSERT INTO price_preference(id,attribute,value,is_tax_inclusive) VALUES($1,$2,$3,$4) RETURNING "+preferenceCols, entityid.New("prpref"), in.Attribute, in.Value, inclusive))
}
func (r *Repository) UpdatePreference(ctx context.Context, id string, in models.PreferenceUpdate) (models.PricePreference, error) {
	sets := []string{}
	args := []any{id}
	put := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	if in.Attribute != nil {
		put("attribute", *in.Attribute)
	}
	if in.Value != nil {
		put("value", *in.Value)
	}
	if in.IsTaxInclusive != nil {
		put("is_tax_inclusive", *in.IsTaxInclusive)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := r.db.Exec(ctx, "UPDATE price_preference SET "+strings.Join(sets, ",")+" WHERE id=$1 AND deleted_at IS NULL", args...)
		if err != nil {
			return models.PricePreference{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.PricePreference{}, pgx.ErrNoRows
		}
	}
	return r.GetPreference(ctx, id)
}
func (r *Repository) DeletePreference(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "UPDATE price_preference SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

var _ domain.Repository = (*Repository)(nil)

const priceListCols = `id,title,description,status,type,starts_at,ends_at,metadata,created_at,updated_at`

func scanPriceList(row pgx.Row) (models.PriceList, error) {
	var v models.PriceList
	err := row.Scan(&v.ID, &v.Title, &v.Description, &v.Status, &v.Type, &v.StartsAt, &v.EndsAt, &v.Metadata, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func (r *Repository) ListPriceLists(ctx context.Context, ids, statuses, types []string, limit, offset int) ([]models.PriceList, int, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	for _, f := range []struct {
		col    string
		values []string
	}{{"id", ids}, {"status", statuses}, {"type", types}} {
		if len(f.values) > 0 {
			args = append(args, f.values)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", f.col, len(args)))
		}
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM price_list"+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM price_list%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", priceListCols, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.PriceList{}
	for rows.Next() {
		v, e := scanPriceList(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetPriceList(ctx context.Context, id string) (models.PriceList, error) {
	return scanPriceList(r.db.QueryRow(ctx, "SELECT "+priceListCols+" FROM price_list WHERE id=$1 AND deleted_at IS NULL", id))
}
func (r *Repository) CreatePriceList(ctx context.Context, in models.PriceList) (models.PriceList, error) {
	if in.Status == "" {
		in.Status = "draft"
	}
	if in.Type == "" {
		in.Type = "sale"
	}
	if in.Metadata == nil {
		in.Metadata = json.RawMessage("null")
	}
	return scanPriceList(r.db.QueryRow(ctx, "INSERT INTO price_list(id,title,description,status,type,starts_at,ends_at,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "+priceListCols, entityid.New("plist"), in.Title, in.Description, in.Status, in.Type, in.StartsAt, in.EndsAt, in.Metadata))
}
func (r *Repository) UpdatePriceList(ctx context.Context, id string, in models.PriceList) (models.PriceList, error) {
	sets, args := []string{}, []any{id}
	put := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	if in.Title != "" {
		put("title", in.Title)
	}
	if in.Description != "" {
		put("description", in.Description)
	}
	if in.Status != "" {
		put("status", in.Status)
	}
	if in.Type != "" {
		put("type", in.Type)
	}
	if in.StartsAt != nil {
		put("starts_at", in.StartsAt)
	}
	if in.EndsAt != nil {
		put("ends_at", in.EndsAt)
	}
	if in.Metadata != nil {
		put("metadata", in.Metadata)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := r.db.Exec(ctx, "UPDATE price_list SET "+strings.Join(sets, ",")+" WHERE id=$1 AND deleted_at IS NULL", args...)
		if err != nil {
			return models.PriceList{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.PriceList{}, pgx.ErrNoRows
		}
	}
	return r.GetPriceList(ctx, id)
}
func (r *Repository) DeletePriceList(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, "UPDATE price_list SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	for _, table := range []string{"price", "price_list_rule"} {
		if _, err = tx.Exec(ctx, "UPDATE "+table+" SET deleted_at=now(),updated_at=now() WHERE price_list_id=$1 AND deleted_at IS NULL", id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
