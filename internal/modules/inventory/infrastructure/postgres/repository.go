package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/example/medusa-go-clone/internal/modules/inventory/domain"
	"github.com/example/medusa-go-clone/internal/modules/inventory/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
	"strings"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }
func scanItem(row pgx.Row) (models.InventoryItem, error) {
	var v models.InventoryItem
	err := row.Scan(&v.ID, &v.SKU, &v.OriginCountry, &v.HSCode, &v.MidCode, &v.Material, &v.UnitOfMeasure, &v.Weight, &v.Length, &v.Height, &v.Width, &v.RequiresShipping, &v.Description, &v.Title, &v.Thumbnail, &v.Metadata, &v.ReservedQuantity, &v.StockedQuantity, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

const itemCols = `i.id,i.sku,i.origin_country,i.hs_code,i.mid_code,i.material,i.unit_of_measure,i.weight,i.length,i.height,i.width,i.requires_shipping,i.description,i.title,i.thumbnail,i.metadata,COALESCE((SELECT sum(l.reserved_quantity) FROM inventory_level l WHERE l.inventory_item_id=i.id AND l.deleted_at IS NULL),0),COALESCE((SELECT sum(l.stocked_quantity) FROM inventory_level l WHERE l.inventory_item_id=i.id AND l.deleted_at IS NULL),0),i.created_at,i.updated_at`

func (r *Repository) ListItems(ctx context.Context, f models.Filter) ([]models.InventoryItem, int, error) {
	where := []string{"i.deleted_at IS NULL"}
	args := []any{}
	add := func(col string, v []string) {
		if len(v) > 0 {
			args = append(args, v)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", col, len(args)))
		}
	}
	add("i.id", f.IDs)
	add("i.sku", f.SKUs)
	add("i.origin_country", f.OriginCountries)
	add("i.mid_code", f.MidCodes)
	add("i.hs_code", f.HSCodes)
	add("i.material", f.Materials)
	if f.RequiresShipping != nil {
		args = append(args, *f.RequiresShipping)
		where = append(where, fmt.Sprintf("i.requires_shipping=$%d", len(args)))
	}
	if len(f.LocationIDs) > 0 {
		args = append(args, f.LocationIDs)
		where = append(where, fmt.Sprintf("EXISTS(SELECT 1 FROM inventory_level ll WHERE ll.inventory_item_id=i.id AND ll.deleted_at IS NULL AND ll.location_id=ANY($%d))", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(COALESCE(i.sku,'') ILIKE $%d OR COALESCE(i.title,'') ILIKE $%d OR COALESCE(i.description,'') ILIKE $%d OR COALESCE(i.hs_code,'') ILIKE $%d)", len(args), len(args), len(args), len(args)))
	}
	w := ` WHERE ` + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM inventory_item i`+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s FROM inventory_item i%s ORDER BY i.created_at DESC LIMIT $%d OFFSET $%d`, itemCols, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.InventoryItem{}
	for rows.Next() {
		v, e := scanItem(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	for n := range out {
		lv, _, e := r.ListLevels(ctx, out[n].ID, nil, 1000, 0)
		if e != nil {
			return nil, 0, e
		}
		out[n].LocationLevels = lv
	}
	return out, count, nil
}
func (r *Repository) GetItem(ctx context.Context, id string) (models.InventoryItem, error) {
	v, err := scanItem(r.db.QueryRow(ctx, `SELECT `+itemCols+` FROM inventory_item i WHERE i.id=$1 AND i.deleted_at IS NULL`, id))
	if err != nil {
		return v, err
	}
	v.LocationLevels, _, err = r.ListLevels(ctx, id, nil, 1000, 0)
	return v, err
}
func (r *Repository) CreateItem(ctx context.Context, in models.ItemInput) (models.InventoryItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.InventoryItem{}, err
	}
	defer tx.Rollback(ctx)
	id := entityid.New("iitem")
	shipping := true
	if in.RequiresShipping != nil {
		shipping = *in.RequiresShipping
	}
	_, err = tx.Exec(ctx, `INSERT INTO inventory_item(id,sku,origin_country,hs_code,mid_code,material,unit_of_measure,weight,length,height,width,requires_shipping,description,title,thumbnail,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, id, in.SKU, in.OriginCountry, in.HSCode, in.MidCode, in.Material, in.UnitOfMeasure, in.Weight, in.Length, in.Height, in.Width, shipping, in.Description, in.Title, in.Thumbnail, jsonNull(in.Metadata))
	if err != nil {
		return models.InventoryItem{}, err
	}
	for _, lv := range in.LocationLevels {
		if _, err = insertLevel(ctx, tx, id, lv); err != nil {
			return models.InventoryItem{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.InventoryItem{}, err
	}
	return r.GetItem(ctx, id)
}
func (r *Repository) UpdateItem(ctx context.Context, id string, in models.UpdateItemInput) (models.InventoryItem, error) {
	sets := []string{}
	args := []any{id}
	put := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	if in.SKU != nil {
		put("sku", in.SKU)
	}
	if in.OriginCountry != nil {
		put("origin_country", in.OriginCountry)
	}
	if in.HSCode != nil {
		put("hs_code", in.HSCode)
	}
	if in.MidCode != nil {
		put("mid_code", in.MidCode)
	}
	if in.Material != nil {
		put("material", in.Material)
	}
	if in.UnitOfMeasure != nil {
		put("unit_of_measure", in.UnitOfMeasure)
	}
	if in.Weight != nil {
		put("weight", in.Weight)
	}
	if in.Length != nil {
		put("length", in.Length)
	}
	if in.Height != nil {
		put("height", in.Height)
	}
	if in.Width != nil {
		put("width", in.Width)
	}
	if in.RequiresShipping != nil {
		put("requires_shipping", *in.RequiresShipping)
	}
	if in.Description != nil {
		put("description", in.Description)
	}
	if in.Title != nil {
		put("title", in.Title)
	}
	if in.Thumbnail != nil {
		put("thumbnail", in.Thumbnail)
	}
	if in.Metadata != nil {
		put("metadata", in.Metadata)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := r.db.Exec(ctx, `UPDATE inventory_item SET `+strings.Join(sets, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
		if err != nil {
			return models.InventoryItem{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.InventoryItem{}, pgx.ErrNoRows
		}
	}
	return r.GetItem(ctx, id)
}
func (r *Repository) DeleteItem(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var reserved float64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(reserved_quantity),0) FROM inventory_level WHERE inventory_item_id=$1 AND deleted_at IS NULL`, id).Scan(&reserved); err != nil {
		return err
	}
	if reserved > 0 {
		return fmt.Errorf("cannot remove inventory item %s since it has reservations", id)
	}
	tag, err := tx.Exec(ctx, `UPDATE inventory_item SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	for _, table := range []string{"inventory_level", "reservation_item", "product_variant_inventory_item"} {
		column := "inventory_item_id"
		if table == "product_variant_inventory_item" {
			column = "inventory_item_id"
		}
		if _, err = tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET deleted_at=now(),updated_at=now() WHERE %s=$1 AND deleted_at IS NULL`, table, column), id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (r *Repository) ListLevels(ctx context.Context, itemID string, locations []string, limit, offset int) ([]models.InventoryLevel, int, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	if itemID != "" {
		args = append(args, itemID)
		where = append(where, fmt.Sprintf("inventory_item_id=$%d", len(args)))
	}
	if len(locations) > 0 {
		args = append(args, locations)
		where = append(where, fmt.Sprintf("location_id=ANY($%d)", len(args)))
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM inventory_level`+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT id,inventory_item_id,location_id,stocked_quantity,reserved_quantity,incoming_quantity,stocked_quantity-reserved_quantity AS available_quantity,metadata,created_at,updated_at FROM inventory_level%s ORDER BY created_at LIMIT $%d OFFSET $%d`, w, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.InventoryLevel{}
	for rows.Next() {
		var v models.InventoryLevel
		if err := rows.Scan(&v.ID, &v.InventoryItemID, &v.LocationID, &v.StockedQuantity, &v.ReservedQuantity, &v.IncomingQuantity, &v.AvailableQuantity, &v.Metadata, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, count, rows.Err()
}
func (r *Repository) CreateLevels(ctx context.Context, in []models.LevelInput) ([]models.InventoryLevel, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	out := []models.InventoryLevel{}
	for _, v := range in {
		itemID := v.InventoryItemID
		if itemID == "" {
			return nil, fmt.Errorf("inventory_item_id is required")
		}
		created, err := insertLevel(ctx, tx, itemID, v)
		if err != nil {
			return nil, err
		}
		out = append(out, created)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
func insertLevel(ctx context.Context, tx pgx.Tx, itemID string, v models.LevelInput) (models.InventoryLevel, error) {
	stock, incoming := float64(0), float64(0)
	if v.StockedQuantity != nil {
		stock = *v.StockedQuantity
	}
	if v.IncomingQuantity != nil {
		incoming = *v.IncomingQuantity
	}
	id := entityid.New("ilev")
	var out models.InventoryLevel
	err := tx.QueryRow(ctx, `INSERT INTO inventory_level(id,inventory_item_id,location_id,stocked_quantity,reserved_quantity,incoming_quantity,raw_stocked_quantity,raw_reserved_quantity,raw_incoming_quantity,metadata) VALUES($1,$2,$3,$4,0,$5,$6,$7,$8,$9) RETURNING id,inventory_item_id,location_id,stocked_quantity,reserved_quantity,incoming_quantity,stocked_quantity-reserved_quantity,metadata,created_at,updated_at`, id, itemID, v.LocationID, stock, incoming, rawNumber(stock), rawNumber(0), rawNumber(incoming), jsonNull(v.Metadata)).Scan(&out.ID, &out.InventoryItemID, &out.LocationID, &out.StockedQuantity, &out.ReservedQuantity, &out.IncomingQuantity, &out.AvailableQuantity, &out.Metadata, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}
func (r *Repository) UpdateLevels(ctx context.Context, in []models.LevelUpdate) ([]models.InventoryLevel, error) {
	out := []models.InventoryLevel{}
	for _, v := range in {
		sets := []string{}
		args := []any{v.ID}
		if v.StockedQuantity != nil {
			args = append(args, *v.StockedQuantity)
			sets = append(sets, fmt.Sprintf("stocked_quantity=$%d", len(args)))
			args = append(args, rawNumber(*v.StockedQuantity))
			sets = append(sets, fmt.Sprintf("raw_stocked_quantity=$%d", len(args)))
		}
		if v.IncomingQuantity != nil {
			args = append(args, *v.IncomingQuantity)
			sets = append(sets, fmt.Sprintf("incoming_quantity=$%d", len(args)))
			args = append(args, rawNumber(*v.IncomingQuantity))
			sets = append(sets, fmt.Sprintf("raw_incoming_quantity=$%d", len(args)))
		}
		if v.Metadata != nil {
			args = append(args, v.Metadata)
			sets = append(sets, fmt.Sprintf("metadata=$%d", len(args)))
		}
		if len(sets) == 0 {
			continue
		}
		sets = append(sets, "updated_at=now()")
		_, err := r.db.Exec(ctx, `UPDATE inventory_level SET `+strings.Join(sets, ",")+` WHERE id=$1 AND deleted_at IS NULL`, args...)
		if err != nil {
			return nil, err
		}
		rows, _, err := r.ListLevels(ctx, v.InventoryItemID, []string{v.LocationID}, 1, 0)
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			out = append(out, rows[0])
		}
	}
	return out, nil
}
func (r *Repository) DeleteLevels(ctx context.Context, ids []string, force bool) ([]string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,stocked_quantity,reserved_quantity,incoming_quantity FROM inventory_level WHERE id=ANY($1) AND deleted_at IS NULL FOR UPDATE`, ids)
	if err != nil {
		return nil, err
	}
	type vlevel struct {
		id                   string
		stock, res, incoming float64
	}
	levels := []vlevel{}
	for rows.Next() {
		var v vlevel
		if err = rows.Scan(&v.id, &v.stock, &v.res, &v.incoming); err != nil {
			rows.Close()
			return nil, err
		}
		levels = append(levels, v)
	}
	rows.Close()
	deleted := []string{}
	for _, v := range levels {
		if v.res > 0 || v.incoming > 0 || (!force && v.stock > 0) {
			continue
		}
		if _, err = tx.Exec(ctx, `UPDATE inventory_level SET deleted_at=now(),updated_at=now() WHERE id=$1`, v.id); err != nil {
			return nil, err
		}
		deleted = append(deleted, v.id)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return deleted, nil
}
func (r *Repository) UpdateLevelAtLocation(ctx context.Context, id, location string, in models.LevelUpdate) (models.InventoryItem, error) {
	rows, _, err := r.ListLevels(ctx, id, []string{location}, 1, 0)
	if err != nil {
		return models.InventoryItem{}, err
	}
	if len(rows) == 0 {
		return models.InventoryItem{}, pgx.ErrNoRows
	}
	in.ID = rows[0].ID
	in.InventoryItemID = id
	in.LocationID = location
	if _, err = r.UpdateLevels(ctx, []models.LevelUpdate{in}); err != nil {
		return models.InventoryItem{}, err
	}
	return r.GetItem(ctx, id)
}
func (r *Repository) DeleteLevelAtLocation(ctx context.Context, id, location string) (string, error) {
	rows, _, err := r.ListLevels(ctx, id, []string{location}, 1, 0)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", pgx.ErrNoRows
	}
	if rows[0].ReservedQuantity > 0 {
		return "", fmt.Errorf("cannot remove inventory level because reservations exist")
	}
	deleted, err := r.DeleteLevels(ctx, []string{rows[0].ID}, false)
	if err != nil {
		return "", err
	}
	if len(deleted) == 0 {
		return "", fmt.Errorf("inventory level cannot be deleted while stock or incoming quantity remains")
	}
	return rows[0].ID, nil
}

func (r *Repository) LocationsExist(ctx context.Context, ids []string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT count(DISTINCT id) FROM stock_location WHERE id=ANY($1) AND deleted_at IS NULL`, ids).Scan(&count)
	return count == len(uniqueStrings(ids)), err
}

func uniqueStrings(ids []string) []string {
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
func jsonNull(v []byte) []byte {
	if v == nil {
		return []byte("null")
	}
	return v
}
func rawNumber(v float64) []byte {
	b, _ := json.Marshal(map[string]any{"value": strconv.FormatFloat(v, 'f', -1, 64), "precision": 20})
	return b
}

const reservationCols = `id,inventory_item_id,line_item_id,allow_backorder,location_id,quantity,raw_quantity,external_id,description,created_by,metadata,created_at,updated_at`

func scanReservation(row pgx.Row) (models.ReservationItem, error) {
	var v models.ReservationItem
	err := row.Scan(&v.ID, &v.InventoryItemID, &v.LineItemID, &v.AllowBackorder, &v.LocationID, &v.Quantity, &v.RawQuantity, &v.ExternalID, &v.Description, &v.CreatedBy, &v.Metadata, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

func (r *Repository) ListReservations(ctx context.Context, f models.ReservationFilter) ([]models.ReservationItem, int, error) {
	where, args := []string{"deleted_at IS NULL"}, []any{}
	add := func(col string, values []string) {
		if len(values) > 0 {
			args = append(args, values)
			where = append(where, fmt.Sprintf("%s=ANY($%d)", col, len(args)))
		}
	}
	add("id", f.IDs)
	add("inventory_item_id", f.InventoryItemIDs)
	add("location_id", f.LocationIDs)
	add("line_item_id", f.LineItemIDs)
	add("created_by", f.CreatedBy)
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(COALESCE(description,'') ILIKE $%d OR COALESCE(external_id,'') ILIKE $%d)", len(args), len(args)))
	}
	if f.Description != "" {
		args = append(args, "%"+f.Description+"%")
		where = append(where, fmt.Sprintf("COALESCE(description,'') ILIKE $%d", len(args)))
	}
	w := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM reservation_item`+w, args...).Scan(&count); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(ctx, `SELECT `+reservationCols+` FROM reservation_item`+w+fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []models.ReservationItem{}
	for rows.Next() {
		v, err := scanReservation(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	rows.Close()
	for i := range out {
		item, err := r.GetItem(ctx, out[i].InventoryItemID)
		if err != nil {
			return nil, 0, err
		}
		out[i].InventoryItem = &item
	}
	return out, count, nil
}

func (r *Repository) GetReservation(ctx context.Context, id string) (models.ReservationItem, error) {
	reservation, err := scanReservation(r.db.QueryRow(ctx, `SELECT `+reservationCols+` FROM reservation_item WHERE id=$1 AND deleted_at IS NULL`, id))
	if err != nil {
		return reservation, err
	}
	item, err := r.GetItem(ctx, reservation.InventoryItemID)
	if err != nil {
		return reservation, err
	}
	reservation.InventoryItem = &item
	return reservation, nil
}

func (r *Repository) CreateReservations(ctx context.Context, input []models.ReservationInput, actor *string) ([]models.ReservationItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	ids := make([]string, 0, len(input))
	for _, in := range input {
		id := entityid.New("resitem")
		metadata := jsonNull(in.Metadata)
		_, err = tx.Exec(ctx, `INSERT INTO reservation_item(id,inventory_item_id,line_item_id,location_id,quantity,raw_quantity,description,created_by,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, in.InventoryItemID, in.LineItemID, in.LocationID, in.Quantity, rawNumber(in.Quantity), in.Description, actor, metadata)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	out := make([]models.ReservationItem, 0, len(ids))
	for _, id := range ids {
		v, err := r.GetReservation(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *Repository) UpdateReservations(ctx context.Context, input []models.ReservationUpdate) ([]models.ReservationItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	ids := make([]string, 0, len(input))
	for _, in := range input {
		sets, args := []string{}, []any{in.ID}
		put := func(column string, value any) {
			args = append(args, value)
			sets = append(sets, fmt.Sprintf("%s=$%d", column, len(args)))
		}
		if in.LocationID != nil {
			put("location_id", *in.LocationID)
		}
		if in.Quantity != nil {
			put("quantity", *in.Quantity)
			put("raw_quantity", rawNumber(*in.Quantity))
		}
		if in.Description != nil {
			var description *string
			if string(in.Description) != "null" {
				var value string
				if err := json.Unmarshal(in.Description, &value); err != nil {
					return nil, err
				}
				description = &value
			}
			put("description", description)
		}
		if in.Metadata != nil {
			put("metadata", in.Metadata)
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			tag, err := tx.Exec(ctx, "UPDATE reservation_item SET "+strings.Join(sets, ",")+" WHERE id=$1 AND deleted_at IS NULL", args...)
			if err != nil {
				return nil, err
			}
			if tag.RowsAffected() == 0 {
				return nil, pgx.ErrNoRows
			}
		}
		ids = append(ids, in.ID)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	out := make([]models.ReservationItem, 0, len(ids))
	for _, id := range ids {
		v, err := r.GetReservation(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *Repository) DeleteReservations(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `UPDATE reservation_item SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids)
	return err
}

func (r *Repository) RestoreReservations(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `UPDATE reservation_item SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1) AND deleted_at IS NOT NULL`, ids)
	return err
}

var _ domain.Repository = (*Repository)(nil)
