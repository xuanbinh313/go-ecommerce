package models

import (
	"encoding/json"
	"time"
)

type SalesChannel struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	IsDisabled  bool            `json:"is_disabled"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
}

type Input struct {
	ID             *string         `json:"id,omitempty"`
	Name           *string         `json:"name,omitempty"`
	Description    *string         `json:"description,omitempty"`
	IsDisabled     *bool           `json:"is_disabled,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	DescriptionSet bool            `json:"-"`
	MetadataSet    bool            `json:"-"`
}
