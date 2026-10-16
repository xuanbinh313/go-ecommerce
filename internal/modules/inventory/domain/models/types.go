package models

import (
	"encoding/json"
	"time"
)

type InventoryItem struct {
	ID               string           `json:"id"`
	SKU              *string          `json:"sku"`
	OriginCountry    *string          `json:"origin_country"`
	HSCode           *string          `json:"hs_code"`
	MidCode          *string          `json:"mid_code"`
	Material         *string          `json:"material"`
	UnitOfMeasure    *string          `json:"unit_of_measure"`
	Weight           *int64           `json:"weight"`
	Length           *int64           `json:"length"`
	Height           *int64           `json:"height"`
	Width            *int64           `json:"width"`
	RequiresShipping bool             `json:"requires_shipping"`
	Description      *string          `json:"description"`
	Title            *string          `json:"title"`
	Thumbnail        *string          `json:"thumbnail"`
	Metadata         json.RawMessage  `json:"metadata"`
	ReservedQuantity float64          `json:"reserved_quantity"`
	StockedQuantity  float64          `json:"stocked_quantity"`
	LocationLevels   []InventoryLevel `json:"location_levels,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}
type InventoryLevel struct {
	ID                string          `json:"id"`
	InventoryItemID   string          `json:"inventory_item_id"`
	LocationID        string          `json:"location_id"`
	StockedQuantity   float64         `json:"stocked_quantity"`
	ReservedQuantity  float64         `json:"reserved_quantity"`
	IncomingQuantity  float64         `json:"incoming_quantity"`
	AvailableQuantity float64         `json:"available_quantity"`
	Metadata          json.RawMessage `json:"metadata"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}
type ItemInput struct {
	SKU              *string         `json:"sku"`
	OriginCountry    *string         `json:"origin_country"`
	HSCode           *string         `json:"hs_code"`
	MidCode          *string         `json:"mid_code"`
	Material         *string         `json:"material"`
	UnitOfMeasure    *string         `json:"unit_of_measure"`
	Weight           *int64          `json:"weight"`
	Length           *int64          `json:"length"`
	Height           *int64          `json:"height"`
	Width            *int64          `json:"width"`
	RequiresShipping *bool           `json:"requires_shipping"`
	Description      *string         `json:"description"`
	Title            *string         `json:"title"`
	Thumbnail        *string         `json:"thumbnail"`
	Metadata         json.RawMessage `json:"metadata"`
	LocationLevels   []LevelInput    `json:"location_levels,omitempty"`
}
type UpdateItemInput struct {
	SKU              *string         `json:"sku"`
	OriginCountry    *string         `json:"origin_country"`
	HSCode           *string         `json:"hs_code"`
	MidCode          *string         `json:"mid_code"`
	Material         *string         `json:"material"`
	UnitOfMeasure    *string         `json:"unit_of_measure"`
	Weight           *int64          `json:"weight"`
	Length           *int64          `json:"length"`
	Height           *int64          `json:"height"`
	Width            *int64          `json:"width"`
	RequiresShipping *bool           `json:"requires_shipping"`
	Description      *string         `json:"description"`
	Title            *string         `json:"title"`
	Thumbnail        *string         `json:"thumbnail"`
	Metadata         json.RawMessage `json:"metadata"`
}
type LevelInput struct {
	ID               string          `json:"id,omitempty"`
	InventoryItemID  string          `json:"inventory_item_id,omitempty"`
	LocationID       string          `json:"location_id"`
	StockedQuantity  *float64        `json:"stocked_quantity"`
	IncomingQuantity *float64        `json:"incoming_quantity"`
	Metadata         json.RawMessage `json:"metadata"`
}
type LevelUpdate struct {
	ID               string          `json:"id,omitempty"`
	InventoryItemID  string          `json:"inventory_item_id,omitempty"`
	LocationID       string          `json:"location_id,omitempty"`
	StockedQuantity  *float64        `json:"stocked_quantity"`
	IncomingQuantity *float64        `json:"incoming_quantity"`
	Metadata         json.RawMessage `json:"metadata"`
}
type Filter struct {
	IDs, SKUs, OriginCountries, MidCodes, HSCodes, Materials, LocationIDs []string
	RequiresShipping                                                      *bool
	Query                                                                 string
	Limit, Offset                                                         int
}
type BatchLevels struct {
	Create []LevelInput  `json:"create"`
	Update []LevelUpdate `json:"update"`
	Delete []string      `json:"delete"`
	Force  bool          `json:"force"`
}

type ReservationItem struct {
	ID              string          `json:"id"`
	InventoryItemID string          `json:"inventory_item_id"`
	LineItemID      *string         `json:"line_item_id"`
	AllowBackorder  bool            `json:"allow_backorder"`
	LocationID      string          `json:"location_id"`
	Quantity        float64         `json:"quantity"`
	RawQuantity     json.RawMessage `json:"raw_quantity"`
	ExternalID      *string         `json:"external_id"`
	Description     *string         `json:"description"`
	CreatedBy       *string         `json:"created_by"`
	Metadata        json.RawMessage `json:"metadata"`
	InventoryItem   *InventoryItem  `json:"inventory_item,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type ReservationInput struct {
	InventoryItemID string          `json:"inventory_item_id"`
	LineItemID      *string         `json:"line_item_id"`
	LocationID      string          `json:"location_id"`
	Quantity        float64         `json:"quantity"`
	Description     *string         `json:"description"`
	Metadata        json.RawMessage `json:"metadata"`
}

type ReservationUpdate struct {
	ID          string          `json:"id"`
	LocationID  *string         `json:"location_id"`
	Quantity    *float64        `json:"quantity"`
	Description json.RawMessage `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
}

type ReservationFilter struct {
	IDs, InventoryItemIDs, LocationIDs, LineItemIDs, CreatedBy []string
	Query, Description                                         string
	Limit, Offset                                              int
}
