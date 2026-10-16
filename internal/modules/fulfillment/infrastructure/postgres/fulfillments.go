package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/fulfillment/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
)

const fulfillmentColumns = `id,location_id,packed_at,shipped_at,marked_shipped_by,created_by,delivered_at,canceled_at,data,requires_shipping,provider_id,shipping_option_id,metadata,delivery_address_id,created_at,updated_at,deleted_at`

func (r *Repository) PersistFulfillment(ctx context.Context, input models.FulfillmentCreateInput) (models.FulfillmentRecord, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM stock_location WHERE id=$1 AND deleted_at IS NULL)`, input.LocationID).Scan(&exists); err != nil {
		return models.FulfillmentRecord{}, err
	}
	if !exists {
		return models.FulfillmentRecord{}, pgx.ErrNoRows
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fulfillment_provider WHERE id=$1 AND is_enabled=true)`, input.ProviderID).Scan(&exists); err != nil {
		return models.FulfillmentRecord{}, err
	}
	if !exists {
		return models.FulfillmentRecord{}, fmt.Errorf("fulfillment provider %q is not registered", input.ProviderID)
	}
	addressID := entityid.New("fuladdr")
	address := input.DeliveryAddress
	addressJSON, _ := json.Marshal(address.Metadata)
	_, err = tx.Exec(ctx, `INSERT INTO fulfillment_address(id,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, addressID, address.Company, address.FirstName, address.LastName, address.Address1, address.Address2, address.City, address.CountryCode, address.Province, address.PostalCode, address.Phone, addressJSON)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	fulfillmentID := entityid.New("ful")
	data := input.Data
	metadata := input.Metadata
	if len(data) == 0 {
		data = []byte("null")
	}
	if len(metadata) == 0 {
		metadata = []byte("null")
	}
	_, err = tx.Exec(ctx, `INSERT INTO fulfillment(id,location_id,packed_at,shipped_at,created_by,delivered_at,canceled_at,data,provider_id,shipping_option_id,metadata,delivery_address_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, fulfillmentID, input.LocationID, input.PackedAt, input.ShippedAt, input.CreatedBy, input.DeliveredAt, input.CanceledAt, data, input.ProviderID, input.ShippingOptionID, metadata, addressID)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	for _, item := range input.Items {
		quantity := strings.TrimSpace(string(item.Quantity))
		if quantity == "" {
			quantity = "0"
		}
		rawQuantity, _ := json.Marshal(map[string]any{"value": quantity, "precision": 20})
		if _, err = tx.Exec(ctx, `INSERT INTO fulfillment_item(id,title,sku,barcode,quantity,raw_quantity,line_item_id,inventory_item_id,fulfillment_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, entityid.New("fulit"), item.Title, item.SKU, item.Barcode, quantity, rawQuantity, item.LineItemID, item.InventoryItemID, fulfillmentID); err != nil {
			return models.FulfillmentRecord{}, err
		}
	}
	for _, label := range input.Labels {
		if _, err = tx.Exec(ctx, `INSERT INTO fulfillment_label(id,tracking_number,tracking_url,label_url,fulfillment_id) VALUES($1,$2,$3,$4,$5)`, entityid.New("fulla"), label.TrackingNumber, label.TrackingURL, label.LabelURL, fulfillmentID); err != nil {
			return models.FulfillmentRecord{}, err
		}
	}
	if input.OrderID != "" {
		if _, err = tx.Exec(ctx, `INSERT INTO order_fulfillment(id,order_id,fulfillment_id) VALUES($1,$2,$3)`, entityid.New("ordful"), input.OrderID, fulfillmentID); err != nil {
			return models.FulfillmentRecord{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.FulfillmentRecord{}, err
	}
	return r.GetFulfillment(ctx, fulfillmentID)
}

func (r *Repository) GetFulfillment(ctx context.Context, id string) (models.FulfillmentRecord, error) {
	var v models.FulfillmentRecord
	var addressID *string
	if err := r.db.QueryRow(ctx, `SELECT `+fulfillmentColumns+` FROM fulfillment WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&v.ID, &v.LocationID, &v.PackedAt, &v.ShippedAt, &v.MarkedShippedBy, &v.CreatedBy, &v.DeliveredAt, &v.CanceledAt, &v.Data, &v.RequiresShipping, &v.ProviderID, &v.ShippingOptionID, &v.Metadata, &addressID, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt); err != nil {
		return v, err
	}
	v.Items = []models.FulfillmentItem{}
	itemRows, err := r.db.Query(ctx, `SELECT id,title,sku,barcode,raw_quantity,line_item_id,inventory_item_id FROM fulfillment_item WHERE fulfillment_id=$1 AND deleted_at IS NULL ORDER BY created_at,id`, id)
	if err != nil {
		return v, err
	}
	for itemRows.Next() {
		var item models.FulfillmentItem
		if err = itemRows.Scan(&item.ID, &item.Title, &item.SKU, &item.Barcode, &item.Quantity, &item.LineItemID, &item.InventoryItemID); err != nil {
			itemRows.Close()
			return v, err
		}
		v.Items = append(v.Items, item)
	}
	err = itemRows.Err()
	itemRows.Close()
	if err != nil {
		return v, err
	}
	v.Labels = []models.FulfillmentLabel{}
	labelRows, err := r.db.Query(ctx, `SELECT id,tracking_number,tracking_url,label_url FROM fulfillment_label WHERE fulfillment_id=$1 AND deleted_at IS NULL ORDER BY created_at,id`, id)
	if err != nil {
		return v, err
	}
	for labelRows.Next() {
		var label models.FulfillmentLabel
		if err = labelRows.Scan(&label.ID, &label.TrackingNumber, &label.TrackingURL, &label.LabelURL); err != nil {
			labelRows.Close()
			return v, err
		}
		v.Labels = append(v.Labels, label)
	}
	err = labelRows.Err()
	labelRows.Close()
	if err != nil {
		return v, err
	}
	if addressID != nil {
		var address models.FulfillmentAddress
		address.ID = *addressID
		if err = r.db.QueryRow(ctx, `SELECT company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata FROM fulfillment_address WHERE id=$1 AND deleted_at IS NULL`, *addressID).Scan(&address.Company, &address.FirstName, &address.LastName, &address.Address1, &address.Address2, &address.City, &address.CountryCode, &address.Province, &address.PostalCode, &address.Phone, &address.Metadata); err != nil {
			return v, err
		}
		v.DeliveryAddress = &address
	}
	var orderID string
	if err = r.db.QueryRow(ctx, `SELECT order_id FROM order_fulfillment WHERE fulfillment_id=$1 AND deleted_at IS NULL ORDER BY created_at LIMIT 1`, id).Scan(&orderID); err == nil {
		v.OrderID = &orderID
	} else if err != pgx.ErrNoRows {
		return v, err
	}
	return v, nil
}

func (r *Repository) UpdateFulfillmentProviderResult(ctx context.Context, id string, result models.FulfillmentProviderResult) (models.FulfillmentRecord, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.FulfillmentRecord{}, err
	}
	defer tx.Rollback(ctx)
	data, _ := json.Marshal(result.Data)
	if result.Data == nil {
		data = []byte("{}")
	}
	if _, err = tx.Exec(ctx, `UPDATE fulfillment SET data=$2,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, data); err != nil {
		return models.FulfillmentRecord{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE fulfillment_label SET deleted_at=now(),updated_at=now() WHERE fulfillment_id=$1 AND deleted_at IS NULL`, id); err != nil {
		return models.FulfillmentRecord{}, err
	}
	for _, raw := range result.Labels {
		var label models.FulfillmentLabel
		b, e := json.Marshal(raw)
		if e != nil {
			return models.FulfillmentRecord{}, e
		}
		if e = json.Unmarshal(b, &label); e != nil {
			return models.FulfillmentRecord{}, e
		}
		if label.TrackingNumber == "" && label.TrackingURL == "" && label.LabelURL == "" {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO fulfillment_label(id,tracking_number,tracking_url,label_url,fulfillment_id) VALUES($1,$2,$3,$4,$5)`, entityid.New("fulla"), label.TrackingNumber, label.TrackingURL, label.LabelURL, id); err != nil {
			return models.FulfillmentRecord{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.FulfillmentRecord{}, err
	}
	return r.GetFulfillment(ctx, id)
}

func (r *Repository) HardDeleteFulfillment(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var addressID *string
	if err = tx.QueryRow(ctx, `SELECT delivery_address_id FROM fulfillment WHERE id=$1`, id).Scan(&addressID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM order_fulfillment WHERE fulfillment_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM fulfillment WHERE id=$1`, id); err != nil {
		return err
	}
	if addressID != nil {
		if _, err = tx.Exec(ctx, `DELETE FROM fulfillment_address WHERE id=$1`, *addressID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) MarkFulfillmentCanceled(ctx context.Context, id string) (models.FulfillmentRecord, error) {
	current, err := r.GetFulfillment(ctx, id)
	if err != nil {
		return current, err
	}
	if current.ShippedAt != nil {
		return current, fmt.Errorf("fulfillment with id %s already shipped", id)
	}
	if current.DeliveredAt != nil {
		return current, fmt.Errorf("fulfillment with id %s already delivered", id)
	}
	if current.CanceledAt != nil {
		return current, nil
	}
	if _, err = r.db.Exec(ctx, `UPDATE fulfillment SET canceled_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id); err != nil {
		return current, err
	}
	return r.GetFulfillment(ctx, id)
}

func (r *Repository) MarkFulfillmentShipped(ctx context.Context, id string, input models.CreateShipmentInput, shippedAt time.Time) (models.FulfillmentRecord, models.FulfillmentRecord, error) {
	previous, err := r.GetFulfillment(ctx, id)
	if err != nil {
		return previous, models.FulfillmentRecord{}, err
	}
	if previous.ShippedAt != nil {
		return previous, models.FulfillmentRecord{}, fmt.Errorf("shipment has already been created")
	}
	if previous.CanceledAt != nil {
		return previous, models.FulfillmentRecord{}, fmt.Errorf("cannot create shipment for a canceled fulfillment")
	}
	if previous.ShippingOptionID == nil || *previous.ShippingOptionID == "" {
		return previous, models.FulfillmentRecord{}, fmt.Errorf("cannot create shipment without a shipping option")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return previous, models.FulfillmentRecord{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE fulfillment SET shipped_at=$2,marked_shipped_by=$3,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, shippedAt, input.MarkedShippedBy); err != nil {
		return previous, models.FulfillmentRecord{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE fulfillment_label SET deleted_at=now(),updated_at=now() WHERE fulfillment_id=$1 AND deleted_at IS NULL`, id); err != nil {
		return previous, models.FulfillmentRecord{}, err
	}
	for _, label := range input.Labels {
		if _, err = tx.Exec(ctx, `INSERT INTO fulfillment_label(id,tracking_number,tracking_url,label_url,fulfillment_id) VALUES($1,$2,$3,$4,$5)`, entityid.New("fulla"), label.TrackingNumber, label.TrackingURL, label.LabelURL, id); err != nil {
			return previous, models.FulfillmentRecord{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return previous, models.FulfillmentRecord{}, err
	}
	updated, err := r.GetFulfillment(ctx, id)
	return previous, updated, err
}

func (r *Repository) RestoreFulfillment(ctx context.Context, previous models.FulfillmentRecord) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE fulfillment SET packed_at=$2,shipped_at=$3,marked_shipped_by=$4,delivered_at=$5,canceled_at=$6,data=$7,metadata=$8,updated_at=$9 WHERE id=$1`, previous.ID, previous.PackedAt, previous.ShippedAt, previous.MarkedShippedBy, previous.DeliveredAt, previous.CanceledAt, previous.Data, previous.Metadata, previous.UpdatedAt); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE fulfillment_label SET deleted_at=now(),updated_at=now() WHERE fulfillment_id=$1 AND deleted_at IS NULL`, previous.ID); err != nil {
		return err
	}
	for _, label := range previous.Labels {
		if _, err = tx.Exec(ctx, `INSERT INTO fulfillment_label(id,tracking_number,tracking_url,label_url,fulfillment_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,now(),now()) ON CONFLICT(id) DO UPDATE SET tracking_number=EXCLUDED.tracking_number,tracking_url=EXCLUDED.tracking_url,label_url=EXCLUDED.label_url,deleted_at=NULL,updated_at=now()`, label.ID, label.TrackingNumber, label.TrackingURL, label.LabelURL, previous.ID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
