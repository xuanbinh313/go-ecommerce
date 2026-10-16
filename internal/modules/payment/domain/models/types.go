package models

import (
	"encoding/json"
	"time"
)

type PaymentCollectionStatus string

const (
	PaymentCollectionNotPaid             PaymentCollectionStatus = "not_paid"
	PaymentCollectionAwaiting            PaymentCollectionStatus = "awaiting"
	PaymentCollectionAuthorized          PaymentCollectionStatus = "authorized"
	PaymentCollectionPartiallyAuthorized PaymentCollectionStatus = "partially_authorized"
	PaymentCollectionCanceled            PaymentCollectionStatus = "canceled"
	PaymentCollectionFailed              PaymentCollectionStatus = "failed"
	PaymentCollectionPartiallyCaptured   PaymentCollectionStatus = "partially_captured"
	PaymentCollectionCompleted           PaymentCollectionStatus = "completed"
)

type PaymentSessionStatus string

const (
	PaymentSessionAuthorized           PaymentSessionStatus = "authorized"
	PaymentSessionCaptured             PaymentSessionStatus = "captured"
	PaymentSessionPending              PaymentSessionStatus = "pending"
	PaymentSessionRequiresMore         PaymentSessionStatus = "requires_more"
	PaymentSessionError                PaymentSessionStatus = "error"
	PaymentSessionCanceled             PaymentSessionStatus = "canceled"
	PaymentSessionPendingAuthorization PaymentSessionStatus = "pending_authorization"
)

type RefundStatus string

const (
	RefundPending   RefundStatus = "pending"
	RefundSucceeded RefundStatus = "succeeded"
	RefundFailed    RefundStatus = "failed"
)

// Amount stores PostgreSQL NUMERIC as a decimal string; this avoids float loss.
type Amount string

type PaymentCollection struct {
	ID                  string                  `json:"id"`
	CurrencyCode        string                  `json:"currency_code"`
	Amount              Amount                  `json:"amount"`
	RawAmount           json.RawMessage         `json:"raw_amount"`
	AuthorizedAmount    *Amount                 `json:"authorized_amount,omitempty"`
	RawAuthorizedAmount json.RawMessage         `json:"raw_authorized_amount,omitempty"`
	CapturedAmount      *Amount                 `json:"captured_amount,omitempty"`
	RawCapturedAmount   json.RawMessage         `json:"raw_captured_amount,omitempty"`
	RefundedAmount      *Amount                 `json:"refunded_amount,omitempty"`
	RawRefundedAmount   json.RawMessage         `json:"raw_refunded_amount,omitempty"`
	CompletedAt         *time.Time              `json:"completed_at,omitempty"`
	Status              PaymentCollectionStatus `json:"status"`
	Metadata            map[string]any          `json:"metadata,omitempty"`
	PaymentProviders    []PaymentProvider       `json:"payment_providers,omitempty"`
	PaymentSessions     []PaymentSession        `json:"payment_sessions,omitempty"`
	Payments            []Payment               `json:"payments,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
	DeletedAt           *time.Time              `json:"deleted_at,omitempty"`
}

type PaymentProvider struct {
	ID        string `json:"id"`
	IsEnabled bool   `json:"is_enabled"`
}

type PaymentSession struct {
	ID                  string               `json:"id"`
	CurrencyCode        string               `json:"currency_code"`
	Amount              Amount               `json:"amount"`
	RawAmount           json.RawMessage      `json:"raw_amount"`
	ProviderID          string               `json:"provider_id"`
	Data                map[string]any       `json:"data"`
	Context             map[string]any       `json:"context,omitempty"`
	Status              PaymentSessionStatus `json:"status"`
	AuthorizedAt        *time.Time           `json:"authorized_at,omitempty"`
	PaymentCollectionID string               `json:"payment_collection_id"`
	PaymentID           *string              `json:"payment_id,omitempty"`
	Metadata            map[string]any       `json:"metadata,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	DeletedAt           *time.Time           `json:"deleted_at,omitempty"`
}

type Payment struct {
	ID                  string          `json:"id"`
	Amount              Amount          `json:"amount"`
	RawAmount           json.RawMessage `json:"raw_amount"`
	CurrencyCode        string          `json:"currency_code"`
	ProviderID          string          `json:"provider_id"`
	Data                map[string]any  `json:"data,omitempty"`
	Metadata            map[string]any  `json:"metadata,omitempty"`
	CapturedAt          *time.Time      `json:"captured_at,omitempty"`
	CanceledAt          *time.Time      `json:"canceled_at,omitempty"`
	PaymentCollectionID string          `json:"payment_collection_id"`
	PaymentSessionID    string          `json:"payment_session_id"`
	Refunds             []Refund        `json:"refunds,omitempty"`
	Captures            []Capture       `json:"captures,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	DeletedAt           *time.Time      `json:"deleted_at,omitempty"`
}

type Capture struct {
	ID        string          `json:"id"`
	Amount    Amount          `json:"amount"`
	RawAmount json.RawMessage `json:"raw_amount"`
	PaymentID string          `json:"payment_id"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
	CreatedBy *string         `json:"created_by,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}

type Refund struct {
	ID             string          `json:"id"`
	Amount         Amount          `json:"amount"`
	RawAmount      json.RawMessage `json:"raw_amount"`
	Status         RefundStatus    `json:"status"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	PaymentID      string          `json:"payment_id"`
	RefundReasonID *string         `json:"refund_reason_id,omitempty"`
	RefundReason   *RefundReason   `json:"refund_reason"`
	Note           *string         `json:"note,omitempty"`
	CreatedBy      *string         `json:"created_by,omitempty"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty"`
}

type RefundReason struct {
	ID          string         `json:"id"`
	Label       string         `json:"label"`
	Code        string         `json:"code"`
	Description *string        `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   *time.Time     `json:"deleted_at,omitempty"`
}

type RefundReasonInput struct {
	Label       string  `json:"label"`
	Code        string  `json:"code"`
	Description *string `json:"description"`
}
type RefundReasonFilter struct {
	IDs                             []string
	Query                           string
	CreatedAt, UpdatedAt, DeletedAt map[string]string
	Limit, Offset                   int
}

type AccountHolder struct {
	ID         string         `json:"id"`
	ProviderID string         `json:"provider_id"`
	ExternalID string         `json:"external_id"`
	Email      *string        `json:"email,omitempty"`
	Data       map[string]any `json:"data"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  *time.Time     `json:"deleted_at,omitempty"`
}

type CreatePaymentCollectionInput struct {
	CurrencyCode       string          `json:"currency_code"`
	Amount             Amount          `json:"amount"`
	RawAmount          json.RawMessage `json:"raw_amount"`
	Metadata           map[string]any  `json:"metadata,omitempty"`
	PaymentProviderIDs []string        `json:"payment_provider_ids,omitempty"`
}

type PaymentFilter struct {
	IDs               []string
	PaymentSessionIDs []string
	Query             string
	CreatedAt         map[string]string
	UpdatedAt         map[string]string
	DeletedAt         map[string]string
	Limit             int
	Offset            int
}

type PaymentProviderFilter struct {
	IDs       []string
	IsEnabled *bool
	Limit     int
	Offset    int
}
