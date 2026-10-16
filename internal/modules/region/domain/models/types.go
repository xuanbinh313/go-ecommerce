package models

import (
	"encoding/json"
	"time"
)

type Region struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	CurrencyCode   string          `json:"currency_code"`
	AutomaticTaxes bool            `json:"automatic_taxes"`
	Countries      []Country       `json:"countries"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty"`
}

type Country struct {
	ISO2        string          `json:"iso_2"`
	ISO3        string          `json:"iso_3"`
	NumCode     string          `json:"num_code"`
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	RegionID    *string         `json:"region_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata"`
}

type RegionInput struct {
	ID               *string         `json:"id,omitempty"`
	Name             *string         `json:"name,omitempty"`
	CurrencyCode     *string         `json:"currency_code,omitempty"`
	AutomaticTaxes   *bool           `json:"automatic_taxes,omitempty"`
	Countries        []string        `json:"countries,omitempty"`
	PaymentProviders []string        `json:"payment_providers,omitempty"`
	IsTaxInclusive   *bool           `json:"is_tax_inclusive,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}
