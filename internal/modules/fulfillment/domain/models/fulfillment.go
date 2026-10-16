package models

import (
	"encoding/json"
	"time"
)

type FulfillmentAddress struct {
	ID          string          `json:"id"`
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

type FulfillmentItem struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	SKU             string          `json:"sku"`
	Barcode         string          `json:"barcode"`
	Quantity        json.RawMessage `json:"quantity"`
	LineItemID      *string         `json:"line_item_id"`
	InventoryItemID *string         `json:"inventory_item_id"`
}

type FulfillmentLabel struct {
	ID             string `json:"id"`
	TrackingNumber string `json:"tracking_number"`
	TrackingURL    string `json:"tracking_url"`
	LabelURL       string `json:"label_url"`
}

type FulfillmentAddressInput struct {
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

type FulfillmentItemInput struct {
	Title           string          `json:"title"`
	SKU             string          `json:"sku"`
	Barcode         string          `json:"barcode"`
	Quantity        json.RawMessage `json:"quantity"`
	LineItemID      *string         `json:"line_item_id"`
	InventoryItemID *string         `json:"inventory_item_id"`
}

type FulfillmentLabelInput struct {
	TrackingNumber string `json:"tracking_number"`
	TrackingURL    string `json:"tracking_url"`
	LabelURL       string `json:"label_url"`
}

type FulfillmentRecord struct {
	ID               string              `json:"id"`
	LocationID       string              `json:"location_id"`
	PackedAt         *time.Time          `json:"packed_at"`
	ShippedAt        *time.Time          `json:"shipped_at"`
	MarkedShippedBy  *string             `json:"marked_shipped_by"`
	CreatedBy        *string             `json:"created_by"`
	DeliveredAt      *time.Time          `json:"delivered_at"`
	CanceledAt       *time.Time          `json:"canceled_at"`
	Data             json.RawMessage     `json:"data"`
	RequiresShipping bool                `json:"requires_shipping"`
	ProviderID       *string             `json:"provider_id"`
	ShippingOptionID *string             `json:"shipping_option_id"`
	Metadata         json.RawMessage     `json:"metadata"`
	DeliveryAddress  *FulfillmentAddress `json:"delivery_address"`
	Items            []FulfillmentItem   `json:"items"`
	Labels           []FulfillmentLabel  `json:"labels"`
	OrderID          *string             `json:"order_id,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	DeletedAt        *time.Time          `json:"deleted_at"`
}

type FulfillmentCreateInput struct {
	LocationID       string                  `json:"location_id"`
	ProviderID       string                  `json:"provider_id"`
	DeliveryAddress  FulfillmentAddressInput `json:"delivery_address"`
	Items            []FulfillmentItemInput  `json:"items"`
	Labels           []FulfillmentLabelInput `json:"labels"`
	OrderID          string                  `json:"order_id"`
	ShippingOptionID *string                 `json:"shipping_option_id"`
	Data             json.RawMessage         `json:"data"`
	PackedAt         *time.Time              `json:"packed_at"`
	ShippedAt        *time.Time              `json:"shipped_at"`
	DeliveredAt      *time.Time              `json:"delivered_at"`
	CanceledAt       *time.Time              `json:"canceled_at"`
	Metadata         json.RawMessage         `json:"metadata"`
	CreatedBy        string                  `json:"-"`
	Location         map[string]any          `json:"-"`
}

type CreateShipmentInput struct {
	Labels          []FulfillmentLabelInput `json:"labels"`
	MarkedShippedBy string                  `json:"-"`
}
