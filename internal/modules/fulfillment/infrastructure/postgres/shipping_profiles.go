package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const shippingProfileColumns = `id,name,type,metadata,created_at,updated_at,deleted_at`

func scanShippingProfile(row pgx.Row) (models.ShippingProfile, error) {
	var v models.ShippingProfile
	err := row.Scan(&v.ID, &v.Name, &v.Type, &v.Metadata, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}
func (r *Repository) ListShippingProfiles(ctx context.Context, f models.ShippingProfileFilter, limit, offset int) ([]models.ShippingProfile, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	if len(f.IDs) > 0 {
		args = append(args, f.IDs)
		where = append(where, fmt.Sprintf("id=ANY($%d::text[])", len(args)))
	}
	if len(f.Names) > 0 {
		args = append(args, f.Names)
		where = append(where, fmt.Sprintf("name=ANY($%d::text[])", len(args)))
	}
	if f.Type != "" {
		args = append(args, f.Type)
		where = append(where, fmt.Sprintf("type=$%d", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR type ILIKE $%d)", len(args), len(args)))
	}
	w := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM shipping_profile WHERE `+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s FROM shipping_profile WHERE %s ORDER BY created_at DESC,id LIMIT $%d OFFSET $%d`, shippingProfileColumns, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.ShippingProfile{}
	for rows.Next() {
		v, e := scanShippingProfile(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetShippingProfile(ctx context.Context, id string) (models.ShippingProfile, error) {
	return scanShippingProfile(r.db.QueryRow(ctx, `SELECT `+shippingProfileColumns+` FROM shipping_profile WHERE id=$1 AND deleted_at IS NULL`, id))
}
func (r *Repository) CreateShippingProfiles(ctx context.Context, in []models.ShippingProfileInput) ([]models.ShippingProfile, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]models.ShippingProfile, 0, len(in))
	for _, v := range in {
		created, e := scanShippingProfile(tx.QueryRow(ctx, `INSERT INTO shipping_profile(id,name,type,metadata) VALUES($1,$2,$3,$4) RETURNING `+shippingProfileColumns, entityid.New("sp"), v.Name, v.Type, metadata(v.Metadata)))
		if e != nil {
			return nil, e
		}
		out = append(out, created)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) UpdateShippingProfile(ctx context.Context, id string, fields map[string]any) (models.ShippingProfile, error) {
	cols := map[string]string{"name": "name", "type": "type", "metadata": "metadata"}
	sets := []string{}
	args := []any{id}
	for k, v := range fields {
		if col, ok := cols[k]; ok {
			args = append(args, v)
			sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := r.db.Exec(ctx, `UPDATE shipping_profile SET `+strings.Join(sets, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
		if err != nil {
			return models.ShippingProfile{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.ShippingProfile{}, pgx.ErrNoRows
		}
	}
	return r.GetShippingProfile(ctx, id)
}
func (r *Repository) SoftDeleteShippingProfiles(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE shipping_profile SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestoreShippingProfiles(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE shipping_profile SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) HardDeleteShippingProfiles(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM shipping_profile WHERE id=ANY($1::text[])`, ids)
	return err
}
