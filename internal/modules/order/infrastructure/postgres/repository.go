package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/medusa-go-clone/internal/modules/order/domain"
	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, f models.Filter) (models.ListResult, error) {
	where := []string{"o.deleted_at IS NULL"}
	args := []any{}
	if len(f.IDs) > 0 {
		args = append(args, f.IDs)
		where = append(where, fmt.Sprintf("o.id=ANY($%d)", len(args)))
	}
	if len(f.CustomerIDs) > 0 {
		args = append(args, f.CustomerIDs)
		where = append(where, fmt.Sprintf("o.customer_id=ANY($%d)", len(args)))
	} else if f.CustomerID != nil {
		args = append(args, *f.CustomerID)
		where = append(where, fmt.Sprintf("o.customer_id=$%d", len(args)))
	}
	if len(f.RegionIDs) > 0 {
		args = append(args, f.RegionIDs)
		where = append(where, fmt.Sprintf("o.region_id=ANY($%d)", len(args)))
	} else if f.RegionID != nil {
		args = append(args, *f.RegionID)
		where = append(where, fmt.Sprintf("o.region_id=$%d", len(args)))
	}
	if len(f.SalesChannelIDs) > 0 {
		args = append(args, f.SalesChannelIDs)
		where = append(where, fmt.Sprintf("o.sales_channel_id=ANY($%d)", len(args)))
	} else if f.SalesChannelID != nil {
		args = append(args, *f.SalesChannelID)
		where = append(where, fmt.Sprintf("o.sales_channel_id=$%d", len(args)))
	}
	if f.IsDraft != nil {
		args = append(args, *f.IsDraft)
		where = append(where, fmt.Sprintf("o.is_draft_order=$%d", len(args)))
	}
	if len(f.Status) > 0 {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("o.status=ANY($%d)", len(args)))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(o.id ILIKE $%d OR o.email ILIKE $%d OR o.custom_display_id ILIKE $%d OR CAST(o.display_id AS TEXT) ILIKE $%d)", len(args), len(args), len(args), len(args)))
	}
	for _, filter := range []struct {
		column string
		values map[string]string
	}{{"o.created_at", f.CreatedAt}, {"o.updated_at", f.UpdatedAt}} {
		for op, value := range filter.values {
			operator := map[string]string{"$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<=", "$eq": "=", "$ne": "!="}[op]
			if operator == "" {
				continue
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("%s %s $%d", filter.column, operator, len(args)))
		}
	}
	clause := " WHERE " + strings.Join(where, " AND ")
	var count int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM \"order\" o"+clause, args...).Scan(&count); err != nil {
		return models.ListResult{}, err
	}
	orderBy := "o.created_at DESC"
	if f.Order == "created_at" {
		orderBy = "o.created_at ASC"
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf("SELECT %s FROM \"order\" o%s ORDER BY %s LIMIT $%d OFFSET $%d", orderColumns, clause, orderBy, len(args)-1, len(args)), args...)
	if err != nil {
		return models.ListResult{}, err
	}
	defer rows.Close()
	out := models.ListResult{Orders: []models.Order{}, Count: count}
	type relatedOrder struct {
		order             models.Order
		shipping, billing *string
	}
	selected := []relatedOrder{}
	for rows.Next() {
		v, shipping, billing, e := scanOrder(rows)
		if e != nil {
			return out, e
		}
		selected = append(selected, relatedOrder{v, shipping, billing})
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	rows.Close()
	for _, item := range selected {
		if err = r.populate(ctx, &item.order, item.shipping, item.billing); err != nil {
			return out, err
		}
		out.Orders = append(out.Orders, item.order)
	}
	return out, nil
}

const orderColumns = `o.id,o.display_id,o.custom_display_id,o.region_id,o.customer_id,o.version,o.sales_channel_id,o.status,o.is_draft_order,o.email,o.currency_code,o.locale,o.no_notification,o.metadata,o.canceled_at,o.shipping_address_id,o.billing_address_id,o.created_at,o.updated_at`

type scanner interface{ Scan(...any) error }

func scanOrder(row scanner) (models.Order, *string, *string, error) {
	var v models.Order
	var shipping, billing *string
	err := row.Scan(&v.ID, &v.DisplayID, &v.CustomDisplayID, &v.RegionID, &v.CustomerID, &v.Version, &v.SalesChannelID, &v.Status, &v.IsDraftOrder, &v.Email, &v.CurrencyCode, &v.Locale, &v.NoNotification, &v.Metadata, &v.CanceledAt, &shipping, &billing, &v.CreatedAt, &v.UpdatedAt)
	return v, shipping, billing, err
}
func (r *Repository) Get(ctx context.Context, id string, version *int) (models.Order, error) {
	var v models.Order
	var shipping, billing *string
	query := "SELECT " + orderColumns + " FROM \"order\" o WHERE o.id=$1 AND o.deleted_at IS NULL"
	args := []any{id}
	if version != nil {
		args = append(args, *version)
		query += fmt.Sprintf(" AND o.version=$%d", len(args))
	} else {
		query += " ORDER BY o.version DESC LIMIT 1"
	}
	var err error
	v, shipping, billing, err = scanOrder(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		return v, err
	}
	if err = r.populate(ctx, &v, shipping, billing); err != nil {
		return v, err
	}
	return v, nil
}

func (r *Repository) Archive(ctx context.Context, ids []string) ([]models.Order, error) {
	if len(ids) == 0 {
		return []models.Order{}, nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,status FROM "order" WHERE id=ANY($1) AND deleted_at IS NULL FOR UPDATE`, ids)
	if err != nil {
		return nil, err
	}
	statuses := map[string]string{}
	for rows.Next() {
		var id, status string
		if err = rows.Scan(&id, &status); err != nil {
			rows.Close()
			return nil, err
		}
		statuses[id] = status
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(statuses) != len(ids) {
		return nil, pgx.ErrNoRows
	}
	blocked := []string{}
	for _, id := range ids {
		status := statuses[id]
		if status != "completed" && status != "canceled" && status != "draft" {
			blocked = append(blocked, id)
		}
	}
	if len(blocked) > 0 {
		return nil, domain.InvalidData(fmt.Sprintf("Orders %s are completed, canceled, or in draft and cannot be archived", strings.Join(blocked, ", ")))
	}
	if _, err = tx.Exec(ctx, `UPDATE "order" SET status='archived',updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	out := make([]models.Order, 0, len(ids))
	for _, id := range ids {
		row, e := r.Get(ctx, id, nil)
		if e != nil {
			return nil, e
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *Repository) Complete(ctx context.Context, ids []string) ([]models.Order, map[string]string, error) {
	if len(ids) == 0 {
		return []models.Order{}, map[string]string{}, nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,status FROM "order" WHERE id=ANY($1) AND deleted_at IS NULL FOR UPDATE`, ids)
	if err != nil {
		return nil, nil, err
	}
	previous := map[string]string{}
	for rows.Next() {
		var id, status string
		if err = rows.Scan(&id, &status); err != nil {
			rows.Close()
			return nil, nil, err
		}
		previous[id] = status
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	if len(previous) != len(ids) {
		return nil, nil, pgx.ErrNoRows
	}
	blocked := []string{}
	for _, id := range ids {
		if previous[id] == "canceled" {
			blocked = append(blocked, id)
		}
	}
	if len(blocked) > 0 {
		return nil, nil, domain.InvalidData(fmt.Sprintf("Orders %s are canceled and cannot be completed", strings.Join(blocked, ", ")))
	}
	if _, err = tx.Exec(ctx, `UPDATE "order" SET status='completed',updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, ids); err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	out := make([]models.Order, 0, len(ids))
	for _, id := range ids {
		row, e := r.Get(ctx, id, nil)
		if e != nil {
			return nil, nil, e
		}
		out = append(out, row)
	}
	return out, previous, nil
}

func (r *Repository) RestoreStatuses(ctx context.Context, statuses map[string]string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for id, status := range statuses {
		if _, err = tx.Exec(ctx, `UPDATE "order" SET status=$2,updated_at=now() WHERE id=$1`, id, status); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeleteDraftOrders(ctx context.Context, ids []string) error {
	unique := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	if len(unique) == 0 {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,status,is_draft_order FROM "order" WHERE id=ANY($1) AND deleted_at IS NULL FOR UPDATE`, unique)
	if err != nil {
		return err
	}
	found := map[string]bool{}
	for rows.Next() {
		var id, status string
		var draft bool
		if err = rows.Scan(&id, &status, &draft); err != nil {
			rows.Close()
			return err
		}
		if status != "draft" || !draft {
			rows.Close()
			return domain.InvalidData("One or more orders are not draft")
		}
		found[id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(found) != len(unique) {
		return pgx.ErrNoRows
	}
	for _, table := range []string{"order_cart", "order_fulfillment", "order_payment_collection", "order_promotion"} {
		if _, err = tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET deleted_at=now(),updated_at=now() WHERE order_id=ANY($1) AND deleted_at IS NULL`, table), unique); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE "order" SET deleted_at=now(),updated_at=now() WHERE id=ANY($1) AND deleted_at IS NULL`, unique); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Update(ctx context.Context, in models.UpdateInput) (models.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Order{}, err
	}
	defer tx.Rollback(ctx)
	var status string
	var version int
	var orderEmail, locale *string
	var customerID *string
	var metadata []byte
	var shippingID, billingID *string
	err = tx.QueryRow(ctx, `SELECT status,version,email,locale,metadata,shipping_address_id,billing_address_id,customer_id FROM "order" WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, in.ID).Scan(&status, &version, &orderEmail, &locale, &metadata, &shippingID, &billingID, &customerID)
	if err != nil {
		return models.Order{}, err
	}
	if status == "canceled" {
		return models.Order{}, domain.InvalidData(fmt.Sprintf("Order with id %s has been canceled.", in.ID))
	}
	sets := []string{}
	args := []any{in.ID}
	changes := []struct {
		kind string
		old  any
		new  any
	}{}
	add := func(column, kind string, old, new any) {
		args = append(args, new)
		sets = append(sets, fmt.Sprintf("%s=$%d", column, len(args)))
		changes = append(changes, struct {
			kind string
			old  any
			new  any
		}{kind, old, new})
	}
	if in.Email != nil {
		args = append(args, *in.Email)
		sets = append(sets, fmt.Sprintf("email=$%d", len(args)))
		if *in.Email != "" {
			changes = append(changes, struct {
				kind string
				old  any
				new  any
			}{"email", orderEmail, *in.Email})
		}
	}
	if in.Locale != nil && *in.Locale != value(locale) {
		add("locale", "locale", locale, *in.Locale)
	}
	if in.Metadata != nil {
		add("metadata", "metadata", jsonValue(metadata), jsonValue(*in.Metadata))
	}
	if in.CustomerID != nil {
		args = append(args, *in.CustomerID)
		sets = append(sets, fmt.Sprintf("customer_id=$%d", len(args)))
	}
	if in.ShippingAddress != nil {
		old := loadAddressTx(ctx, tx, shippingID)
		a, err := mergeAddress(ctx, tx, shippingID, in.ShippingAddress)
		if err != nil {
			return models.Order{}, err
		}
		if a.CustomerID == nil {
			a.CustomerID = customerID
		}
		id, err := insertOrderAddress(ctx, tx, a)
		if err != nil {
			return models.Order{}, err
		}
		args = append(args, id)
		sets = append(sets, fmt.Sprintf("shipping_address_id=$%d", len(args)))
		changes = append(changes, struct {
			kind string
			old  any
			new  any
		}{"shipping_address", old, a})
	}
	if in.BillingAddress != nil {
		old := loadAddressTx(ctx, tx, billingID)
		a, err := mergeAddress(ctx, tx, billingID, in.BillingAddress)
		if err != nil {
			return models.Order{}, err
		}
		if a.CustomerID == nil {
			a.CustomerID = customerID
		}
		id, err := insertOrderAddress(ctx, tx, a)
		if err != nil {
			return models.Order{}, err
		}
		args = append(args, id)
		sets = append(sets, fmt.Sprintf("billing_address_id=$%d", len(args)))
		changes = append(changes, struct {
			kind string
			old  any
			new  any
		}{"billing_address", old, a})
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		if _, err = tx.Exec(ctx, `UPDATE "order" SET `+strings.Join(sets, ",")+` WHERE id=$1`, args...); err != nil {
			return models.Order{}, err
		}
	}
	for _, change := range changes {
		details, marshalErr := json.Marshal(map[string]any{"type": change.kind, "old": change.old, "new": change.new})
		if marshalErr != nil {
			return models.Order{}, marshalErr
		}
		changeID := entityid.New("ordch")
		actionID := entityid.New("ordchact")
		if _, err = tx.Exec(ctx, `INSERT INTO order_change(id,order_id,version,change_type,status,created_by,confirmed_by,confirmed_at) VALUES($1,$2,$3,'update_order','confirmed',$4,$4,now())`, changeID, in.ID, version, in.UserID); err != nil {
			return models.Order{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO order_change_action(id,order_id,order_change_id,version,action,details,applied) VALUES($1,$2,$3,$4,'update_order_properties',$5,true)`, actionID, in.ID, changeID, version, details); err != nil {
			return models.Order{}, err
		}
	}
	for _, item := range in.ItemTranslations {
		if _, err = tx.Exec(ctx, `UPDATE order_line_item SET title=$2,subtitle=$3,product_title=$4,product_description=$5,product_subtitle=$6,product_type=$7,product_collection=$8,product_handle=$9,variant_title=$10,updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, item.ID, item.Title, item.Subtitle, item.ProductTitle, item.ProductDescription, item.ProductSubtitle, item.ProductType, item.ProductCollection, item.ProductHandle, item.VariantTitle); err != nil {
			return models.Order{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Order{}, err
	}
	return r.Get(ctx, in.ID, nil)
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func jsonValue(v []byte) any {
	if len(v) == 0 {
		return nil
	}
	var out any
	if json.Unmarshal(v, &out) != nil {
		return string(v)
	}
	return out
}
func mergeAddress(ctx context.Context, tx pgx.Tx, id *string, patch *models.AddressUpdate) (models.Address, error) {
	a := models.Address{}
	if id != nil {
		err := tx.QueryRow(ctx, `SELECT id,customer_id,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata FROM order_address WHERE id=$1 AND deleted_at IS NULL`, *id).Scan(&a.ID, &a.CustomerID, &a.Company, &a.FirstName, &a.LastName, &a.Address1, &a.Address2, &a.City, &a.CountryCode, &a.Province, &a.PostalCode, &a.Phone, &a.Metadata)
		if err != nil {
			return a, err
		}
	}
	if patch.CountryCode != nil && *patch.CountryCode != "" && (a.CountryCode == nil || *patch.CountryCode != *a.CountryCode) {
		return a, domain.InvalidData("Country code cannot be changed")
	}
	if patch.Company != nil {
		a.Company = patch.Company
	}
	if patch.FirstName != nil {
		a.FirstName = patch.FirstName
	}
	if patch.LastName != nil {
		a.LastName = patch.LastName
	}
	if patch.Address1 != nil {
		a.Address1 = patch.Address1
	}
	if patch.Address2 != nil {
		a.Address2 = patch.Address2
	}
	if patch.City != nil {
		a.City = patch.City
	}
	if patch.CountryCode != nil {
		a.CountryCode = patch.CountryCode
	}
	if patch.Province != nil {
		a.Province = patch.Province
	}
	if patch.PostalCode != nil {
		a.PostalCode = patch.PostalCode
	}
	if patch.Phone != nil {
		a.Phone = patch.Phone
	}
	if patch.Metadata != nil {
		a.Metadata = *patch.Metadata
	}
	return a, nil
}
func loadAddressTx(ctx context.Context, tx pgx.Tx, id *string) any {
	if id == nil {
		return nil
	}
	var a models.Address
	err := tx.QueryRow(ctx, `SELECT id,customer_id,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata FROM order_address WHERE id=$1`, *id).Scan(&a.ID, &a.CustomerID, &a.Company, &a.FirstName, &a.LastName, &a.Address1, &a.Address2, &a.City, &a.CountryCode, &a.Province, &a.PostalCode, &a.Phone, &a.Metadata)
	if err != nil {
		return nil
	}
	return a
}
func insertOrderAddress(ctx context.Context, tx pgx.Tx, a models.Address) (string, error) {
	id := entityid.New("ordaddr")
	if _, err := tx.Exec(ctx, `INSERT INTO order_address(id,customer_id,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, id, a.CustomerID, a.Company, a.FirstName, a.LastName, a.Address1, a.Address2, a.City, a.CountryCode, a.Province, a.PostalCode, a.Phone, a.Metadata); err != nil {
		return "", err
	}
	return id, nil
}
func (r *Repository) populate(ctx context.Context, v *models.Order, addresses ...*string) error {
	if len(addresses) > 0 && addresses[0] != nil {
		a, e := r.getAddress(ctx, *addresses[0])
		if e != nil {
			return e
		}
		v.ShippingAddress = &a
	}
	if len(addresses) > 1 && addresses[1] != nil {
		a, e := r.getAddress(ctx, *addresses[1])
		if e != nil {
			return e
		}
		v.BillingAddress = &a
	}
	rows, err := r.db.Query(ctx, `SELECT oi.id,oi.version,oi.raw_unit_price,oi.raw_compare_at_unit_price,oi.raw_quantity,oi.raw_fulfilled_quantity,oi.raw_delivered_quantity,oi.raw_shipped_quantity,oi.raw_return_requested_quantity,oi.raw_return_received_quantity,oi.raw_return_dismissed_quantity,oi.raw_written_off_quantity,oi.metadata,li.id,li.title,li.subtitle,li.thumbnail,li.variant_id,li.product_id,li.product_title,li.product_description,li.product_subtitle,li.product_type,li.product_type_id,li.product_collection,li.product_handle,li.variant_sku,li.variant_barcode,li.variant_title,li.variant_option_values,li.requires_shipping,li.is_giftcard,li.is_discountable,li.is_tax_inclusive,li.raw_compare_at_unit_price,li.raw_unit_price,li.is_custom_price,li.metadata FROM order_item oi JOIN order_line_item li ON li.id=oi.item_id AND li.deleted_at IS NULL WHERE oi.order_id=$1 AND oi.version=$2 AND oi.deleted_at IS NULL ORDER BY oi.created_at`, v.ID, v.Version)
	if err != nil {
		return err
	}
	defer rows.Close()
	v.Items = []models.Item{}
	for rows.Next() {
		var i models.Item
		var x models.LineItem
		err = rows.Scan(&i.ID, &i.Version, &i.UnitPrice, &i.CompareAtUnitPrice, &i.Quantity, &i.FulfilledQuantity, &i.DeliveredQuantity, &i.ShippedQuantity, &i.ReturnRequestedQuantity, &i.ReturnReceivedQuantity, &i.ReturnDismissedQuantity, &i.WrittenOffQuantity, &i.Metadata, &x.ID, &x.Title, &x.Subtitle, &x.Thumbnail, &x.VariantID, &x.ProductID, &x.ProductTitle, &x.ProductDescription, &x.ProductSubtitle, &x.ProductType, &x.ProductTypeID, &x.ProductCollection, &x.ProductHandle, &x.VariantSKU, &x.VariantBarcode, &x.VariantTitle, &x.VariantOptionValues, &x.RequiresShipping, &x.IsGiftcard, &x.IsDiscountable, &x.IsTaxInclusive, &x.CompareAtUnitPrice, &x.UnitPrice, &x.IsCustomPrice, &x.Metadata)
		if err != nil {
			return err
		}
		i.Item = x
		v.Items = append(v.Items, i)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return r.db.QueryRow(ctx, `SELECT totals FROM order_summary WHERE order_id=$1 AND version=$2 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`, v.ID, v.Version).Scan(&v.Summary)
}
func (r *Repository) getAddress(ctx context.Context, id string) (models.Address, error) {
	var a models.Address
	err := r.db.QueryRow(ctx, `SELECT id,customer_id,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata FROM order_address WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&a.ID, &a.CustomerID, &a.Company, &a.FirstName, &a.LastName, &a.Address1, &a.Address2, &a.City, &a.CountryCode, &a.Province, &a.PostalCode, &a.Phone, &a.Metadata)
	return a, err
}

var _ domain.Repository = (*Repository)(nil)
