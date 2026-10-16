package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/payment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const refundReasonColumns = `id,label,code,description,metadata,created_at,updated_at,deleted_at`

func scanRefundReason(row pgx.Row) (models.RefundReason, error) {
	var v models.RefundReason
	err := row.Scan(&v.ID, &v.Label, &v.Code, &v.Description, &v.Metadata, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return v, err
}
func (r *Repository) ListRefundReasons(ctx context.Context, f models.RefundReasonFilter) ([]models.RefundReason, int64, error) {
	where := []string{}
	args := []any{}
	if len(f.DeletedAt) == 0 {
		where = append(where, "deleted_at IS NULL")
	}
	if len(f.IDs) > 0 {
		args = append(args, f.IDs)
		where = append(where, fmt.Sprintf("id=ANY($%d::text[])", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(label ILIKE $%d OR code ILIKE $%d OR COALESCE(description,'') ILIKE $%d)", n, n, n))
	}
	for _, x := range []struct {
		name string
		ops  map[string]string
	}{{"created_at", f.CreatedAt}, {"updated_at", f.UpdatedAt}, {"deleted_at", f.DeletedAt}} {
		if err := appendDateConditions(&where, &args, x.name, x.ops); err != nil {
			return nil, 0, err
		}
	}
	w := strings.Join(where, " AND ")
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM refund_reason WHERE `+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit < 1 {
		limit = 15
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s FROM refund_reason WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, refundReasonColumns, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.RefundReason{}
	for rows.Next() {
		v, e := scanRefundReason(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetRefundReason(ctx context.Context, id string) (*models.RefundReason, error) {
	v, err := scanRefundReason(r.db.QueryRow(ctx, `SELECT `+refundReasonColumns+` FROM refund_reason WHERE id=$1 AND deleted_at IS NULL`, id))
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func (r *Repository) CreateRefundReasons(ctx context.Context, in []models.RefundReasonInput) ([]models.RefundReason, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := make([]models.RefundReason, 0, len(in))
	for _, v := range in {
		created, e := scanRefundReason(tx.QueryRow(ctx, `INSERT INTO refund_reason(id,label,code,description) VALUES($1,$2,$3,$4) RETURNING `+refundReasonColumns, entityid.New("refr"), v.Label, v.Code, v.Description))
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
func (r *Repository) UpdateRefundReasons(ctx context.Context, in []models.RefundReason) ([]models.RefundReason, error) {
	out := make([]models.RefundReason, 0, len(in))
	for _, v := range in {
		_, err := r.db.Exec(ctx, `UPDATE refund_reason SET label=$2,code=$3,description=$4,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, v.ID, v.Label, v.Code, v.Description)
		if err != nil {
			return nil, err
		}
		updated, err := r.GetRefundReason(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *updated)
	}
	return out, nil
}
func (r *Repository) SoftDeleteRefundReasons(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE refund_reason SET deleted_at=now(),updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestoreRefundReasons(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE refund_reason SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) DeleteRefundReasons(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refund_reason WHERE id=ANY($1::text[])`, ids)
	return err
}
