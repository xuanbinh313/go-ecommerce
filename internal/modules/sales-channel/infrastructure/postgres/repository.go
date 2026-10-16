package postgres

import (
	"context"
	"encoding/json"
	"github.com/example/medusa-go-clone/internal/modules/sales-channel/domain"
	"github.com/example/medusa-go-clone/internal/modules/sales-channel/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func (r *Repository) Create(ctx context.Context, in models.Input) (models.SalesChannel, error) {
	var out models.SalesChannel
	id := entityid.New("sc")
	description := in.Description
	isDisabled := false
	if in.IsDisabled != nil {
		isDisabled = *in.IsDisabled
	}
	metadata := in.Metadata
	if metadata == nil {
		metadata = json.RawMessage("null")
	}
	name := ""
	if in.Name != nil {
		name = *in.Name
	}
	err := r.db.QueryRow(ctx, `INSERT INTO sales_channel(id,name,description,is_disabled,metadata) VALUES($1,$2,$3,$4,$5) RETURNING id,name,description,is_disabled,metadata,created_at,updated_at`, id, name, description, isDisabled, metadata).Scan(&out.ID, &out.Name, &out.Description, &out.IsDisabled, &out.Metadata, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}
func (r *Repository) Get(ctx context.Context, id string) (models.SalesChannel, error) {
	var out models.SalesChannel
	err := r.db.QueryRow(ctx, `SELECT id,name,description,is_disabled,metadata,created_at,updated_at,deleted_at FROM sales_channel WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&out.ID, &out.Name, &out.Description, &out.IsDisabled, &out.Metadata, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt)
	return out, err
}
func (r *Repository) List(ctx context.Context) ([]models.SalesChannel, error) {
	rows, err := r.db.Query(ctx, `SELECT id,name,description,is_disabled,metadata,created_at,updated_at,deleted_at FROM sales_channel WHERE deleted_at IS NULL ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.SalesChannel{}
	for rows.Next() {
		var x models.SalesChannel
		if err := rows.Scan(&x.ID, &x.Name, &x.Description, &x.IsDisabled, &x.Metadata, &x.CreatedAt, &x.UpdatedAt, &x.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) Update(ctx context.Context, id string, in models.Input) (models.SalesChannel, error) {
	tag, err := r.db.Exec(ctx, `UPDATE sales_channel SET name=COALESCE($2,name),description=CASE WHEN $6 THEN $3 ELSE description END,is_disabled=COALESCE($4,is_disabled),metadata=CASE WHEN $7 THEN $5 ELSE metadata END,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, in.Name, in.Description, in.IsDisabled, in.Metadata, in.DescriptionSet, in.MetadataSet)
	if err != nil {
		return models.SalesChannel{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.SalesChannel{}, pgx.ErrNoRows
	}
	return r.Get(ctx, id)
}
func (r *Repository) Restore(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE sales_channel SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[]) AND deleted_at IS NOT NULL`, ids)
	return err
}
func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE sales_channel SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func (r *Repository) DeleteMany(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sales_channel WHERE id=ANY($1)`, ids)
	return err
}

var _ domain.Repository = (*Repository)(nil)
