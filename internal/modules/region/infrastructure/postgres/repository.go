package postgres

import (
	"context"
	"encoding/json"
	"github.com/example/medusa-go-clone/internal/modules/region/domain"
	"github.com/example/medusa-go-clone/internal/modules/region/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, in models.RegionInput) (models.Region, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Region{}, err
	}
	defer tx.Rollback(ctx)
	id := entityid.New("reg")
	automatic := true
	if in.AutomaticTaxes != nil {
		automatic = *in.AutomaticTaxes
	}
	currency := ""
	if in.CurrencyCode != nil {
		currency = *in.CurrencyCode
	}
	name := ""
	if in.Name != nil {
		name = *in.Name
	}
	metadata := in.Metadata
	if metadata == nil {
		metadata = json.RawMessage("null")
	}
	var out models.Region
	err = tx.QueryRow(ctx, `INSERT INTO region(id,name,currency_code,automatic_taxes,metadata) VALUES($1,$2,$3,$4,$5) RETURNING id,name,currency_code,automatic_taxes,metadata,created_at,updated_at,deleted_at`, id, name, currency, automatic, metadata).Scan(&out.ID, &out.Name, &out.CurrencyCode, &out.AutomaticTaxes, &out.Metadata, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt)
	if err != nil {
		return out, err
	}
	if in.Countries != nil {
		if _, err = tx.Exec(ctx, `UPDATE region_country SET region_id=$1 WHERE iso_2=ANY($2)`, id, in.Countries); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return r.Get(ctx, id)
}

func (r *Repository) Get(ctx context.Context, id string) (models.Region, error) {
	var out models.Region
	err := r.db.QueryRow(ctx, `SELECT id,name,currency_code,automatic_taxes,metadata,created_at,updated_at,deleted_at FROM region WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&out.ID, &out.Name, &out.CurrencyCode, &out.AutomaticTaxes, &out.Metadata, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt)
	if err != nil {
		return out, err
	}
	out.Countries, err = r.regionCountries(ctx, id)
	return out, err
}
func (r *Repository) regionCountries(ctx context.Context, id string) ([]models.Country, error) {
	rows, err := r.db.Query(ctx, `SELECT iso_2,iso_3,num_code,name,display_name,region_id,metadata FROM region_country WHERE region_id=$1 ORDER BY iso_2`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Country{}
	for rows.Next() {
		var c models.Country
		if err := rows.Scan(&c.ISO2, &c.ISO3, &c.NumCode, &c.Name, &c.DisplayName, &c.RegionID, &c.Metadata); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *Repository) List(ctx context.Context) ([]models.Region, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM region WHERE deleted_at IS NULL ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Region{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		x, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id string, in models.RegionInput) (models.Region, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Region{}, err
	}
	defer tx.Rollback(ctx)
	if in.Countries != nil {
		if _, err = tx.Exec(ctx, `UPDATE region_country SET region_id=NULL WHERE region_id=$1`, id); err != nil {
			return models.Region{}, err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE region SET name=COALESCE($2,name),currency_code=COALESCE($3,currency_code),automatic_taxes=COALESCE($4,automatic_taxes),metadata=COALESCE($5,metadata),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, in.Name, in.CurrencyCode, in.AutomaticTaxes, in.Metadata)
	if err != nil {
		return models.Region{}, err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM region WHERE id=$1 AND deleted_at IS NULL)`, id).Scan(&exists); err != nil {
		return models.Region{}, err
	}
	if !exists {
		return models.Region{}, pgx.ErrNoRows
	}
	if in.Countries != nil {
		if _, err = tx.Exec(ctx, `UPDATE region_country SET region_id=$1 WHERE iso_2=ANY($2)`, id, in.Countries); err != nil {
			return models.Region{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Region{}, err
	}
	return r.Get(ctx, id)
}

func (r *Repository) SoftDelete(ctx context.Context, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE region SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) RestoreRegions(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `UPDATE region SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1)`, ids)
	return err
}

func (r *Repository) DeleteRegions(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM region WHERE id=ANY($1)`, ids)
	return err
}

func (r *Repository) Countries(ctx context.Context, codes []string, regionID string) ([]models.Country, error) {
	normalized := make([]string, len(codes))
	for i, c := range codes {
		normalized[i] = strings.ToLower(c)
	}
	rows, err := r.db.Query(ctx, `SELECT iso_2,iso_3,num_code,name,display_name,region_id,metadata FROM region_country WHERE iso_2=ANY($1)`, normalized)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Country{}
	for rows.Next() {
		var c models.Country
		if err := rows.Scan(&c.ISO2, &c.ISO3, &c.NumCode, &c.Name, &c.DisplayName, &c.RegionID, &c.Metadata); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var _ domain.Repository = (*Repository)(nil)
