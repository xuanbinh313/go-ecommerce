package postgres

import (
	"context"
	"github.com/example/medusa-go-clone/internal/modules/user/domain"
	"github.com/example/medusa-go-clone/internal/modules/user/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) CreateUser(ctx context.Context, in models.UserInput) (models.User, error) {
	id := entityid.New("user")
	if in.ID != nil {
		id = *in.ID
	}
	var out models.User
	err := r.db.QueryRow(ctx, `INSERT INTO "user"(id,first_name,last_name,email,avatar_url,metadata) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,first_name,last_name,email,avatar_url,metadata,created_at,updated_at,deleted_at`, id, in.FirstName, in.LastName, in.Email, in.AvatarURL, in.Metadata).Scan(&out.ID, &out.FirstName, &out.LastName, &out.Email, &out.AvatarURL, &out.Metadata, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt)
	return out, err
}
func (r *Repository) GetUser(ctx context.Context, id string) (models.User, error) {
	var out models.User
	err := r.db.QueryRow(ctx, `SELECT id,first_name,last_name,email,avatar_url,metadata,created_at,updated_at,deleted_at FROM "user" WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&out.ID, &out.FirstName, &out.LastName, &out.Email, &out.AvatarURL, &out.Metadata, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt)
	return out, err
}
func (r *Repository) ListUsers(ctx context.Context, ids []string) ([]models.User, error) {
	if len(ids) == 0 {
		return r.queryUsers(ctx, `SELECT id,first_name,last_name,email,avatar_url,metadata,created_at,updated_at,deleted_at FROM "user" WHERE deleted_at IS NULL ORDER BY created_at`)
	}
	return r.queryUsers(ctx, `SELECT id,first_name,last_name,email,avatar_url,metadata,created_at,updated_at,deleted_at FROM "user" WHERE id=ANY($1) AND deleted_at IS NULL ORDER BY created_at`, ids)
}
func (r *Repository) FindUsersByEmail(ctx context.Context, emails []string) ([]models.User, error) {
	return r.queryUsers(ctx, `SELECT id,first_name,last_name,email,avatar_url,metadata,created_at,updated_at,deleted_at FROM "user" WHERE email=ANY($1) AND deleted_at IS NULL`, emails)
}
func (r *Repository) queryUsers(ctx context.Context, query string, args ...any) ([]models.User, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.User{}
	for rows.Next() {
		var x models.User
		if err := rows.Scan(&x.ID, &x.FirstName, &x.LastName, &x.Email, &x.AvatarURL, &x.Metadata, &x.CreatedAt, &x.UpdatedAt, &x.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *Repository) UpdateUser(ctx context.Context, id string, in models.UserInput) (models.User, error) {
	tag, err := r.db.Exec(ctx, `UPDATE "user" SET first_name=CASE WHEN $7 THEN $2 ELSE first_name END,last_name=CASE WHEN $8 THEN $3 ELSE last_name END,email=COALESCE($4,email),avatar_url=CASE WHEN $9 THEN $5 ELSE avatar_url END,metadata=CASE WHEN $10 THEN $6 ELSE metadata END,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, in.FirstName, in.LastName, in.Email, in.AvatarURL, in.Metadata, in.FirstNameSet, in.LastNameSet, in.AvatarURLSet, in.MetadataSet)
	if err != nil {
		return models.User{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.User{}, pgx.ErrNoRows
	}
	return r.GetUser(ctx, id)
}
func (r *Repository) SoftDeleteUsers(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE "user" SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	return err
}

func (r *Repository) RestoreUsers(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE "user" SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1)`, ids)
	return err
}

func (r *Repository) DeleteUsers(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "user" WHERE id=ANY($1)`, ids)
	return err
}

func (r *Repository) CreateInvite(ctx context.Context, in models.Invite) (models.Invite, error) {
	var out models.Invite
	err := r.db.QueryRow(ctx, `INSERT INTO invite(id,email,accepted,token,expires_at,metadata) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,email,accepted,token,expires_at,metadata,created_at,updated_at,deleted_at`, in.ID, in.Email, in.Accepted, in.Token, in.ExpiresAt, in.Metadata).Scan(&out.ID, &out.Email, &out.Accepted, &out.Token, &out.ExpiresAt, &out.Metadata, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt)
	return out, err
}
func (r *Repository) GetInvite(ctx context.Context, id string) (models.Invite, error) {
	var out models.Invite
	err := r.db.QueryRow(ctx, `SELECT id,email,accepted,token,expires_at,metadata,created_at,updated_at,deleted_at FROM invite WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&out.ID, &out.Email, &out.Accepted, &out.Token, &out.ExpiresAt, &out.Metadata, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt)
	return out, err
}
func (r *Repository) ListInvites(ctx context.Context, ids []string) ([]models.Invite, error) {
	rows, err := r.db.Query(ctx, `SELECT id,email,accepted,token,expires_at,metadata,created_at,updated_at,deleted_at FROM invite WHERE id=ANY($1) AND deleted_at IS NULL ORDER BY created_at`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Invite{}
	for rows.Next() {
		var x models.Invite
		if err := rows.Scan(&x.ID, &x.Email, &x.Accepted, &x.Token, &x.ExpiresAt, &x.Metadata, &x.CreatedAt, &x.UpdatedAt, &x.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repository) ListInvitesPage(ctx context.Context, ids, emails []string, limit, offset int) ([]models.Invite, int, error) {
	where := ` WHERE deleted_at IS NULL AND (cardinality($1::text[])=0 OR id=ANY($1)) AND (cardinality($2::text[])=0 OR email=ANY($2))`
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM invite`+where, ids, emails).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,email,accepted,token,expires_at,metadata,created_at,updated_at,deleted_at FROM invite`+where+` ORDER BY created_at DESC LIMIT $3 OFFSET $4`, ids, emails, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.Invite{}
	for rows.Next() {
		var x models.Invite
		if err = rows.Scan(&x.ID, &x.Email, &x.Accepted, &x.Token, &x.ExpiresAt, &x.Metadata, &x.CreatedAt, &x.UpdatedAt, &x.DeletedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, x)
	}
	return out, count, rows.Err()
}

func (r *Repository) SoftDeleteInvites(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE invite SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	return err
}
func (r *Repository) RestoreInvites(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE invite SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1)`, ids)
	return err
}
func (r *Repository) DeleteInvites(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM invite WHERE id=ANY($1)`, ids)
	return err
}
func (r *Repository) UpdateInvite(ctx context.Context, in models.Invite) (models.Invite, error) {
	tag, err := r.db.Exec(ctx, `UPDATE invite SET token=$2,expires_at=$3,accepted=$4,metadata=COALESCE($5,metadata),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, in.ID, in.Token, in.ExpiresAt, in.Accepted, in.Metadata)
	if err != nil {
		return models.Invite{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Invite{}, pgx.ErrNoRows
	}
	return r.GetInvite(ctx, in.ID)
}

var _ domain.Repository = (*Repository)(nil)
