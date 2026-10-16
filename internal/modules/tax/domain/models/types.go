package models

import (
	"encoding/json"
	"time"
)

type TaxRegion struct {
	ID           string          `json:"id"`
	CountryCode  string          `json:"country_code"`
	ProvinceCode *string         `json:"province_code"`
	ProviderID   *string         `json:"provider_id"`
	ParentID     *string         `json:"parent_id"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedBy    *string         `json:"created_by"`
	TaxRates     []TaxRate       `json:"tax_rates,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type TaxProvider struct {
	ID        string `json:"id"`
	IsEnabled bool   `json:"is_enabled"`
}
type TaxRateRule struct {
	ID          string          `json:"id"`
	TaxRateID   string          `json:"tax_rate_id"`
	Reference   string          `json:"reference"`
	ReferenceID string          `json:"reference_id"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedBy   *string         `json:"created_by"`
}
type TaxRate struct {
	ID           string          `json:"id"`
	TaxRegionID  string          `json:"tax_region_id"`
	Rate         *float64        `json:"rate"`
	Code         string          `json:"code"`
	Name         string          `json:"name"`
	IsDefault    bool            `json:"is_default"`
	IsCombinable bool            `json:"is_combinable"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedBy    *string         `json:"created_by"`
	Rules        []TaxRateRule   `json:"rules,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}
type RateInput struct {
	Rate         *float64        `json:"rate"`
	Code         string          `json:"code"`
	Name         string          `json:"name"`
	IsDefault    *bool           `json:"is_default"`
	IsCombinable *bool           `json:"is_combinable"`
	Metadata     json.RawMessage `json:"metadata"`
	Rules        []RuleInput     `json:"rules"`
}
type RuleInput struct {
	Reference   string          `json:"reference"`
	ReferenceID string          `json:"reference_id"`
	Metadata    json.RawMessage `json:"metadata"`
}
type RuleCreateInput struct {
	TaxRateID   string          `json:"tax_rate_id"`
	Reference   string          `json:"reference"`
	ReferenceID string          `json:"reference_id"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedBy   *string         `json:"created_by"`
}
type RegionInput struct {
	CountryCode    string          `json:"country_code"`
	ProviderID     *string         `json:"provider_id"`
	ProvinceCode   *string         `json:"province_code"`
	ParentID       *string         `json:"parent_id"`
	DefaultTaxRate *RateInput      `json:"default_tax_rate"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedBy      *string         `json:"-"`
}
type RegionUpdate struct {
	ProvinceCode *string         `json:"province_code"`
	ProviderID   *string         `json:"provider_id"`
	Metadata     json.RawMessage `json:"metadata"`
}
type RateUpdate struct {
	Rate         *float64        `json:"rate"`
	Code         *string         `json:"code"`
	Name         *string         `json:"name"`
	IsDefault    *bool           `json:"is_default"`
	IsCombinable *bool           `json:"is_combinable"`
	Metadata     json.RawMessage `json:"metadata"`
	Rules        *[]RuleInput    `json:"rules,omitempty"`
}
