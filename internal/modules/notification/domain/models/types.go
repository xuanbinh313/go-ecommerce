package models

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID                     string          `json:"id"`
	To                     string          `json:"to"`
	From                   *string         `json:"from"`
	Channel                string          `json:"channel"`
	Template               *string         `json:"template"`
	Data                   json.RawMessage `json:"data"`
	ProviderData           json.RawMessage `json:"provider_data"`
	TriggerType            *string         `json:"trigger_type"`
	ResourceID             *string         `json:"resource_id"`
	ResourceType           *string         `json:"resource_type"`
	ReceiverID             *string         `json:"receiver_id"`
	OriginalNotificationID *string         `json:"original_notification_id"`
	IdempotencyKey         *string         `json:"idempotency_key"`
	ExternalID             *string         `json:"external_id"`
	Status                 string          `json:"status"`
	ProviderID             *string         `json:"provider_id"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

type NotificationFilter struct {
	IDs     []string
	To      []string
	Channel []string
	Query   string
}
