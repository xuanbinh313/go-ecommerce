package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/example/medusa-go-clone/internal/modules/cart/domain"
	"github.com/example/medusa-go-clone/internal/modules/cart/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const cartCols = `id,region_id,customer_id,sales_channel_id,email,currency_code,locale,shipping_address_id,billing_address_id,metadata,completed_at,created_at,updated_at`

func scanCart(row pgx.Row) (models.Cart, *string, *string, error) {
	var c models.Cart
	var shipping, billing *string
	err := row.Scan(&c.ID, &c.RegionID, &c.CustomerID, &c.SalesChannelID, &c.Email, &c.CurrencyCode, &c.Locale, &shipping, &billing, &c.Metadata, &c.CompletedAt, &c.CreatedAt, &c.UpdatedAt)
	return c, shipping, billing, err
}
func (r *Repository) Get(ctx context.Context, id string) (models.Cart, error) {
	c, shippingID, billingID, err := scanCart(r.db.QueryRow(ctx, "SELECT "+cartCols+" FROM cart WHERE id=$1 AND deleted_at IS NULL", id))
	if err != nil {
		return c, err
	}
	if shippingID != nil {
		c.ShippingAddress, err = r.getAddress(ctx, *shippingID)
		if err != nil {
			return c, err
		}
	}
	if billingID != nil {
		c.BillingAddress, err = r.getAddress(ctx, *billingID)
		if err != nil {
			return c, err
		}
	}
	c.Items, err = r.listCartItems(ctx, id)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (r *Repository) listCartItems(ctx context.Context, cartID string) ([]models.LineItem, error) {
	rows, err := r.db.Query(ctx, `SELECT id,cart_id,title,subtitle,thumbnail,quantity,variant_id,product_id,product_title,product_description,product_subtitle,product_type,product_type_id,product_collection,product_handle,variant_sku,variant_barcode,variant_title,variant_option_values,requires_shipping,is_discountable,is_giftcard,is_tax_inclusive,is_custom_price,unit_price::text,raw_unit_price,compare_at_unit_price::text,raw_compare_at_unit_price,metadata,created_at,updated_at,deleted_at FROM cart_line_item WHERE cart_id=$1 AND deleted_at IS NULL ORDER BY created_at,id`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.LineItem{}
	for rows.Next() {
		var item models.LineItem
		if err := rows.Scan(&item.ID, &item.CartID, &item.Title, &item.Subtitle, &item.Thumbnail, &item.Quantity, &item.VariantID, &item.ProductID, &item.ProductTitle, &item.ProductDescription, &item.ProductSubtitle, &item.ProductType, &item.ProductTypeID, &item.ProductCollection, &item.ProductHandle, &item.VariantSKU, &item.VariantBarcode, &item.VariantTitle, &item.VariantOptionValues, &item.RequiresShipping, &item.IsDiscountable, &item.IsGiftcard, &item.IsTaxInclusive, &item.IsCustomPrice, &item.UnitPrice, &item.RawUnitPrice, &item.CompareAtUnitPrice, &item.RawCompareAtUnitPrice, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateLineItems(ctx context.Context, cartID string, inputs []models.LineItemInput) ([]models.LineItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cart WHERE id=$1 AND deleted_at IS NULL)`, cartID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, pgx.ErrNoRows
	}
	ids := make([]string, 0, len(inputs))
	for _, in := range inputs {
		id := entityid.New("cali")
		requiresShipping, discountable, giftcard, taxInclusive, customPrice := true, true, false, false, false
		if in.RequiresShipping != nil {
			requiresShipping = *in.RequiresShipping
		}
		if in.IsDiscountable != nil {
			discountable = *in.IsDiscountable
		}
		if in.IsGiftcard != nil {
			giftcard = *in.IsGiftcard
		}
		if in.IsTaxInclusive != nil {
			taxInclusive = *in.IsTaxInclusive
		}
		if in.IsCustomPrice != nil {
			customPrice = *in.IsCustomPrice
		}
		metadata := in.Metadata
		if len(metadata) == 0 {
			metadata = []byte("null")
		}
		variantValues := in.VariantOptionValues
		if len(variantValues) == 0 {
			variantValues = []byte("null")
		}
		rawCompare := in.RawCompareAtUnitPrice
		if len(rawCompare) == 0 {
			rawCompare = []byte("null")
		}
		_, err = tx.Exec(ctx, `INSERT INTO cart_line_item(id,cart_id,title,subtitle,thumbnail,quantity,variant_id,product_id,product_title,product_description,product_subtitle,product_type,product_type_id,product_collection,product_handle,variant_sku,variant_barcode,variant_title,variant_option_values,requires_shipping,is_discountable,is_giftcard,is_tax_inclusive,is_custom_price,unit_price,raw_unit_price,compare_at_unit_price,raw_compare_at_unit_price,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)`, id, cartID, in.Title, in.Subtitle, in.Thumbnail, in.Quantity, in.VariantID, in.ProductID, in.ProductTitle, in.ProductDescription, in.ProductSubtitle, in.ProductType, in.ProductTypeID, in.ProductCollection, in.ProductHandle, in.VariantSKU, in.VariantBarcode, in.VariantTitle, variantValues, requiresShipping, discountable, giftcard, taxInclusive, customPrice, in.UnitPrice, in.RawUnitPrice, in.CompareAtUnitPrice, rawCompare, metadata)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	all, err := r.listCartItems(ctx, cartID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]models.LineItem, len(all))
	for _, item := range all {
		byID[item.ID] = item
	}
	created := make([]models.LineItem, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			created = append(created, item)
		}
	}
	return created, nil
}

func (r *Repository) GetLineItem(ctx context.Context, cartID, itemID string) (models.LineItem, error) {
	items, err := r.listCartItems(ctx, cartID)
	if err != nil {
		return models.LineItem{}, err
	}
	for _, item := range items {
		if item.ID == itemID {
			return item, nil
		}
	}
	return models.LineItem{}, pgx.ErrNoRows
}

func (r *Repository) UpdateLineItem(ctx context.Context, cartID, itemID string, in models.LineItemUpdate) (models.LineItem, error) {
	sets, args := []string{}, []any{itemID, cartID}
	if in.Quantity != nil {
		args = append(args, *in.Quantity)
		sets = append(sets, fmt.Sprintf("quantity=$%d", len(args)))
	}
	if in.Metadata != nil || in.MetadataSet {
		args = append(args, in.Metadata)
		sets = append(sets, fmt.Sprintf("metadata=$%d", len(args)))
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := r.db.Exec(ctx, "UPDATE cart_line_item SET "+join(sets)+" WHERE id=$1 AND cart_id=$2 AND deleted_at IS NULL", args...)
		if err != nil {
			return models.LineItem{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.LineItem{}, pgx.ErrNoRows
		}
	}
	items, err := r.listCartItems(ctx, cartID)
	if err != nil {
		return models.LineItem{}, err
	}
	for _, item := range items {
		if item.ID == itemID {
			return item, nil
		}
	}
	return models.LineItem{}, pgx.ErrNoRows
}

func (r *Repository) SoftDeleteLineItems(ctx context.Context, cartID string, ids []string) ([]models.LineItem, error) {
	if len(ids) == 0 {
		return []models.LineItem{}, nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,cart_id,title,subtitle,thumbnail,quantity,variant_id,product_id,product_title,product_description,product_subtitle,product_type,product_type_id,product_collection,product_handle,variant_sku,variant_barcode,variant_title,variant_option_values,requires_shipping,is_discountable,is_giftcard,is_tax_inclusive,is_custom_price,unit_price::text,raw_unit_price,compare_at_unit_price::text,raw_compare_at_unit_price,metadata,created_at,updated_at,deleted_at FROM cart_line_item WHERE cart_id=$1 AND id=ANY($2::text[]) AND deleted_at IS NULL FOR UPDATE`, cartID, ids)
	if err != nil {
		return nil, err
	}
	before := []models.LineItem{}
	for rows.Next() {
		var item models.LineItem
		if err = rows.Scan(&item.ID, &item.CartID, &item.Title, &item.Subtitle, &item.Thumbnail, &item.Quantity, &item.VariantID, &item.ProductID, &item.ProductTitle, &item.ProductDescription, &item.ProductSubtitle, &item.ProductType, &item.ProductTypeID, &item.ProductCollection, &item.ProductHandle, &item.VariantSKU, &item.VariantBarcode, &item.VariantTitle, &item.VariantOptionValues, &item.RequiresShipping, &item.IsDiscountable, &item.IsGiftcard, &item.IsTaxInclusive, &item.IsCustomPrice, &item.UnitPrice, &item.RawUnitPrice, &item.CompareAtUnitPrice, &item.RawCompareAtUnitPrice, &item.Metadata, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			rows.Close()
			return nil, err
		}
		before = append(before, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(before) != len(ids) {
		return nil, pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, `UPDATE cart_line_item SET deleted_at=now(),updated_at=now() WHERE cart_id=$1 AND id=ANY($2::text[]) AND deleted_at IS NULL`, cartID, ids); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return before, nil
}

func (r *Repository) RestoreLineItems(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `UPDATE cart_line_item SET deleted_at=NULL,updated_at=now() WHERE id=ANY($1::text[])`, ids)
	return err
}
func (r *Repository) getAddress(ctx context.Context, id string) (*models.Address, error) {
	var a models.Address
	err := r.db.QueryRow(ctx, `SELECT id,customer_id,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata FROM cart_address WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&a.ID, &a.CustomerID, &a.Company, &a.FirstName, &a.LastName, &a.Address1, &a.Address2, &a.City, &a.CountryCode, &a.Province, &a.PostalCode, &a.Phone, &a.Metadata)
	return &a, err
}
func (r *Repository) Create(ctx context.Context, in models.CreateInput) (models.Cart, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Cart{}, err
	}
	defer tx.Rollback(ctx)
	shipping, err := insertAddress(ctx, tx, in.ShippingAddress, in.CustomerID)
	if err != nil {
		return models.Cart{}, err
	}
	billing, err := insertAddress(ctx, tx, in.BillingAddress, in.CustomerID)
	if err != nil {
		return models.Cart{}, err
	}
	metadata := in.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage("null")
	}
	id := entityid.New("cart")
	_, err = tx.Exec(ctx, `INSERT INTO cart(id,region_id,customer_id,sales_channel_id,email,currency_code,locale,shipping_address_id,billing_address_id,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, in.RegionID, in.CustomerID, in.SalesChannelID, in.Email, *in.CurrencyCode, in.Locale, shipping, billing, metadata)
	if err != nil {
		return models.Cart{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Cart{}, err
	}
	return r.Get(ctx, id)
}
func insertAddress(ctx context.Context, tx pgx.Tx, raw json.RawMessage, customer *string) (*string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var id string
	if err := json.Unmarshal(raw, &id); err == nil {
		if id == "" {
			return nil, nil
		}
		return &id, nil
	}
	var a models.Address
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("invalid cart address: %w", err)
	}
	if a.CustomerID == nil {
		a.CustomerID = customer
	}
	a.ID = entityid.New("caaddr")
	meta := a.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage("null")
	}
	_, err := tx.Exec(ctx, `INSERT INTO cart_address(id,customer_id,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, a.ID, a.CustomerID, a.Company, a.FirstName, a.LastName, a.Address1, a.Address2, a.City, a.CountryCode, a.Province, a.PostalCode, a.Phone, meta)
	return &a.ID, err
}
func (r *Repository) Update(ctx context.Context, id string, in models.UpdateInput) (models.Cart, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Cart{}, err
	}
	defer tx.Rollback(ctx)
	var cartCustomerID *string
	if err = tx.QueryRow(ctx, "SELECT customer_id FROM cart WHERE id=$1 AND deleted_at IS NULL", id).Scan(&cartCustomerID); err != nil {
		return models.Cart{}, err
	}
	sets := []string{}
	args := []any{id}
	put := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	if in.RegionID != nil {
		put("region_id", *in.RegionID)
	}
	if in.CurrencyCode != nil {
		put("currency_code", *in.CurrencyCode)
	}
	if in.CustomerID != nil || in.CustomerIDSet {
		put("customer_id", in.CustomerID)
	}
	if in.SalesChannelID != nil {
		put("sales_channel_id", *in.SalesChannelID)
	}
	if in.Email != nil || in.EmailSet {
		put("email", in.Email)
	}
	if in.Locale != nil {
		put("locale", *in.Locale)
	}
	if in.Metadata != nil {
		put("metadata", in.Metadata)
	}
	if in.ShippingAddress != nil {
		addressID, addressErr := updateCartAddress(ctx, tx, in.ShippingAddress, cartCustomerID)
		if addressErr != nil {
			return models.Cart{}, addressErr
		}
		put("shipping_address_id", addressID)
	}
	if in.BillingAddress != nil {
		addressID, addressErr := updateCartAddress(ctx, tx, in.BillingAddress, cartCustomerID)
		if addressErr != nil {
			return models.Cart{}, addressErr
		}
		put("billing_address_id", addressID)
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=now()")
		tag, err := tx.Exec(ctx, "UPDATE cart SET "+join(sets)+" WHERE id=$1 AND deleted_at IS NULL", args...)
		if err != nil {
			return models.Cart{}, err
		}
		if tag.RowsAffected() == 0 {
			return models.Cart{}, pgx.ErrNoRows
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return models.Cart{}, err
	}
	return r.Get(ctx, id)
}

func updateCartAddress(ctx context.Context, tx pgx.Tx, raw json.RawMessage, customerID *string) (*string, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	var addressID string
	if err := json.Unmarshal(raw, &addressID); err == nil {
		if addressID == "" {
			return nil, nil
		}
		return &addressID, nil
	}
	var address models.Address
	if err := json.Unmarshal(raw, &address); err != nil {
		return nil, fmt.Errorf("invalid cart address: %w", err)
	}
	if address.ID == "" {
		if address.CustomerID == nil {
			address.CustomerID = customerID
			encoded, err := json.Marshal(address)
			if err != nil {
				return nil, err
			}
			raw = encoded
		}
		return insertAddress(ctx, tx, raw, customerID)
	}
	updates := []string{}
	args := []any{address.ID}
	put := func(column string, value any) {
		args = append(args, value)
		updates = append(updates, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if address.CustomerID != nil {
		put("customer_id", address.CustomerID)
	}
	if address.Company != nil {
		put("company", address.Company)
	}
	if address.FirstName != nil {
		put("first_name", address.FirstName)
	}
	if address.LastName != nil {
		put("last_name", address.LastName)
	}
	if address.Address1 != nil {
		put("address_1", address.Address1)
	}
	if address.Address2 != nil {
		put("address_2", address.Address2)
	}
	if address.City != nil {
		put("city", address.City)
	}
	if address.CountryCode != nil {
		put("country_code", address.CountryCode)
	}
	if address.Province != nil {
		put("province", address.Province)
	}
	if address.PostalCode != nil {
		put("postal_code", address.PostalCode)
	}
	if address.Phone != nil {
		put("phone", address.Phone)
	}
	if address.Metadata != nil {
		put("metadata", address.Metadata)
	}
	if len(updates) > 0 {
		updates = append(updates, "updated_at=now()")
		if _, err := tx.Exec(ctx, "UPDATE cart_address SET "+join(updates)+" WHERE id=$1 AND deleted_at IS NULL", args...); err != nil {
			return nil, err
		}
	}
	return &address.ID, nil
}
func join(v []string) string {
	out := ""
	for i, s := range v {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

var _ domain.Repository = (*Repository)(nil)
