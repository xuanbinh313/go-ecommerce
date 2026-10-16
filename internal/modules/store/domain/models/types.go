package models

import (
	"encoding/json"
	"time"
)

type Store struct {
	ID                    string          `json:"id"`
	Name                  string          `json:"name"`
	DefaultSalesChannelID *string         `json:"default_sales_channel_id"`
	DefaultRegionID       *string         `json:"default_region_id"`
	DefaultLocationID     *string         `json:"default_location_id"`
	Metadata              json.RawMessage `json:"metadata"`
	SupportedCurrencies   []StoreCurrency `json:"supported_currencies"`
	SupportedLocales      []StoreLocale   `json:"supported_locales"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	DeletedAt             *time.Time      `json:"deleted_at,omitempty"`
}

type StoreCurrency struct {
	ID           string  `json:"id"`
	StoreID      *string `json:"store_id,omitempty"`
	CurrencyCode string  `json:"currency_code"`
	IsDefault    bool    `json:"is_default"`
}

type StoreLocale struct {
	ID         string  `json:"id"`
	StoreID    *string `json:"store_id,omitempty"`
	LocaleCode string  `json:"locale_code"`
}

type StoreInput struct {
	ID                    *string              `json:"id,omitempty"`
	Name                  *string              `json:"name,omitempty"`
	DefaultSalesChannelID *string              `json:"default_sales_channel_id,omitempty"`
	DefaultRegionID       *string              `json:"default_region_id,omitempty"`
	DefaultLocationID     *string              `json:"default_location_id,omitempty"`
	Metadata              json.RawMessage      `json:"metadata,omitempty"`
	SupportedCurrencies   []StoreCurrencyInput `json:"supported_currencies,omitempty"`
	SupportedLocales      []StoreLocaleInput   `json:"supported_locales,omitempty"`
}

type StoreCurrencyInput struct {
	CurrencyCode   string `json:"currency_code"`
	IsDefault      bool   `json:"is_default"`
	IsTaxInclusive *bool  `json:"is_tax_inclusive,omitempty"`
}
type StoreLocaleInput struct {
	LocaleCode string `json:"locale_code"`
}
