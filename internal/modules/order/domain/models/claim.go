package models

import (
	"encoding/json"
	"time"
)

type Claim struct {
	ID              string          `json:"id"`
	OrderVersion    int             `json:"order_version"`
	DisplayID       int64           `json:"display_id"`
	Type            string          `json:"type"`
	NoNotification  *bool           `json:"no_notification"`
	RefundAmount    json.RawMessage `json:"refund_amount"`
	CreatedBy       *string         `json:"created_by"`
	CanceledAt      *time.Time      `json:"canceled_at"`
	Metadata        json.RawMessage `json:"metadata"`
	OrderID         string          `json:"order_id"`
	ReturnID        *string         `json:"return_id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
	AdditionalItems []ClaimItem     `json:"additional_items"`
	ClaimItems      []ClaimItem     `json:"claim_items"`
}

type ClaimItem struct {
	ID               string           `json:"id"`
	ClaimID          string           `json:"claim_id"`
	ItemID           string           `json:"item_id"`
	IsAdditionalItem bool             `json:"is_additional_item"`
	Reason           *string          `json:"reason"`
	Quantity         json.RawMessage  `json:"quantity"`
	Note             *string          `json:"note"`
	Metadata         json.RawMessage  `json:"metadata"`
	Images           []ClaimItemImage `json:"images,omitempty"`
}

type ClaimItemImage struct {
	ID          string          `json:"id"`
	ClaimItemID string          `json:"claim_item_id"`
	URL         string          `json:"url"`
	Metadata    json.RawMessage `json:"metadata"`
}

type ClaimFilter struct {
	IDs, OrderIDs                   []string
	CreatedAt, UpdatedAt, DeletedAt map[string]string
	Limit, Offset                   int
	IncludeItems                    bool
}

type CreateClaimInput struct {
	Type         string          `json:"type"`
	OrderID      string          `json:"order_id"`
	Description  *string         `json:"description"`
	InternalNote *string         `json:"internal_note"`
	CreatedBy    *string         `json:"created_by"`
	Metadata     json.RawMessage `json:"metadata"`
}
