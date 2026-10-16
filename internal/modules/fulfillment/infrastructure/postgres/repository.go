package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain"
	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func metadata(v json.RawMessage) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage("null")
	}
	return v
}
func (r *Repository) CreateSet(ctx context.Context, in models.CreateSetInput) (models.FulfillmentSet, error) {
	id := entityid.New("fuset")
	_, err := r.db.Exec(ctx, "INSERT INTO fulfillment_set(id,name,type,metadata) VALUES($1,$2,$3,$4)", id, in.Name, in.Type, metadata(in.Metadata))
	if err != nil {
		return models.FulfillmentSet{}, err
	}
	return r.GetSet(ctx, id)
}
func (r *Repository) GetSet(ctx context.Context, id string) (models.FulfillmentSet, error) {
	var v models.FulfillmentSet
	err := r.db.QueryRow(ctx, "SELECT id,name,type,metadata FROM fulfillment_set WHERE id=$1 AND deleted_at IS NULL", id).Scan(&v.ID, &v.Name, &v.Type, &v.Metadata)
	if err != nil {
		return v, err
	}
	rows, err := r.db.Query(ctx, "SELECT id,name,fulfillment_set_id,metadata FROM service_zone WHERE fulfillment_set_id=$1 AND deleted_at IS NULL ORDER BY created_at", id)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	v.ServiceZones = []models.ServiceZone{}
	for rows.Next() {
		var z models.ServiceZone
		if err = rows.Scan(&z.ID, &z.Name, &z.FulfillmentSetID, &z.Metadata); err != nil {
			return v, err
		}
		v.ServiceZones = append(v.ServiceZones, z)
	}
	if err = rows.Err(); err != nil {
		return v, err
	}
	rows.Close()
	for i := range v.ServiceZones {
		v.ServiceZones[i].GeoZones, err = r.listGeoZones(ctx, v.ServiceZones[i].ID)
		if err != nil {
			return v, err
		}
	}
	return v, nil
}
func (r *Repository) listGeoZones(ctx context.Context, zoneID string) ([]models.GeoZone, error) {
	rows, err := r.db.Query(ctx, "SELECT id,type,country_code,province_code,city,postal_expression,service_zone_id,metadata FROM geo_zone WHERE service_zone_id=$1 AND deleted_at IS NULL ORDER BY created_at", zoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.GeoZone{}
	for rows.Next() {
		var v models.GeoZone
		if err = rows.Scan(&v.ID, &v.Type, &v.CountryCode, &v.ProvinceCode, &v.City, &v.PostalExpression, &v.ServiceZoneID, &v.Metadata); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) GetServiceZone(ctx context.Context, id string) (models.ServiceZone, error) {
	var v models.ServiceZone
	err := r.db.QueryRow(ctx, "SELECT id,name,fulfillment_set_id,metadata FROM service_zone WHERE id=$1 AND deleted_at IS NULL", id).Scan(&v.ID, &v.Name, &v.FulfillmentSetID, &v.Metadata)
	if err != nil {
		return v, err
	}
	v.GeoZones, err = r.listGeoZones(ctx, id)
	return v, err
}
func insertGeo(ctx context.Context, tx pgx.Tx, zone string, in models.GeoZoneInput) (string, error) {
	id := entityid.New("fgz")
	typ := in.Type
	if typ == "" {
		typ = "country"
	}
	if len(in.PostalExpression) == 0 {
		in.PostalExpression = json.RawMessage("null")
	}
	if _, err := tx.Exec(ctx, "INSERT INTO geo_zone(id,type,country_code,province_code,city,postal_expression,service_zone_id,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", id, typ, in.CountryCode, in.ProvinceCode, in.City, in.PostalExpression, zone, metadata(in.Metadata)); err != nil {
		return "", err
	}
	return id, nil
}
func (r *Repository) CreateServiceZone(ctx context.Context, setID string, in models.ServiceZoneInput) (models.FulfillmentSet, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.FulfillmentSet{}, err
	}
	defer tx.Rollback(ctx)
	id := entityid.New("serzo")
	if _, err = tx.Exec(ctx, "INSERT INTO service_zone(id,name,fulfillment_set_id,metadata) VALUES($1,$2,$3,$4)", id, in.Name, setID, metadata(in.Metadata)); err != nil {
		return models.FulfillmentSet{}, err
	}
	for _, z := range in.GeoZones {
		if _, err = insertGeo(ctx, tx, id, z); err != nil {
			return models.FulfillmentSet{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.FulfillmentSet{}, err
	}
	return r.GetSet(ctx, setID)
}
func (r *Repository) UpdateServiceZone(ctx context.Context, id string, in models.ServiceZoneUpdate) (models.FulfillmentSet, error) {
	zone, err := r.GetServiceZone(ctx, id)
	if err != nil {
		return models.FulfillmentSet{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.FulfillmentSet{}, err
	}
	defer tx.Rollback(ctx)
	sets, args := []string{}, []any{id}
	put := func(c string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", c, len(args)))
	}
	if in.Name != nil {
		put("name", *in.Name)
	}
	if in.Metadata != nil {
		put("metadata", in.Metadata)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		if _, err = tx.Exec(ctx, "UPDATE service_zone SET "+strings.Join(sets, ",")+" WHERE id=$1 AND deleted_at IS NULL", args...); err != nil {
			return models.FulfillmentSet{}, err
		}
	}
	if in.GeoZones != nil {
		ids := []string{}
		for _, g := range *in.GeoZones {
			if g.ID != nil {
				ids = append(ids, *g.ID)
				meta := metadata(g.Metadata)
				typ := g.Type
				if typ == "" {
					typ = "country"
				}
				tag, execErr := tx.Exec(ctx, "UPDATE geo_zone SET type=$2,country_code=$3,province_code=$4,city=$5,postal_expression=$6,metadata=$7,updated_at=now() WHERE id=$1 AND service_zone_id=$8 AND deleted_at IS NULL", *g.ID, typ, g.CountryCode, g.ProvinceCode, g.City, metadata(g.PostalExpression), meta, id)
				if execErr != nil {
					err = execErr
					return models.FulfillmentSet{}, err
				}
				if tag.RowsAffected() == 0 {
					return models.FulfillmentSet{}, pgx.ErrNoRows
				}
			} else if _, err = insertGeo(ctx, tx, id, g); err != nil {
				return models.FulfillmentSet{}, err
			}
		}
		if len(ids) == 0 {
			_, err = tx.Exec(ctx, "UPDATE geo_zone SET deleted_at=now(),updated_at=now() WHERE service_zone_id=$1 AND deleted_at IS NULL", id)
		} else {
			_, err = tx.Exec(ctx, "UPDATE geo_zone SET deleted_at=now(),updated_at=now() WHERE service_zone_id=$1 AND deleted_at IS NULL AND id<>ALL($2)", id, ids)
		}
		if err != nil {
			return models.FulfillmentSet{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.FulfillmentSet{}, err
	}
	return r.GetSet(ctx, zone.FulfillmentSetID)
}
func (r *Repository) DeleteServiceZone(ctx context.Context, id string) (models.FulfillmentSet, error) {
	zone, err := r.GetServiceZone(ctx, id)
	if err != nil {
		return models.FulfillmentSet{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.FulfillmentSet{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "UPDATE geo_zone SET deleted_at=now(),updated_at=now() WHERE service_zone_id=$1 AND deleted_at IS NULL", id); err != nil {
		return models.FulfillmentSet{}, err
	}
	if _, err = tx.Exec(ctx, "UPDATE service_zone SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL", id); err != nil {
		return models.FulfillmentSet{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.FulfillmentSet{}, err
	}
	return r.GetSet(ctx, zone.FulfillmentSetID)
}
func (r *Repository) DeleteSet(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "UPDATE geo_zone SET deleted_at=now(),updated_at=now() WHERE service_zone_id IN (SELECT id FROM service_zone WHERE fulfillment_set_id=$1 AND deleted_at IS NULL) AND deleted_at IS NULL", id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "UPDATE service_zone SET deleted_at=now(),updated_at=now() WHERE fulfillment_set_id=$1 AND deleted_at IS NULL", id); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, "UPDATE fulfillment_set SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
}

func (r *Repository) RestoreSet(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE fulfillment_set SET deleted_at=NULL,updated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE service_zone SET deleted_at=NULL,updated_at=now() WHERE fulfillment_set_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE geo_zone SET deleted_at=NULL,updated_at=now() WHERE service_zone_id IN (SELECT id FROM service_zone WHERE fulfillment_set_id=$1)`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ domain.Repository = (*Repository)(nil)
