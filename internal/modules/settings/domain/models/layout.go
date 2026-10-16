package models

import (
	"encoding/json"
	"time"
)

type LayoutConfiguration struct {
	ID              string          `json:"id"`
	Zone            string          `json:"zone"`
	UserID          *string         `json:"user_id"`
	IsSystemDefault bool            `json:"is_system_default"`
	Configuration   json.RawMessage `json:"configuration"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type PropertyLabel struct {
	ID          string    `json:"id"`
	Entity      string    `json:"entity"`
	Property    string    `json:"property"`
	Label       string    `json:"label"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type PropertyLabelInput struct {
	Entity      string  `json:"entity"`
	Property    string  `json:"property"`
	Label       string  `json:"label"`
	Description *string `json:"description"`
}
type PropertyLabelFilter struct {
	Entity   string
	Property string
	Query    string
}

type LayoutConfigurations struct {
	PersonalConfiguration *LayoutConfiguration `json:"personal_configuration"`
	DefaultConfiguration  *LayoutConfiguration `json:"default_configuration"`
	ActiveScope           string               `json:"active_scope"`
}

type LayoutConfigurationFilter struct {
	IDs             []string
	Zones           []string
	UserIDs         []string
	IsSystemDefault *bool
}
