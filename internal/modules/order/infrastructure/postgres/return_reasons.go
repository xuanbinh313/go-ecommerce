package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const returnReasonColumns = `id,value,label,description,metadata,parent_return_reason_id,created_at,updated_at,deleted_at`

func scanReturnReason(row pgx.Row) (models.ReturnReason, error) {
	var v models.ReturnReason
	err := row.Scan(&v.ID, &v.Value, &v.Label, &v.Description, &v.Metadata, &v.ParentReturnReasonID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}

func (r *Repository) ListReturnReasons(ctx context.Context, f models.ReturnReasonFilter, limit, offset int) ([]models.ReturnReason, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	add := func(column string, values []string) {
		if len(values) > 0 {
			args = append(args, values)
			where = append(where, fmt.Sprintf("%s=ANY($%d::text[])", column, len(args)))
		}
	}
	add("id", f.IDs)
	add("value", f.Values)
	add("label", f.Labels)
	add("description", f.Descriptions)
	add("parent_return_reason_id", f.ParentIDs)
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(value ILIKE $%d OR label ILIKE $%d OR COALESCE(description,'') ILIKE $%d)", n, n, n))
	}
	w := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM return_reason WHERE `+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s FROM return_reason WHERE %s ORDER BY created_at,id LIMIT $%d OFFSET $%d`, returnReasonColumns, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.ReturnReason{}
	for rows.Next() {
		v, e := scanReturnReason(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetReturnReason(ctx context.Context, id string) (models.ReturnReason, error) {
	v, err := scanReturnReason(r.db.QueryRow(ctx, `SELECT `+returnReasonColumns+` FROM return_reason WHERE id=$1 AND deleted_at IS NULL`, id))
	if err != nil {
		return v, err
	}
	if v.ParentReturnReasonID != nil {
		p, e := scanReturnReason(r.db.QueryRow(ctx, `SELECT `+returnReasonColumns+` FROM return_reason WHERE id=$1 AND deleted_at IS NULL`, *v.ParentReturnReasonID))
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return v, e
		}
		if e == nil {
			v.ParentReturnReason = &p
		}
	}
	rows, e := r.db.Query(ctx, `SELECT `+returnReasonColumns+` FROM return_reason WHERE parent_return_reason_id=$1 AND deleted_at IS NULL ORDER BY created_at,id`, v.ID)
	if e != nil {
		return v, e
	}
	defer rows.Close()
	v.ReturnReasonChildren = []models.ReturnReason{}
	for rows.Next() {
		c, x := scanReturnReason(rows)
		if x != nil {
			return v, x
		}
		v.ReturnReasonChildren = append(v.ReturnReasonChildren, c)
	}
	return v, rows.Err()
}
func (r *Repository) CreateReturnReasons(ctx context.Context, in []models.ReturnReasonInput) ([]models.ReturnReason, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]models.ReturnReason, 0, len(in))
	for _, v := range in {
		var metadata any
		if v.Metadata != nil {
			if !json.Valid(v.Metadata) {
				return nil, fmt.Errorf("invalid metadata json")
			}
			metadata = v.Metadata
		}
		created, e := scanReturnReason(tx.QueryRow(ctx, `INSERT INTO return_reason(id,value,label,description,metadata,parent_return_reason_id) VALUES($1,$2,$3,$4,$5,$6) RETURNING `+returnReasonColumns, entityid.New("rr"), v.Value, v.Label, v.Description, metadata, v.ParentReturnReasonID))
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
func (r *Repository) UpdateReturnReasons(ctx context.Context, ids []string, fields map[string]any) ([]models.ReturnReason, error) {
	allowed := map[string]string{"value": "value", "label": "label", "description": "description", "metadata": "metadata"}
	sets := []string{}
	args := []any{ids}
	for k, v := range fields {
		if col, ok := allowed[k]; ok {
			args = append(args, v)
			sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		_, err := r.db.Exec(ctx, `UPDATE return_reason SET `+strings.Join(sets, ",")+` WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, args...)
		if err != nil {
			return nil, err
		}
	}
	out := make([]models.ReturnReason, 0, len(ids))
	for _, id := range ids {
		v, err := r.GetReturnReason(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
func (r *Repository) SoftDeleteReturnReasons(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE return_reason SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestoreReturnReasons(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE return_reason SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) HardDeleteReturnReasons(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM return_reason WHERE id=ANY($1::text[])`, ids)
	return err
}
