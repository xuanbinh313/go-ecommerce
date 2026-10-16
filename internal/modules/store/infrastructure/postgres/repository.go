package postgres

import (
	"context"

	"github.com/example/medusa-go-clone/internal/modules/store/domain"
	"github.com/example/medusa-go-clone/internal/modules/store/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, in models.StoreInput) (models.Store, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Store{}, err
	}
	defer tx.Rollback(ctx)
	name := "Medusa Store"
	if in.Name != nil {
		name = *in.Name
	}
	store := models.Store{ID: entityid.New("store"), Name: name, Metadata: in.Metadata, DefaultSalesChannelID: in.DefaultSalesChannelID, DefaultRegionID: in.DefaultRegionID, DefaultLocationID: in.DefaultLocationID}
	err = tx.QueryRow(ctx, `INSERT INTO store(id,name,default_sales_channel_id,default_region_id,default_location_id,metadata) VALUES($1,$2,$3,$4,$5,$6) RETURNING created_at,updated_at`, store.ID, store.Name, store.DefaultSalesChannelID, store.DefaultRegionID, store.DefaultLocationID, store.Metadata).Scan(&store.CreatedAt, &store.UpdatedAt)
	if err != nil {
		return models.Store{}, err
	}
	if in.SupportedCurrencies != nil {
		if err := replaceCurrencies(ctx, tx, store.ID, in.SupportedCurrencies); err != nil {
			return models.Store{}, err
		}
	}
	if in.SupportedLocales != nil {
		if err := replaceLocales(ctx, tx, store.ID, in.SupportedLocales); err != nil {
			return models.Store{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Store{}, err
	}
	return r.Get(ctx, store.ID)
}

func (r *Repository) Get(ctx context.Context, id string) (models.Store, error) {
	var s models.Store
	err := r.db.QueryRow(ctx, `SELECT id,name,default_sales_channel_id,default_region_id,default_location_id,metadata,created_at,updated_at,deleted_at FROM store WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&s.ID, &s.Name, &s.DefaultSalesChannelID, &s.DefaultRegionID, &s.DefaultLocationID, &s.Metadata, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
	if err != nil {
		return s, err
	}
	s.SupportedCurrencies = []models.StoreCurrency{}
	rows, err := r.db.Query(ctx, `SELECT id,store_id,currency_code,is_default FROM store_currency WHERE store_id=$1 ORDER BY created_at`, id)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var x models.StoreCurrency
		if err := rows.Scan(&x.ID, &x.StoreID, &x.CurrencyCode, &x.IsDefault); err != nil {
			rows.Close()
			return s, err
		}
		s.SupportedCurrencies = append(s.SupportedCurrencies, x)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return s, err
	}
	rows.Close()
	s.SupportedLocales = []models.StoreLocale{}
	locales, err := r.db.Query(ctx, `SELECT id,store_id,locale_code FROM store_locale WHERE store_id=$1 ORDER BY created_at`, id)
	if err != nil {
		return s, err
	}
	defer locales.Close()
	for locales.Next() {
		var x models.StoreLocale
		if err := locales.Scan(&x.ID, &x.StoreID, &x.LocaleCode); err != nil {
			return s, err
		}
		s.SupportedLocales = append(s.SupportedLocales, x)
	}
	return s, locales.Err()
}

func (r *Repository) List(ctx context.Context) ([]models.Store, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM store WHERE deleted_at IS NULL ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Store{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		store, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, store)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id string, in models.StoreInput) (models.Store, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Store{}, err
	}
	defer tx.Rollback(ctx)
	if in.Name != nil {
		_, err = tx.Exec(ctx, `UPDATE store SET name=$2,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, *in.Name)
	}
	if err == nil {
		_, err = tx.Exec(ctx, `UPDATE store SET default_sales_channel_id=COALESCE($2,default_sales_channel_id),default_region_id=COALESCE($3,default_region_id),default_location_id=COALESCE($4,default_location_id),metadata=COALESCE($5,metadata),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, in.DefaultSalesChannelID, in.DefaultRegionID, in.DefaultLocationID, in.Metadata)
	}
	if err != nil {
		return models.Store{}, err
	}
	if in.SupportedCurrencies != nil {
		if err = replaceCurrencies(ctx, tx, id, in.SupportedCurrencies); err != nil {
			return models.Store{}, err
		}
	}
	if in.SupportedLocales != nil {
		if err = replaceLocales(ctx, tx, id, in.SupportedLocales); err != nil {
			return models.Store{}, err
		}
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM store WHERE id=$1 AND deleted_at IS NULL)`, id).Scan(&exists); err != nil {
		return models.Store{}, err
	}
	if !exists {
		return models.Store{}, pgx.ErrNoRows
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Store{}, err
	}
	return r.Get(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE store SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func (r *Repository) DeleteStores(ctx context.Context, ids []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM store WHERE id=ANY($1)`, ids)
	return err
}

func replaceCurrencies(ctx context.Context, tx pgx.Tx, storeID string, items []models.StoreCurrencyInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM store_currency WHERE store_id=$1`, storeID); err != nil {
		return err
	}
	for _, x := range items {
		if _, err := tx.Exec(ctx, `INSERT INTO store_currency(id,store_id,currency_code,is_default) VALUES($1,$2,$3,$4)`, entityid.New("stocur"), storeID, x.CurrencyCode, x.IsDefault); err != nil {
			return err
		}
	}
	return nil
}
func replaceLocales(ctx context.Context, tx pgx.Tx, storeID string, items []models.StoreLocaleInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM store_locale WHERE store_id=$1`, storeID); err != nil {
		return err
	}
	for _, x := range items {
		if _, err := tx.Exec(ctx, `INSERT INTO store_locale(id,store_id,locale_code) VALUES($1,$2,$3)`, entityid.New("stloc"), storeID, x.LocaleCode); err != nil {
			return err
		}
	}
	return nil
}

var _ domain.Repository = (*Repository)(nil)
