package models

import (
	"encoding/json"
	"time"
)

type ShippingProfile struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}
type ShippingProfileInput struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Metadata json.RawMessage `json:"metadata"`
}
type ShippingProfileFilter struct {
	IDs, Names []string
	Type       string
	Query      string
}
