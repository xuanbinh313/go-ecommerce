package models

import (
	"encoding/json"
	"time"
)

type AuthIdentity struct {
	ID                  string                   `json:"id"`
	ProviderIdentities  []ProviderIdentity       `json:"provider_identities,omitempty"`
	MFAMethods          []AuthMFAFactor          `json:"mfa_factors,omitempty"`
	MFARecoveryCodes    []AuthMFARecoveryCode    `json:"mfa_recovery_codes,omitempty"`
	Verifications       []AuthVerification       `json:"verifications,omitempty"`
	PasswordResetTokens []AuthPasswordResetToken `json:"password_reset_tokens,omitempty"`
	AppMetadata         json.RawMessage          `json:"app_metadata"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           time.Time                `json:"updated_at"`
	DeletedAt           *time.Time               `json:"deleted_at,omitempty"`
}
type ProviderIdentity struct {
	ID               string          `json:"id"`
	AuthIdentityID   string          `json:"auth_identity_id"`
	EntityID         string          `json:"entity_id"`
	Provider         string          `json:"provider"`
	UserMetadata     json.RawMessage `json:"user_metadata"`
	ProviderMetadata json.RawMessage `json:"provider_metadata"`
}
type AuthVerification struct {
	ID               string          `json:"id"`
	AuthIdentityID   string          `json:"auth_identity_id"`
	EntityID         string          `json:"entity_id"`
	EntityType       string          `json:"entity_type"`
	CodeProvider     string          `json:"code_provider"`
	VerifiedAt       *time.Time      `json:"verified_at,omitempty"`
	RequestedAt      time.Time       `json:"requested_at"`
	ProviderMetadata json.RawMessage `json:"provider_metadata"`
	Metadata         json.RawMessage `json:"metadata"`
	Code             string          `json:"-"`
	ExpiresAt        *time.Time      `json:"-"`
}

type RequestAuthVerificationInput struct {
	AuthIdentityID string
	EntityID       string
	EntityType     string
	CodeProvider   string
	Metadata       json.RawMessage
}

type ConfirmAuthVerificationInput struct {
	AuthIdentityID string
	Code           string
	CodeProvider   string
}
type AuthPasswordResetToken struct {
	ID                 string    `json:"id"`
	AuthIdentityID     string    `json:"auth_identity_id"`
	ProviderIdentityID *string   `json:"provider_identity_id,omitempty"`
	EntityID           string    `json:"entity_id"`
	TokenHash          string    `json:"token_hash"`
	ExpiresAt          time.Time `json:"expires_at"`
}
type AuthMFAFactor struct {
	ID               string          `json:"id"`
	AuthIdentityID   string          `json:"auth_identity_id"`
	Provider         string          `json:"provider"`
	Status           string          `json:"status"`
	ProviderMetadata json.RawMessage `json:"provider_metadata,omitempty"`
	Metadata         json.RawMessage `json:"metadata"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        *time.Time      `json:"deleted_at,omitempty"`
}

type AuthMFAStartInput struct {
	AuthIdentityID string
	Provider       string
	Label          *string
	Issuer         string
	Metadata       json.RawMessage
}

type AuthMFAEnrollment struct {
	MFA        AuthMFAFactor `json:"mfa"`
	Secret     string        `json:"secret"`
	OTPAuthURL string        `json:"otpauth_url"`
}

type AuthMFAChallenge struct {
	ID             string          `json:"id"`
	AuthIdentityID string          `json:"auth_identity_id"`
	AuthProvider   string          `json:"auth_provider"`
	Methods        []string        `json:"methods"`
	ExpiresAt      time.Time       `json:"expires_at"`
	Attempts       int             `json:"attempts"`
	MaxAttempts    int             `json:"max_attempts"`
	CompletedAt    *time.Time      `json:"completed_at"`
	Metadata       json.RawMessage `json:"metadata"`
}
type AuthMFARecoveryCode struct {
	ID             string     `json:"id"`
	AuthIdentityID string     `json:"auth_identity_id"`
	CodeHash       string     `json:"code_hash"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
