package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
)

func (r *Repository) ListLocales(ctx context.Context, f models.LocaleFilter, limit, offset int) ([]models.Locale, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	if len(f.Codes) > 0 {
		args = append(args, f.Codes)
		where = append(where, fmt.Sprintf("code=ANY($%d::text[])", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", n, n))
	}
	w := strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM locale WHERE `+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	if limit < 1 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT id,code,name,created_at,updated_at FROM locale WHERE %s ORDER BY code LIMIT $%d OFFSET $%d`, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.Locale{}
	for rows.Next() {
		var v models.Locale
		if err = rows.Scan(&v.ID, &v.Code, &v.Name, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) GetLocaleByCode(ctx context.Context, code string) (models.Locale, error) {
	var v models.Locale
	err := r.db.QueryRow(ctx, `SELECT id,code,name,created_at,updated_at FROM locale WHERE code=$1 AND deleted_at IS NULL`, code).Scan(&v.ID, &v.Code, &v.Name, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
