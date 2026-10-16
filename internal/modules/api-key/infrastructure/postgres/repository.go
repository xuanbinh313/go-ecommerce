package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/api-key/domain"
	"github.com/example/medusa-go-clone/internal/modules/api-key/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const keyColumns = `id,title,token,salt,redacted,type,last_used_at,created_by,revoked_by,revoked_at,created_at,updated_at`

func scanKey(row pgx.Row) (models.APIKey, error) {
	var v models.APIKey
	err := row.Scan(&v.ID, &v.Title, &v.Token, &v.Salt, &v.Redacted, &v.Type, &v.LastUsedAt, &v.CreatedBy, &v.RevokedBy, &v.RevokedAt, &v.CreatedAt, &v.UpdatedAt)
	if v.Type == "secret" {
		v.Token = ""
	}
	return v, err
}
func (r *Repository) List(ctx context.Context, f models.Filter) ([]models.APIKey, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	add := func(col string, values []string) {
		if len(values) > 0 {
			args = append(args, values)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", col, len(args)))
		}
	}
	add("id", f.IDs)
	add("title", f.Titles)
	add("type", f.Types)
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR redacted ILIKE $%d)", len(args), len(args)))
	}
	w := ` WHERE ` + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM api_key`+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	q := fmt.Sprintf(`SELECT %s FROM api_key%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, keyColumns, w, len(args)-1, len(args))
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.APIKey{}
	for rows.Next() {
		v, e := scanKey(rows)
		if e != nil {
			return nil, 0, e
		}
		items = append(items, v)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	for i := range items {
		full, e := r.Get(ctx, items[i].ID)
		if e != nil {
			return nil, 0, e
		}
		items[i].SalesChannels = full.SalesChannels
	}
	return items, count, nil
}
func (r *Repository) Get(ctx context.Context, id string) (models.APIKey, error) {
	v, err := scanKey(r.db.QueryRow(ctx, `SELECT `+keyColumns+` FROM api_key WHERE id=$1 AND deleted_at IS NULL`, id))
	if err != nil {
		return v, err
	}
	rows, e := r.db.Query(ctx, `SELECT sc.id,sc.name FROM publishable_api_key_sales_channel l JOIN sales_channel sc ON sc.id=l.sales_channel_id WHERE l.publishable_key_id=$1 AND l.deleted_at IS NULL AND sc.deleted_at IS NULL ORDER BY sc.created_at`, id)
	if e != nil {
		return v, e
	}
	defer rows.Close()
	v.SalesChannels = []models.SalesChannel{}
	for rows.Next() {
		var sc models.SalesChannel
		if e = rows.Scan(&sc.ID, &sc.Name); e != nil {
			return v, e
		}
		v.SalesChannels = append(v.SalesChannels, sc)
	}
	return v, rows.Err()
}

func (r *Repository) GetPublishableKeyByToken(ctx context.Context, token string) (models.APIKey, error) {
	var id string
	err := r.db.QueryRow(ctx, `SELECT id FROM api_key WHERE token=$1 AND type='publishable' AND deleted_at IS NULL AND (revoked_at IS NULL OR revoked_at > now())`, token).Scan(&id)
	if err == pgx.ErrNoRows {
		return models.APIKey{}, nil
	}
	if err != nil {
		return models.APIKey{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repository) Create(ctx context.Context, v models.APIKey) (models.APIKey, error) {
	return scanKey(r.db.QueryRow(ctx, `INSERT INTO api_key(id,title,token,salt,redacted,type,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+keyColumns, v.ID, v.Title, v.Token, v.Salt, v.Redacted, v.Type, v.CreatedBy))
}
func (r *Repository) Update(ctx context.Context, id string, in models.UpdateInput) error {
	tag, err := r.db.Exec(ctx, `UPDATE api_key SET title=$2,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, in.Title)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) Revoke(ctx context.Context, id string, in models.RevokeInput) error {
	tag, err := r.db.Exec(ctx, `UPDATE api_key SET revoked_at=$2,revoked_by=$3,updated_at=now() WHERE id=$1 AND deleted_at IS NULL AND (revoked_at IS NULL OR revoked_at>now())`, id, revocationTime(in), in.RevokedBy)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE api_key SET deleted_at=now(),updated_at=now() WHERE id=$1 AND revoked_at IS NOT NULL AND revoked_at<=now() AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("API key must be revoked before deletion")
	}
	if _, err = tx.Exec(ctx, `DELETE FROM publishable_api_key_sales_channel WHERE publishable_key_id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) DeleteSalesChannelLinks(ctx context.Context, id string, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE publishable_api_key_sales_channel SET deleted_at=now(),updated_at=now() WHERE publishable_key_id=$1 AND sales_channel_id=ANY($2) AND deleted_at IS NULL`, id, ids)
	return err
}
func (r *Repository) CreateSalesChannelLinks(ctx context.Context, id string, ids []string) error {
	for _, sc := range ids {
		_, err := r.db.Exec(ctx, `INSERT INTO publishable_api_key_sales_channel(id,publishable_key_id,sales_channel_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, entityid.New("pksc"), id, sc)
		if err != nil {
			return err
		}
	}
	return nil
}
func (r *Repository) SalesChannelsExist(ctx context.Context, ids []string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT count(DISTINCT id) FROM sales_channel WHERE id=ANY($1) AND deleted_at IS NULL`, ids).Scan(&count)
	return count == len(unique(ids)), err
}
func unique(ids []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
func revocationTime(in models.RevokeInput) time.Time {
	at := time.Now()
	if in.RevokeIn != nil {
		at = at.Add(time.Duration(*in.RevokeIn) * time.Second)
	}
	return at
}

var _ domain.Repository = (*Repository)(nil)
