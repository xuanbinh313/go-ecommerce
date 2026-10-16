package models

import (
	"encoding/json"
	"time"
)

type ReturnReason struct {
	ID                   string          `json:"id"`
	Value                string          `json:"value"`
	Label                string          `json:"label"`
	Description          *string         `json:"description"`
	Metadata             json.RawMessage `json:"metadata"`
	ParentReturnReasonID *string         `json:"parent_return_reason_id"`
	ParentReturnReason   *ReturnReason   `json:"parent_return_reason,omitempty"`
	ReturnReasonChildren []ReturnReason  `json:"return_reason_children,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	DeletedAt            *time.Time      `json:"deleted_at,omitempty"`
}
type ReturnReasonInput struct {
	Value                string          `json:"value"`
	Label                string          `json:"label"`
	Description          *string         `json:"description"`
	Metadata             json.RawMessage `json:"metadata"`
	ParentReturnReasonID *string         `json:"parent_return_reason_id"`
}
type ReturnReasonFilter struct {
	IDs, Values, Labels, Descriptions, ParentIDs []string
	Query                                        string
}
