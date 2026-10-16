package security

import (
	"fmt"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain"
	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct{ secret []byte }

type claims struct {
	ActorID                 string         `json:"actor_id"`
	ActorType               string         `json:"actor_type"`
	AuthIdentityID          string         `json:"auth_identity_id"`
	AuthProvider            string         `json:"auth_provider,omitempty"`
	Provider                string         `json:"provider,omitempty"`
	EntityID                string         `json:"entity_id,omitempty"`
	Purpose                 string         `json:"purpose,omitempty"`
	MFAEnabled              bool           `json:"mfa_enabled"`
	MFAChallengeCompletedAt *time.Time     `json:"mfa_challenge_completed_at"`
	AppMetadata             map[string]any `json:"app_metadata"`
	UserMetadata            map[string]any `json:"user_metadata"`
	jwt.RegisteredClaims
}

type resetClaims struct {
	EntityID  string `json:"entity_id"`
	Provider  string `json:"provider"`
	ActorType string `json:"actor_type"`
	Purpose   string `json:"purpose"`
	jwt.RegisteredClaims
}

func (m TokenManager) IssuePasswordResetToken(entityID, provider, actorType, jti string, expiresAt time.Time) (string, error) {
	now := time.Now()
	c := resetClaims{EntityID: entityID, Provider: provider, ActorType: actorType, Purpose: "reset", RegisteredClaims: jwt.RegisteredClaims{ID: jti, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expiresAt)}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

func NewTokenManager(secret string) TokenManager { return TokenManager{secret: []byte(secret)} }

func (m TokenManager) Issue(identityID, actorType, provider, actorID string, appMetadata, userMetadata map[string]any) (string, time.Time, error) {
	return m.IssueRefreshed(identityID, actorType, provider, actorID, appMetadata, userMetadata, false, nil)
}

func (m TokenManager) IssueRefreshed(identityID, actorType, provider, actorID string, appMetadata, userMetadata map[string]any, mfaEnabled bool, mfaChallengeCompletedAt *time.Time) (string, time.Time, error) {
	now, expires := time.Now(), time.Now().Add(24*time.Hour)
	tokenClaims := claims{AuthIdentityID: identityID, ActorType: actorType, AuthProvider: provider, ActorID: actorID, AppMetadata: appMetadata, UserMetadata: userMetadata, MFAEnabled: mfaEnabled, MFAChallengeCompletedAt: mfaChallengeCompletedAt, RegisteredClaims: jwt.RegisteredClaims{Issuer: "medusa-go-clone", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires)}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims).SignedString(m.secret)
	return token, expires, err
}

func (m TokenManager) Parse(token string) (*domain.TokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	tokenClaims, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	provider := tokenClaims.AuthProvider
	if provider == "" {
		provider = tokenClaims.Provider
	}
	return &domain.TokenClaims{ActorID: tokenClaims.ActorID, ActorType: tokenClaims.ActorType, AuthIdentityID: tokenClaims.AuthIdentityID, EntityID: tokenClaims.EntityID, Provider: provider, Purpose: tokenClaims.Purpose, TokenID: tokenClaims.ID, MFAEnabled: tokenClaims.MFAEnabled, MFAChallengeCompletedAt: tokenClaims.MFAChallengeCompletedAt, UserMetadata: tokenClaims.UserMetadata}, nil
}

var _ domain.TokenManager = TokenManager{}
var _ domain.RefreshTokenManager = TokenManager{}
var _ domain.PasswordHasher = PasswordHasher{}
