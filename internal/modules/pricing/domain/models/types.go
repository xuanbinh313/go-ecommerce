package models

import (
	"encoding/json"
	"time"
)

type PricePreference struct {
	ID             string    `json:"id"`
	Attribute      string    `json:"attribute"`
	Value          *string   `json:"value"`
	IsTaxInclusive bool      `json:"is_tax_inclusive"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PreferenceInput struct {
	Attribute      string `json:"attribute"`
	Value          string `json:"value"`
	IsTaxInclusive *bool  `json:"is_tax_inclusive"`
}

type PreferenceUpdate struct {
	Attribute      *string `json:"attribute"`
	Value          *string `json:"value"`
	IsTaxInclusive *bool   `json:"is_tax_inclusive"`
}

type Price struct {
	ID           string          `json:"id"`
	Title        *string         `json:"title"`
	CurrencyCode string          `json:"currency_code"`
	Amount       json.RawMessage `json:"amount"`
	MinQuantity  json.RawMessage `json:"min_quantity,omitempty"`
	MaxQuantity  json.RawMessage `json:"max_quantity,omitempty"`
	PriceSetID   string          `json:"price_set_id"`
	PriceListID  *string         `json:"price_list_id,omitempty"`
}

type PriceSet struct {
	ID     string  `json:"id"`
	Prices []Price `json:"prices,omitempty"`
}

type PriceList struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Type        string          `json:"type"`
	StartsAt    *time.Time      `json:"starts_at"`
	EndsAt      *time.Time      `json:"ends_at"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
