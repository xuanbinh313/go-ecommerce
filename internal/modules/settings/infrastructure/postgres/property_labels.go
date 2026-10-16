package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/settings/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const propertyLabelColumns = `id,entity,property,label,description,created_at,updated_at`

func scanPropertyLabel(row pgx.Row) (models.PropertyLabel, error) {
	var v models.PropertyLabel
	err := row.Scan(&v.ID, &v.Entity, &v.Property, &v.Label, &v.Description, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

func (r *Repository) ListPropertyLabels(ctx context.Context, f models.PropertyLabelFilter, limit, offset int) ([]models.PropertyLabel, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	for col, val := range map[string]string{"entity": f.Entity, "property": f.Property} {
		if val != "" {
			args = append(args, val)
			where = append(where, fmt.Sprintf("%s=$%d", col, len(args)))
		}
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(label ILIKE $%d OR entity ILIKE $%d OR property ILIKE $%d)", n, n, n))
	}
	w := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM property_label WHERE `+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s FROM property_label WHERE %s ORDER BY entity,property LIMIT $%d OFFSET $%d`, propertyLabelColumns, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.PropertyLabel{}
	for rows.Next() {
		v, e := scanPropertyLabel(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetPropertyLabel(ctx context.Context, id string) (models.PropertyLabel, error) {
	return scanPropertyLabel(r.db.QueryRow(ctx, `SELECT `+propertyLabelColumns+` FROM property_label WHERE id=$1 AND deleted_at IS NULL`, id))
}
func (r *Repository) CreatePropertyLabels(ctx context.Context, inputs []models.PropertyLabelInput) ([]models.PropertyLabel, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]models.PropertyLabel, 0, len(inputs))
	for _, in := range inputs {
		v, e := scanPropertyLabel(tx.QueryRow(ctx, `INSERT INTO property_label(id,entity,property,label,description) VALUES($1,$2,$3,$4,$5) RETURNING `+propertyLabelColumns, entityid.New("plbl"), in.Entity, in.Property, in.Label, in.Description))
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func (r *Repository) UpdatePropertyLabel(ctx context.Context, id string, fields map[string]any) (models.PropertyLabel, error) {
	sets := []string{}
	args := []any{id}
	for k, v := range fields {
		col := ""
		if k == "label" {
			col = "label"
		}
		if k == "description" {
			col = "description"
		}
		if col != "" {
			args = append(args, v)
			sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, e := r.db.Exec(ctx, `UPDATE property_label SET `+strings.Join(sets, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
		if e != nil {
			return models.PropertyLabel{}, e
		}
		if tag.RowsAffected() == 0 {
			return models.PropertyLabel{}, pgx.ErrNoRows
		}
	}
	return r.GetPropertyLabel(ctx, id)
}
func (r *Repository) DeletePropertyLabels(ctx context.Context, ids []string) ([]models.PropertyLabel, error) {
	tx, e := r.db.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	rows, e := tx.Query(ctx, `SELECT `+propertyLabelColumns+` FROM property_label WHERE id=ANY($1::text[]) AND deleted_at IS NULL FOR UPDATE`, ids)
	if e != nil {
		return nil, e
	}
	out := []models.PropertyLabel{}
	for rows.Next() {
		v, x := scanPropertyLabel(rows)
		if x != nil {
			rows.Close()
			return nil, x
		}
		out = append(out, v)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return nil, e
	}
	rows.Close()
	if _, e = tx.Exec(ctx, `UPDATE property_label SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids); e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return out, nil
}
