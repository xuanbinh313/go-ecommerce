package models

import (
	"encoding/json"
	"time"
)

type RuleInput struct {
	Description *string  `json:"description"`
	Attribute   string   `json:"attribute"`
	Operator    string   `json:"operator"`
	Values      []string `json:"values"`
}
type Rule struct {
	ID          string   `json:"id"`
	Description *string  `json:"description"`
	Attribute   string   `json:"attribute"`
	Operator    string   `json:"operator"`
	Values      []string `json:"values"`
}
type ApplicationMethodInput struct {
	Value               *float64    `json:"value"`
	CurrencyCode        *string     `json:"currency_code"`
	MaxQuantity         *int        `json:"max_quantity"`
	ApplyToQuantity     *int        `json:"apply_to_quantity"`
	BuyRulesMinQuantity *int        `json:"buy_rules_min_quantity"`
	Type                string      `json:"type"`
	TargetType          string      `json:"target_type"`
	Allocation          *string     `json:"allocation"`
	TargetRules         []RuleInput `json:"target_rules"`
	BuyRules            []RuleInput `json:"buy_rules"`
}
type ApplicationMethod struct {
	ID                  string   `json:"id"`
	PromotionID         string   `json:"promotion_id"`
	Value               *float64 `json:"value"`
	CurrencyCode        *string  `json:"currency_code"`
	MaxQuantity         *int     `json:"max_quantity"`
	ApplyToQuantity     *int     `json:"apply_to_quantity"`
	BuyRulesMinQuantity *int     `json:"buy_rules_min_quantity"`
	Type                string   `json:"type"`
	TargetType          string   `json:"target_type"`
	Allocation          *string  `json:"allocation"`
	TargetRules         []Rule   `json:"target_rules"`
	BuyRules            []Rule   `json:"buy_rules"`
}
type Promotion struct {
	ID                string             `json:"id"`
	Code              string             `json:"code"`
	IsAutomatic       bool               `json:"is_automatic"`
	IsTaxInclusive    bool               `json:"is_tax_inclusive"`
	Limit             *int               `json:"limit"`
	Used              int                `json:"used"`
	Type              string             `json:"type"`
	Status            string             `json:"status"`
	CampaignID        *string            `json:"campaign_id"`
	ApplicationMethod *ApplicationMethod `json:"application_method"`
	Rules             []Rule             `json:"rules"`
	Metadata          json.RawMessage    `json:"metadata"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}
type CreateInput struct {
	Code              string                 `json:"code"`
	IsAutomatic       *bool                  `json:"is_automatic"`
	IsTaxInclusive    *bool                  `json:"is_tax_inclusive"`
	Limit             *int                   `json:"limit"`
	Type              string                 `json:"type"`
	Status            string                 `json:"status"`
	CampaignID        *string                `json:"campaign_id"`
	ApplicationMethod ApplicationMethodInput `json:"application_method"`
	Rules             []RuleInput            `json:"rules"`
	Metadata          json.RawMessage        `json:"metadata"`
}
type UpdateInput struct {
	Code              *string                 `json:"code"`
	IsAutomatic       *bool                   `json:"is_automatic"`
	IsTaxInclusive    *bool                   `json:"is_tax_inclusive"`
	Limit             *int                    `json:"limit"`
	Type              *string                 `json:"type"`
	Status            *string                 `json:"status"`
	CampaignID        *string                 `json:"campaign_id"`
	ApplicationMethod *ApplicationMethodInput `json:"application_method"`
	Metadata          json.RawMessage         `json:"metadata"`
}
type Filter struct {
	IDs, Codes, CampaignIDs, Statuses []string
	Limit, Offset                     int
}

type CampaignBudget struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	CurrencyCode *string         `json:"currency_code"`
	Limit        json.RawMessage `json:"limit"`
	Used         json.RawMessage `json:"used"`
	Attribute    *string         `json:"attribute"`
}
type Campaign struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Description        *string         `json:"description"`
	CampaignIdentifier string          `json:"campaign_identifier"`
	StartsAt           *time.Time      `json:"starts_at"`
	EndsAt             *time.Time      `json:"ends_at"`
	Budget             *CampaignBudget `json:"budget"`
	Currency           map[string]any  `json:"currency"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	DeletedAt          *time.Time      `json:"deleted_at"`
}
type CampaignBudgetInput struct {
	Type         string          `json:"type"`
	CurrencyCode *string         `json:"currency_code"`
	Limit        json.RawMessage `json:"limit"`
	Attribute    *string         `json:"attribute"`
}
type CampaignBudgetUpdateInput struct {
	Limit json.RawMessage `json:"limit"`
}
type CampaignInput struct {
	Name               string               `json:"name"`
	Description        *string              `json:"description"`
	CampaignIdentifier string               `json:"campaign_identifier"`
	StartsAt           *time.Time           `json:"starts_at"`
	EndsAt             *time.Time           `json:"ends_at"`
	Budget             *CampaignBudgetInput `json:"budget"`
}
type CampaignUpdate struct {
	Name               *string                    `json:"name"`
	Description        *string                    `json:"description"`
	CampaignIdentifier *string                    `json:"campaign_identifier"`
	StartsAt           *time.Time                 `json:"starts_at"`
	EndsAt             *time.Time                 `json:"ends_at"`
	Budget             *CampaignBudgetUpdateInput `json:"budget"`
}
type CampaignFilter struct {
	Query, Identifier, BudgetCurrency string
	Limit, Offset                     int
}
