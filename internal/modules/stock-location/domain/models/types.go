package models

import (
	"encoding/json"
	"time"
)

type Address struct {
	ID          string          `json:"id"`
	Address1    string          `json:"address_1"`
	Address2    *string         `json:"address_2"`
	Company     *string         `json:"company"`
	City        *string         `json:"city"`
	CountryCode string          `json:"country_code"`
	Phone       *string         `json:"phone"`
	Province    *string         `json:"province"`
	PostalCode  *string         `json:"postal_code"`
	Metadata    json.RawMessage `json:"metadata"`
}
type StockLocation struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	AddressID     *string          `json:"address_id,omitempty"`
	Address       *Address         `json:"address,omitempty"`
	Metadata      json.RawMessage  `json:"metadata"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	DeletedAt     *time.Time       `json:"deleted_at,omitempty"`
	SalesChannels []map[string]any `json:"sales_channels,omitempty"`
}
type Input struct {
	Name      string          `json:"name"`
	Address   *AddressInput   `json:"address,omitempty"`
	AddressID *string         `json:"address_id,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
}
type UpdateInput struct {
	Name      *string         `json:"name,omitempty"`
	Address   *AddressInput   `json:"address,omitempty"`
	AddressID *string         `json:"address_id,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
}
type AddressInput struct {
	Address1    string          `json:"address_1"`
	Address2    *string         `json:"address_2"`
	Company     *string         `json:"company"`
	City        *string         `json:"city"`
	CountryCode string          `json:"country_code"`
	Phone       *string         `json:"phone"`
	Province    *string         `json:"province"`
	PostalCode  *string         `json:"postal_code"`
	Metadata    json.RawMessage `json:"metadata"`
}
type Filter struct {
	IDs, Names, AddressIDs, SalesChannelIDs []string
	Query                                   string
	Limit, Offset                           int
}
