package models

import (
	"encoding/json"
	"time"
)

type ShippingOptionType struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Description *string `json:"description"`
	Code        string  `json:"code"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}
type ShippingOptionTypeInput struct {
	Label       string  `json:"label"`
	Description *string `json:"description"`
	Code        string  `json:"code"`
}
type ShippingOptionTypeUpdate struct {
	Label       *string `json:"label"`
	Description *string `json:"description"`
	Code        *string `json:"code"`
}
type ShippingOptionTypeFilter struct {
	IDs          []string
	Labels       []string
	Codes        []string
	Descriptions []string
	Query        string
}

type ShippingOptionRule struct {
	ID               string          `json:"id"`
	Attribute        string          `json:"attribute"`
	Operator         string          `json:"operator"`
	Value            json.RawMessage `json:"value"`
	ShippingOptionID string          `json:"-"`
}
type ShippingOptionRuleUpdate struct {
	ID        string           `json:"id"`
	Attribute *string          `json:"attribute"`
	Operator  *string          `json:"operator"`
	Value     *json.RawMessage `json:"value"`
}
type ShippingOptionRuleBatchInput struct {
	Create []ShippingOptionRuleInput  `json:"create"`
	Update []ShippingOptionRuleUpdate `json:"update"`
	Delete []string                   `json:"delete"`
}
type ShippingOptionRuleBatchResult struct {
	Created         []ShippingOptionRule `json:"created"`
	Updated         []ShippingOptionRule `json:"updated"`
	Deleted         []string             `json:"deleted"`
	PreviousUpdated []ShippingOptionRule `json:"-"`
	DeletedRules    []ShippingOptionRule `json:"-"`
}
type ShippingOption struct {
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	PriceType            string               `json:"price_type"`
	Data                 json.RawMessage      `json:"data"`
	ProviderID           *string              `json:"provider_id"`
	Metadata             json.RawMessage      `json:"metadata"`
	ServiceZoneID        string               `json:"service_zone_id"`
	ShippingProfileID    *string              `json:"shipping_profile_id"`
	ShippingOptionTypeID string               `json:"shipping_option_type_id"`
	Rules                []ShippingOptionRule `json:"rules"`
	Type                 *ShippingOptionType  `json:"type"`
	Prices               []map[string]any     `json:"prices"`
	ServiceZone          *ServiceZone         `json:"service_zone"`
	ShippingProfile      *ShippingProfile     `json:"shipping_profile"`
	Provider             map[string]any       `json:"provider"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}
type ShippingOptionFilter struct {
	IDs, ServiceZoneIDs, ShippingProfileIDs, ProviderIDs, TypeIDs, StockLocationIDs []string
	Query                                                                           string
	CreatedAt, UpdatedAt, DeletedAt                                                 map[string]string
}
type ShippingOptionRuleInput struct {
	Attribute string          `json:"attribute"`
	Operator  string          `json:"operator"`
	Value     json.RawMessage `json:"value"`
}
type ShippingOptionPriceRuleInput struct {
	Attribute string          `json:"attribute"`
	Operator  string          `json:"operator"`
	Value     json.RawMessage `json:"value"`
}
type ShippingOptionPriceInput struct {
	CurrencyCode *string                        `json:"currency_code"`
	RegionID     *string                        `json:"region_id"`
	Amount       json.RawMessage                `json:"amount"`
	Rules        []ShippingOptionPriceRuleInput `json:"rules"`
}
type ShippingOptionInput struct {
	Name              string                     `json:"name"`
	ServiceZoneID     string                     `json:"service_zone_id"`
	ShippingProfileID string                     `json:"shipping_profile_id"`
	Data              map[string]any             `json:"data"`
	PriceType         string                     `json:"price_type"`
	ProviderID        string                     `json:"provider_id"`
	Type              *ShippingOptionTypeInput   `json:"type"`
	TypeID            string                     `json:"type_id"`
	Prices            []ShippingOptionPriceInput `json:"prices"`
	Rules             []ShippingOptionRuleInput  `json:"rules"`
	Metadata          map[string]any             `json:"metadata"`
}

type ShippingOptionUpdatePriceInput struct {
	ID           *string                         `json:"id"`
	CurrencyCode *string                         `json:"currency_code"`
	RegionID     *string                         `json:"region_id"`
	Amount       *json.RawMessage                `json:"amount"`
	Rules        *[]ShippingOptionPriceRuleInput `json:"rules"`
}

type ShippingOptionUpdateInput struct {
	Name              *string                           `json:"name"`
	Data              *map[string]any                   `json:"data"`
	PriceType         *string                           `json:"price_type"`
	ProviderID        *string                           `json:"provider_id"`
	ShippingProfileID *string                           `json:"shipping_profile_id"`
	Type              *ShippingOptionTypeInput          `json:"type"`
	TypeID            *string                           `json:"type_id"`
	Prices            *[]ShippingOptionUpdatePriceInput `json:"prices"`
	Rules             *[]ShippingOptionRuleUpdate       `json:"rules"`
	Metadata          *map[string]any                   `json:"metadata"`
}

type GeoZone struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"`
	CountryCode      string          `json:"country_code"`
	ProvinceCode     *string         `json:"province_code"`
	City             *string         `json:"city"`
	PostalExpression json.RawMessage `json:"postal_expression"`
	ServiceZoneID    string          `json:"service_zone_id"`
	Metadata         json.RawMessage `json:"metadata"`
}
type GeoZoneInput struct {
	ID               *string         `json:"id"`
	Type             string          `json:"type"`
	CountryCode      string          `json:"country_code"`
	ProvinceCode     *string         `json:"province_code"`
	City             *string         `json:"city"`
	PostalExpression json.RawMessage `json:"postal_expression"`
	Metadata         json.RawMessage `json:"metadata"`
}
type ServiceZone struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	FulfillmentSetID string          `json:"fulfillment_set_id"`
	GeoZones         []GeoZone       `json:"geo_zones"`
	Metadata         json.RawMessage `json:"metadata"`
}
type ServiceZoneInput struct {
	Name     string          `json:"name"`
	GeoZones []GeoZoneInput  `json:"geo_zones"`
	Metadata json.RawMessage `json:"metadata"`
}
type ServiceZoneUpdate struct {
	Name     *string         `json:"name"`
	GeoZones *[]GeoZoneInput `json:"geo_zones"`
	Metadata json.RawMessage `json:"metadata"`
}
type FulfillmentSet struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	ServiceZones []ServiceZone   `json:"service_zones"`
	Metadata     json.RawMessage `json:"metadata"`
}
type CreateSetInput struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Metadata json.RawMessage `json:"metadata"`
}
