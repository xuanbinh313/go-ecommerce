package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const shippingOptionTypeColumns = `id,label,description,code,created_at,updated_at`

func scanShippingOptionType(row pgx.Row) (models.ShippingOptionType, error) {
	var v models.ShippingOptionType
	err := row.Scan(&v.ID, &v.Label, &v.Description, &v.Code, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

func (r *Repository) ListShippingOptionTypes(ctx context.Context, f models.ShippingOptionTypeFilter, limit, offset int) ([]models.ShippingOptionType, int, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	add := func(value any, clause string) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if len(f.IDs) > 0 {
		add(f.IDs, "id=ANY($%d::text[])")
	}
	if len(f.Labels) > 0 {
		add(f.Labels, "label=ANY($%d::text[])")
	}
	if len(f.Codes) > 0 {
		add(f.Codes, "code=ANY($%d::text[])")
	}
	if len(f.Descriptions) > 0 {
		add(f.Descriptions, "description=ANY($%d::text[])")
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(label ILIKE $%d OR code ILIKE $%d OR description ILIKE $%d)", len(args), len(args), len(args)))
	}
	w := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM shipping_option_type WHERE "+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM shipping_option_type WHERE %s ORDER BY created_at DESC,id LIMIT $%d OFFSET $%d", shippingOptionTypeColumns, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.ShippingOptionType{}
	for rows.Next() {
		v, e := scanShippingOptionType(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetShippingOptionType(ctx context.Context, id string) (models.ShippingOptionType, error) {
	return scanShippingOptionType(r.db.QueryRow(ctx, "SELECT "+shippingOptionTypeColumns+" FROM shipping_option_type WHERE id=$1 AND deleted_at IS NULL", id))
}
func (r *Repository) CreateShippingOptionTypes(ctx context.Context, in []models.ShippingOptionTypeInput) ([]models.ShippingOptionType, error) {
	tx, e := r.db.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	out := make([]models.ShippingOptionType, 0, len(in))
	for _, v := range in {
		x, e := scanShippingOptionType(tx.QueryRow(ctx, "INSERT INTO shipping_option_type(id,label,description,code) VALUES($1,$2,$3,$4) RETURNING "+shippingOptionTypeColumns, entityid.New("sotype"), v.Label, v.Description, v.Code))
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return out, nil
}
func (r *Repository) UpdateShippingOptionType(ctx context.Context, id string, v models.ShippingOptionTypeUpdate) (models.ShippingOptionType, error) {
	tag, e := r.db.Exec(ctx, "UPDATE shipping_option_type SET label=COALESCE($2,label),description=COALESCE($3,description),code=COALESCE($4,code),updated_at=now() WHERE id=$1 AND deleted_at IS NULL", id, v.Label, v.Description, v.Code)
	if e != nil {
		return models.ShippingOptionType{}, e
	}
	if tag.RowsAffected() == 0 {
		return models.ShippingOptionType{}, pgx.ErrNoRows
	}
	return r.GetShippingOptionType(ctx, id)
}
func (r *Repository) SoftDeleteShippingOptionTypes(ctx context.Context, ids []string) error {
	_, e := r.db.Exec(ctx, "UPDATE shipping_option_type SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL", ids)
	return e
}
func (r *Repository) RestoreShippingOptionTypes(ctx context.Context, ids []string) error {
	_, e := r.db.Exec(ctx, "UPDATE shipping_option_type SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[])", ids)
	return e
}
func (r *Repository) HardDeleteShippingOptionTypes(ctx context.Context, ids []string) error {
	_, e := r.db.Exec(ctx, "DELETE FROM shipping_option_type WHERE id=ANY($1::text[])", ids)
	return e
}
func (r *Repository) ShippingOptionTypeInUse(ctx context.Context, id string) (bool, error) {
	var used bool
	e := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM shipping_option WHERE shipping_option_type_id=$1 AND deleted_at IS NULL)", id).Scan(&used)
	return used, e
}
