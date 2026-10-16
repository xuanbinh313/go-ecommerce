package models

import (
	"encoding/json"
	"time"
)

type Customer struct {
	ID          string          `json:"id"`
	CompanyName *string         `json:"company_name"`
	FirstName   *string         `json:"first_name"`
	LastName    *string         `json:"last_name"`
	Email       *string         `json:"email"`
	Phone       *string         `json:"phone"`
	HasAccount  bool            `json:"has_account"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedBy   *string         `json:"created_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Addresses   []Address       `json:"addresses"`
	Groups      []CustomerGroup `json:"groups"`
}
type Address struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	AddressInput
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type CustomerGroup struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedBy   *string         `json:"created_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CustomerIDs []string        `json:"customer_ids,omitempty"`
}
type CustomerGroupUpdate struct {
	Name     *string         `json:"name"`
	Metadata json.RawMessage `json:"metadata"`
}
type CustomerGroupCustomer struct {
	ID              string          `json:"id"`
	CustomerID      string          `json:"customer_id"`
	CustomerGroupID string          `json:"customer_group_id"`
	CreatedBy       *string         `json:"created_by,omitempty"`
	Metadata        json.RawMessage `json:"metadata"`
}
type AddressInput struct {
	AddressName     *string         `json:"address_name"`
	DefaultShipping bool            `json:"is_default_shipping"`
	DefaultBilling  bool            `json:"is_default_billing"`
	Company         *string         `json:"company"`
	FirstName       *string         `json:"first_name"`
	LastName        *string         `json:"last_name"`
	Address1        *string         `json:"address_1"`
	Address2        *string         `json:"address_2"`
	City            *string         `json:"city"`
	CountryCode     *string         `json:"country_code"`
	Province        *string         `json:"province"`
	PostalCode      *string         `json:"postal_code"`
	Phone           *string         `json:"phone"`
	Metadata        json.RawMessage `json:"metadata"`
}
type Input struct {
	CompanyName *string         `json:"company_name"`
	FirstName   *string         `json:"first_name"`
	LastName    *string         `json:"last_name"`
	Email       *string         `json:"email"`
	Phone       *string         `json:"phone"`
	HasAccount  bool            `json:"has_account"`
	Metadata    json.RawMessage `json:"metadata"`
	Addresses   []AddressInput  `json:"addresses"`
	CreatedBy   *string         `json:"-"`
}
type AddressUpdate struct {
	AddressName     *string         `json:"address_name"`
	DefaultShipping *bool           `json:"is_default_shipping"`
	DefaultBilling  *bool           `json:"is_default_billing"`
	Company         *string         `json:"company"`
	FirstName       *string         `json:"first_name"`
	LastName        *string         `json:"last_name"`
	Address1        *string         `json:"address_1"`
	Address2        *string         `json:"address_2"`
	City            *string         `json:"city"`
	CountryCode     *string         `json:"country_code"`
	Province        *string         `json:"province"`
	PostalCode      *string         `json:"postal_code"`
	Phone           *string         `json:"phone"`
	Metadata        json.RawMessage `json:"metadata"`
}
