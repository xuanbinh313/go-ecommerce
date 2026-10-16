package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain/models"
)

var ErrIdentityNotFound = errors.New("auth identity not found")
var ErrInvalidPasswordResetToken = errors.New("Invalid token")
var ErrInvalidMFARecoveryCode = errors.New("Recovery code is invalid or already used")

type TokenClaims struct {
	ActorID                 string
	ActorType               string
	AuthIdentityID          string
	EntityID                string
	Provider                string
	Purpose                 string
	TokenID                 string
	MFAEnabled              bool
	MFAChallengeCompletedAt *time.Time
	UserMetadata            map[string]any
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Verify(encoded, password string) (valid bool, legacy bool, err error)
}

type TokenManager interface {
	Issue(identityID, actorType, provider, actorID string, appMetadata, userMetadata map[string]any) (token string, expiresAt time.Time, err error)
	Parse(string) (*TokenClaims, error)
}

type RefreshTokenManager interface {
	IssueRefreshed(identityID, actorType, provider, actorID string, appMetadata, userMetadata map[string]any, mfaEnabled bool, mfaChallengeCompletedAt *time.Time) (token string, expiresAt time.Time, err error)
}

type ProviderCredential struct {
	IdentityID         string
	PasswordHash       string
	ActorID            string
	ProviderIdentityID string
	AppMetadata        json.RawMessage
	UserMetadata       json.RawMessage
}

type OAuthIdentity struct {
	IdentityID   string
	ActorID      string
	AppMetadata  json.RawMessage
	UserMetadata json.RawMessage
}

// IdentityRepository defines the auth module's persistence boundary.
type IdentityRepository interface {
	CreateIdentityWithProvider(context.Context, string, string, string) (string, error)
	DeleteIdentity(context.Context, string) error
	FindProviderCredential(context.Context, string, string, string) (ProviderCredential, error)
	UpdateProviderPassword(context.Context, string, string) error
	SetActorMetadata(context.Context, string, string, string) (string, error)
	UnsetActorMetadata(context.Context, string, string) error
	FindIdentityByActor(context.Context, string, string) (AuthIdentityRecord, error)
	ListProviderNamesByActor(context.Context, string, string) ([]string, error)
	ListProviderNamesByIdentity(context.Context, string) ([]string, error)
	RetrieveAuthIdentity(context.Context, string, string) (AuthIdentitySnapshot, error)
	FindOAuthIdentity(context.Context, string, string) (OAuthIdentity, error)
	CreateOAuthIdentity(context.Context, string, string, json.RawMessage, json.RawMessage) (OAuthIdentity, error)
	UpdateOAuthIdentity(context.Context, string, string, json.RawMessage, json.RawMessage) (OAuthIdentity, error)
	UpdateOAuthUserMetadata(context.Context, string, string, json.RawMessage) (OAuthIdentity, error)
	CreatePasswordResetToken(context.Context, string, string, time.Duration) (string, time.Time, error)
	ConsumePasswordResetToken(context.Context, string, string, string) error
	RequestAuthVerification(context.Context, models.RequestAuthVerificationInput, time.Duration) (models.AuthVerification, error)
	ConfirmAuthVerification(context.Context, models.ConfirmAuthVerificationInput, time.Duration) (models.AuthVerification, error)
	ListAuthMFA(context.Context, string, string) ([]models.AuthMFAFactor, error)
	RetrieveAuthMFA(context.Context, string, string) (models.AuthMFAFactor, error)
	CreateAuthMFA(context.Context, models.AuthMFAFactor) (models.AuthMFAFactor, error)
	UpdateAuthMFA(context.Context, string, string, string) (models.AuthMFAFactor, error)
	ListAuthMFARecoveryCodes(context.Context, string) ([]models.AuthMFARecoveryCode, error)
	ReplaceAuthMFARecoveryCodes(context.Context, string, []string) error
	DeleteAuthMFARecoveryCode(context.Context, string) (bool, error)
}

type TOTPProvider interface {
	Start(models.AuthMFAStartInput) (encryptedSecret, secret, issuer, otpauthURL string, err error)
	Verify(encryptedSecret, code string) (bool, error)
}

type MFAChallengeCache interface {
	Get(context.Context, string) (any, bool, error)
	Set(context.Context, string, any, int) error
	Invalidate(context.Context, string) error
}

type OAuthStateCache interface {
	Get(context.Context, string) (any, bool, error)
	Set(context.Context, string, any, int) error
	Invalidate(context.Context, string) error
}

type OAuthProvider interface {
	Begin(context.Context, string, string) (location string, stateData map[string]string, ttlSeconds int, err error)
	Exchange(context.Context, string, string, map[string]string, map[string]string) (entityID string, providerMetadata, userMetadata json.RawMessage, err error)
	UpdateExistingIdentity() bool
	UpdateUserMetadataOnly() bool
	DisplayName() string
}

type AuthIdentitySnapshot struct {
	ID           string
	AppMetadata  json.RawMessage
	UserMetadata json.RawMessage
}

type AuthIdentityRecord struct {
	ID          string
	AppMetadata json.RawMessage
}

type PasswordResetTokenSigner interface {
	IssuePasswordResetToken(entityID, provider, actorType, jti string, expiresAt time.Time) (string, error)
}

type IdentityMetadata interface {
	SetActorMetadata(context.Context, string, string, string) (string, error)
}
