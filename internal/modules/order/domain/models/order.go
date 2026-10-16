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

type LineItem struct {
	ID                  string          `json:"id"`
	Title               string          `json:"title"`
	Subtitle            *string         `json:"subtitle"`
	Thumbnail           *string         `json:"thumbnail"`
	VariantID           *string         `json:"variant_id"`
	ProductID           *string         `json:"product_id"`
	ProductTitle        *string         `json:"product_title"`
	ProductDescription  *string         `json:"product_description"`
	ProductSubtitle     *string         `json:"product_subtitle"`
	ProductType         *string         `json:"product_type"`
	ProductTypeID       *string         `json:"product_type_id"`
	ProductCollection   *string         `json:"product_collection"`
	ProductHandle       *string         `json:"product_handle"`
	VariantSKU          *string         `json:"variant_sku"`
	VariantBarcode      *string         `json:"variant_barcode"`
	VariantTitle        *string         `json:"variant_title"`
	VariantOptionValues json.RawMessage `json:"variant_option_values"`
	RequiresShipping    bool            `json:"requires_shipping"`
	IsGiftcard          bool            `json:"is_giftcard"`
	IsDiscountable      bool            `json:"is_discountable"`
	IsTaxInclusive      bool            `json:"is_tax_inclusive"`
	CompareAtUnitPrice  json.RawMessage `json:"compare_at_unit_price"`
	UnitPrice           json.RawMessage `json:"unit_price"`
	IsCustomPrice       bool            `json:"is_custom_price"`
	Metadata            json.RawMessage `json:"metadata"`
}

type Item struct {
	ID                      string          `json:"id"`
	Version                 int             `json:"version"`
	UnitPrice               json.RawMessage `json:"unit_price"`
	CompareAtUnitPrice      json.RawMessage `json:"compare_at_unit_price"`
	Quantity                json.RawMessage `json:"quantity"`
	FulfilledQuantity       json.RawMessage `json:"fulfilled_quantity"`
	DeliveredQuantity       json.RawMessage `json:"delivered_quantity"`
	ShippedQuantity         json.RawMessage `json:"shipped_quantity"`
	ReturnRequestedQuantity json.RawMessage `json:"return_requested_quantity"`
	ReturnReceivedQuantity  json.RawMessage `json:"return_received_quantity"`
	ReturnDismissedQuantity json.RawMessage `json:"return_dismissed_quantity"`
	WrittenOffQuantity      json.RawMessage `json:"written_off_quantity"`
	Metadata                json.RawMessage `json:"metadata"`
	Item                    LineItem        `json:"item"`
}

type Order struct {
	ID              string          `json:"id"`
	DisplayID       int64           `json:"display_id"`
	CustomDisplayID *string         `json:"custom_display_id"`
	RegionID        *string         `json:"region_id"`
	CustomerID      *string         `json:"customer_id"`
	Version         int             `json:"version"`
	SalesChannelID  *string         `json:"sales_channel_id"`
	Status          string          `json:"status"`
	IsDraftOrder    bool            `json:"is_draft_order"`
	Email           *string         `json:"email"`
	CurrencyCode    string          `json:"currency_code"`
	Locale          *string         `json:"locale"`
	NoNotification  *bool           `json:"no_notification"`
	Metadata        json.RawMessage `json:"metadata"`
	CanceledAt      *time.Time      `json:"canceled_at"`
	ShippingAddress *Address        `json:"shipping_address,omitempty"`
	BillingAddress  *Address        `json:"billing_address,omitempty"`
	Items           []Item          `json:"items,omitempty"`
	Summary         json.RawMessage `json:"summary,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Filter struct {
	IDs             []string
	CustomerIDs     []string
	RegionIDs       []string
	SalesChannelIDs []string
	CustomerID      *string
	RegionID        *string
	SalesChannelID  *string
	Status          []string
	Query           string
	CreatedAt       map[string]string
	UpdatedAt       map[string]string
	IsDraft         *bool
	Limit           int
	Offset          int
	Order           string
}

type AddressUpdate struct {
	Company     *string          `json:"company"`
	FirstName   *string          `json:"first_name"`
	LastName    *string          `json:"last_name"`
	Address1    *string          `json:"address_1"`
	Address2    *string          `json:"address_2"`
	City        *string          `json:"city"`
	CountryCode *string          `json:"country_code"`
	Province    *string          `json:"province"`
	PostalCode  *string          `json:"postal_code"`
	Phone       *string          `json:"phone"`
	Metadata    *json.RawMessage `json:"metadata"`
}

type UpdateInput struct {
	ID               string                      `json:"id"`
	UserID           string                      `json:"user_id"`
	Email            *string                     `json:"email"`
	Locale           *string                     `json:"locale"`
	Metadata         *json.RawMessage            `json:"metadata"`
	CustomerID       *string                     `json:"-"`
	ShippingAddress  *AddressUpdate              `json:"shipping_address"`
	BillingAddress   *AddressUpdate              `json:"billing_address"`
	ItemTranslations []LineItemTranslationUpdate `json:"-"`
}

type LineItemTranslationUpdate struct {
	ID                 string
	Title              string
	Subtitle           *string
	ProductTitle       *string
	ProductDescription *string
	ProductSubtitle    *string
	ProductType        *string
	ProductCollection  *string
	ProductHandle      *string
	VariantTitle       *string
}

type ListResult struct {
	Orders []Order
	Count  int
}
