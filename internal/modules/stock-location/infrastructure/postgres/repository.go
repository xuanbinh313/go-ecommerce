package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/stock-location/domain"
	"github.com/example/medusa-go-clone/internal/modules/stock-location/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func (r *Repository) List(ctx context.Context, f models.Filter) ([]models.StockLocation, int, error) {
	where := []string{"sl.deleted_at IS NULL"}
	args := []any{}
	add := func(col string, v []string) {
		if len(v) > 0 {
			args = append(args, v)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", col, len(args)))
		}
	}
	add("sl.id", f.IDs)
	add("sl.name", f.Names)
	add("sl.address_id", f.AddressIDs)
	if len(f.SalesChannelIDs) > 0 {
		args = append(args, f.SalesChannelIDs)
		where = append(where, fmt.Sprintf("EXISTS(SELECT 1 FROM sales_channel_stock_location x WHERE x.stock_location_id=sl.id AND x.deleted_at IS NULL AND x.sales_channel_id=ANY($%d))", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("sl.name ILIKE $%d", len(args)))
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM stock_location sl`+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT sl.id,sl.name,sl.address_id,sl.metadata,sl.created_at,sl.updated_at FROM stock_location sl%s ORDER BY sl.created_at DESC LIMIT $%d OFFSET $%d`, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.StockLocation{}
	for rows.Next() {
		var v models.StockLocation
		if err := rows.Scan(&v.ID, &v.Name, &v.AddressID, &v.Metadata, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	for i := range out {
		full, e := r.Get(ctx, out[i].ID)
		if e != nil {
			return nil, 0, e
		}
		out[i] = full
	}
	return out, count, nil
}
func (r *Repository) Get(ctx context.Context, id string) (models.StockLocation, error) {
	var v models.StockLocation
	var addressID *string
	err := r.db.QueryRow(ctx, `SELECT id,name,address_id,metadata,created_at,updated_at,deleted_at FROM stock_location WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&v.ID, &v.Name, &addressID, &v.Metadata, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	if err != nil {
		return v, err
	}
	v.AddressID = addressID
	if addressID != nil {
		a := models.Address{}
		err = r.db.QueryRow(ctx, `SELECT id,address_1,address_2,company,city,country_code,phone,province,postal_code,metadata FROM stock_location_address WHERE id=$1 AND deleted_at IS NULL`, *addressID).Scan(&a.ID, &a.Address1, &a.Address2, &a.Company, &a.City, &a.CountryCode, &a.Phone, &a.Province, &a.PostalCode, &a.Metadata)
		if err != nil {
			return v, err
		}
		v.Address = &a
	}
	rows, err := r.db.Query(ctx, `SELECT sc.id,sc.name FROM sales_channel_stock_location l JOIN sales_channel sc ON sc.id=l.sales_channel_id WHERE l.stock_location_id=$1 AND l.deleted_at IS NULL AND sc.deleted_at IS NULL ORDER BY sc.created_at`, id)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	v.SalesChannels = []map[string]any{}
	for rows.Next() {
		var scID, scName string
		if err = rows.Scan(&scID, &scName); err != nil {
			return v, err
		}
		v.SalesChannels = append(v.SalesChannels, map[string]any{"id": scID, "name": scName})
	}
	return v, rows.Err()
}
func (r *Repository) Create(ctx context.Context, in models.Input) (models.StockLocation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.StockLocation{}, err
	}
	defer tx.Rollback(ctx)
	id := entityid.New("sloc")
	var addressID any
	if in.AddressID != nil {
		addressID = *in.AddressID
	} else if in.Address != nil {
		addressID, err = insertAddress(ctx, tx, *in.Address)
		if err != nil {
			return models.StockLocation{}, err
		}
	}
	metadata := in.Metadata
	if metadata == nil {
		metadata = json.RawMessage("null")
	}
	_, err = tx.Exec(ctx, `INSERT INTO stock_location(id,name,address_id,metadata) VALUES($1,$2,$3,$4)`, id, in.Name, addressID, metadata)
	if err != nil {
		return models.StockLocation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.StockLocation{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repository) Update(ctx context.Context, id string, in models.UpdateInput) (models.StockLocation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.StockLocation{}, err
	}
	defer tx.Rollback(ctx)
	sets := []string{}
	args := []any{id}
	if in.Name != nil {
		args = append(args, *in.Name)
		sets = append(sets, fmt.Sprintf("name=$%d", len(args)))
	}
	if in.Metadata != nil {
		args = append(args, in.Metadata)
		sets = append(sets, fmt.Sprintf("metadata=$%d", len(args)))
	}
	if in.AddressID != nil {
		args = append(args, *in.AddressID)
		sets = append(sets, fmt.Sprintf("address_id=$%d", len(args)))
	} else if in.Address != nil {
		var current *string
		err = tx.QueryRow(ctx, `SELECT address_id FROM stock_location WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&current)
		if err != nil {
			return models.StockLocation{}, err
		}
		if current != nil {
			a := in.Address
			_, err = tx.Exec(ctx, `UPDATE stock_location_address SET address_1=$2,address_2=$3,company=$4,city=$5,country_code=$6,phone=$7,province=$8,postal_code=$9,metadata=$10,updated_at=now() WHERE id=$1`, *current, a.Address1, a.Address2, a.Company, a.City, a.CountryCode, a.Phone, a.Province, a.PostalCode, jsonOrNull(a.Metadata))
			if err != nil {
				return models.StockLocation{}, err
			}
		} else {
			newID, e := insertAddress(ctx, tx, *in.Address)
			if e != nil {
				return models.StockLocation{}, e
			}
			args = append(args, newID)
			sets = append(sets, fmt.Sprintf("address_id=$%d", len(args)))
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, e := tx.Exec(ctx, `UPDATE stock_location SET `+strings.Join(sets, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
		if e != nil {
			return models.StockLocation{}, e
		}
		if tag.RowsAffected() == 0 {
			return models.StockLocation{}, pgx.ErrNoRows
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.StockLocation{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE stock_location SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, `UPDATE sales_channel_stock_location SET deleted_at=now(),updated_at=now() WHERE stock_location_id=$1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) LinkSalesChannels(ctx context.Context, id string, add, remove []string) error {
	if len(add) > 0 {
		var count int
		if err := r.db.QueryRow(ctx, `SELECT count(DISTINCT id) FROM sales_channel WHERE id=ANY($1) AND deleted_at IS NULL`, add).Scan(&count); err != nil {
			return err
		}
		if count != len(unique(add)) {
			return fmt.Errorf("one or more sales channels do not exist")
		}
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if len(remove) > 0 {
		if _, err = tx.Exec(ctx, `UPDATE sales_channel_stock_location SET deleted_at=now(),updated_at=now() WHERE stock_location_id=$1 AND sales_channel_id=ANY($2) AND deleted_at IS NULL`, id, remove); err != nil {
			return err
		}
	}
	for _, sc := range unique(add) {
		if _, err = tx.Exec(ctx, `INSERT INTO sales_channel_stock_location(id,sales_channel_id,stock_location_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, entityid.New("scloc"), sc, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func insertAddress(ctx context.Context, tx pgx.Tx, a models.AddressInput) (string, error) {
	id := entityid.New("laddr")
	_, err := tx.Exec(ctx, `INSERT INTO stock_location_address(id,address_1,address_2,company,city,country_code,phone,province,postal_code,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, a.Address1, a.Address2, a.Company, a.City, a.CountryCode, a.Phone, a.Province, a.PostalCode, jsonOrNull(a.Metadata))
	return id, err
}
func jsonOrNull(v json.RawMessage) json.RawMessage {
	if v == nil {
		return json.RawMessage("null")
	}
	return v
}
func unique(v []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range v {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

var _ domain.Repository = (*Repository)(nil)
