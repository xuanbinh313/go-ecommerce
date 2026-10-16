package models

import (
	"encoding/json"
	"time"
)

type Address struct {
	ID          string          `json:"id"`
	CustomerID  *string         `json:"customer_id"`
	Company     *string         `json:"company"`
	FirstName   *string         `json:"first_name"`
	LastName    *string         `json:"last_name"`
	Address1    *string         `json:"address_1"`
	Address2    *string         `json:"address_2"`
	City        *string         `json:"city"`
	CountryCode *string         `json:"country_code"`
	Province    *string         `json:"province"`
	PostalCode  *string         `json:"postal_code"`
	Phone       *string         `json:"phone"`
	Metadata    json.RawMessage `json:"metadata"`
}

type Cart struct {
	ID              string          `json:"id"`
	RegionID        *string         `json:"region_id"`
	CustomerID      *string         `json:"customer_id"`
	SalesChannelID  *string         `json:"sales_channel_id"`
	Email           *string         `json:"email"`
	CurrencyCode    string          `json:"currency_code"`
	Locale          *string         `json:"locale"`
	ShippingAddress *Address        `json:"shipping_address,omitempty"`
	BillingAddress  *Address        `json:"billing_address,omitempty"`
	Items           []LineItem      `json:"items"`
	Metadata        json.RawMessage `json:"metadata"`
	CompletedAt     *time.Time      `json:"completed_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// LineItem preserves the persisted Medusa cart_line_item fields. Totals are
// calculated by cart workflows and are not stored on the line-item row.
type LineItem struct {
	ID                    string          `json:"id"`
	CartID                string          `json:"cart_id"`
	Title                 string          `json:"title"`
	Subtitle              *string         `json:"subtitle"`
	Thumbnail             *string         `json:"thumbnail"`
	Quantity              int             `json:"quantity"`
	VariantID             *string         `json:"variant_id"`
	ProductID             *string         `json:"product_id"`
	ProductTitle          *string         `json:"product_title"`
	ProductDescription    *string         `json:"product_description"`
	ProductSubtitle       *string         `json:"product_subtitle"`
	ProductType           *string         `json:"product_type"`
	ProductTypeID         *string         `json:"product_type_id"`
	ProductCollection     *string         `json:"product_collection"`
	ProductHandle         *string         `json:"product_handle"`
	VariantSKU            *string         `json:"variant_sku"`
	VariantBarcode        *string         `json:"variant_barcode"`
	VariantTitle          *string         `json:"variant_title"`
	VariantOptionValues   json.RawMessage `json:"variant_option_values"`
	RequiresShipping      bool            `json:"requires_shipping"`
	IsDiscountable        bool            `json:"is_discountable"`
	IsGiftcard            bool            `json:"is_giftcard"`
	IsTaxInclusive        bool            `json:"is_tax_inclusive"`
	IsCustomPrice         bool            `json:"is_custom_price"`
	UnitPrice             string          `json:"unit_price"`
	RawUnitPrice          json.RawMessage `json:"raw_unit_price"`
	CompareAtUnitPrice    *string         `json:"compare_at_unit_price"`
	RawCompareAtUnitPrice json.RawMessage `json:"raw_compare_at_unit_price"`
	Metadata              json.RawMessage `json:"metadata"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	DeletedAt             *time.Time      `json:"deleted_at,omitempty"`
}

type LineItemInput struct {
	Title                 string          `json:"title"`
	Subtitle              *string         `json:"subtitle,omitempty"`
	Thumbnail             *string         `json:"thumbnail,omitempty"`
	Quantity              int             `json:"quantity"`
	VariantID             *string         `json:"variant_id,omitempty"`
	ProductID             *string         `json:"product_id,omitempty"`
	ProductTitle          *string         `json:"product_title,omitempty"`
	ProductDescription    *string         `json:"product_description,omitempty"`
	ProductSubtitle       *string         `json:"product_subtitle,omitempty"`
	ProductType           *string         `json:"product_type,omitempty"`
	ProductTypeID         *string         `json:"product_type_id,omitempty"`
	ProductCollection     *string         `json:"product_collection,omitempty"`
	ProductHandle         *string         `json:"product_handle,omitempty"`
	VariantSKU            *string         `json:"variant_sku,omitempty"`
	VariantBarcode        *string         `json:"variant_barcode,omitempty"`
	VariantTitle          *string         `json:"variant_title,omitempty"`
	VariantOptionValues   json.RawMessage `json:"variant_option_values,omitempty"`
	RequiresShipping      *bool           `json:"requires_shipping,omitempty"`
	IsDiscountable        *bool           `json:"is_discountable,omitempty"`
	IsGiftcard            *bool           `json:"is_giftcard,omitempty"`
	IsTaxInclusive        *bool           `json:"is_tax_inclusive,omitempty"`
	IsCustomPrice         *bool           `json:"is_custom_price,omitempty"`
	UnitPrice             string          `json:"unit_price"`
	RawUnitPrice          json.RawMessage `json:"raw_unit_price"`
	CompareAtUnitPrice    *string         `json:"compare_at_unit_price,omitempty"`
	RawCompareAtUnitPrice json.RawMessage `json:"raw_compare_at_unit_price,omitempty"`
	Metadata              json.RawMessage `json:"metadata,omitempty"`
}

type LineItemUpdate struct {
	Quantity *int            `json:"quantity,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	MetadataSet bool         `json:"-"`
}

type CreateInput struct {
	RegionID        *string         `json:"region_id"`
	CustomerID      *string         `json:"-"`
	SalesChannelID  *string         `json:"sales_channel_id"`
	Email           *string         `json:"email"`
	CurrencyCode    *string         `json:"currency_code"`
	Locale          *string         `json:"locale"`
	Metadata        json.RawMessage `json:"metadata"`
	ShippingAddress json.RawMessage `json:"shipping_address"`
	BillingAddress  json.RawMessage `json:"billing_address"`
}

type UpdateInput struct {
	RegionID        *string         `json:"region_id"`
	CurrencyCode    *string         `json:"-"`
	CustomerID      *string         `json:"-"`
	CustomerIDSet   bool            `json:"-"`
	SalesChannelID  *string         `json:"sales_channel_id"`
	Email           *string         `json:"email"`
	EmailSet        bool            `json:"-"`
	ShippingAddress json.RawMessage `json:"shipping_address"`
	BillingAddress  json.RawMessage `json:"billing_address"`
	Locale          *string         `json:"locale"`
	Metadata        json.RawMessage `json:"metadata"`
}
