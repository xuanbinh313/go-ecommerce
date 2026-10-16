package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain"
	"github.com/example/medusa-go-clone/internal/modules/auth/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
)

type Service struct {
	repository        domain.IdentityRepository
	passwords         domain.PasswordHasher
	tokens            domain.TokenManager
	totp              domain.TOTPProvider
	mfaDisablePolicy  string
	challengeCache    domain.MFAChallengeCache
	challengeTTL      int
	challengeAttempts int
	oauthStateCache   domain.OAuthStateCache
	oauthProviders    map[string]oauthProviderConfig
}

type oauthProviderConfig struct {
	ClientID    string
	CallbackURL string
	Provider    domain.OAuthProvider
}

type LoginResult struct {
	Token          string                   `json:"token"`
	AuthIdentityID string                   `json:"auth_identity_id"`
	ActorID        string                   `json:"actor_id,omitempty"`
	ExpiresAt      time.Time                `json:"expires_at"`
	MFARequired    bool                     `json:"mfa_required,omitempty"`
	MFAChallenge   *models.AuthMFAChallenge `json:"mfa_challenge,omitempty"`
}

func NewService(repository domain.IdentityRepository, passwords domain.PasswordHasher, tokens domain.TokenManager) *Service {
	return &Service{repository: repository, passwords: passwords, tokens: tokens, mfaDisablePolicy: "session", challengeTTL: 300, challengeAttempts: 5}
}

func (s *Service) ConfigureMFAChallenges(cache domain.MFAChallengeCache, ttlSeconds, maxAttempts int) {
	s.challengeCache = cache
	if ttlSeconds > 0 {
		s.challengeTTL = ttlSeconds
	}
	if maxAttempts > 0 {
		s.challengeAttempts = maxAttempts
	}
}

func (s *Service) ConfigureMFA(provider domain.TOTPProvider, disablePolicy string) {
	s.totp = provider
	if disablePolicy == "challenge" || disablePolicy == "session" {
		s.mfaDisablePolicy = disablePolicy
	}
}

func (s *Service) ConfigureOAuthProvider(cache domain.OAuthStateCache, name string, provider domain.OAuthProvider, clientID, callbackURL string) {
	s.oauthStateCache = cache
	if s.oauthProviders == nil {
		s.oauthProviders = map[string]oauthProviderConfig{}
	}
	s.oauthProviders[name] = oauthProviderConfig{ClientID: clientID, CallbackURL: callbackURL, Provider: provider}
}

func (s *Service) ConfigureGitHubOAuth(cache domain.OAuthStateCache, provider domain.OAuthProvider, clientID, callbackURL string) {
	s.ConfigureOAuthProvider(cache, "github", provider, clientID, callbackURL)
}

func (s *Service) OAuthProviderConfigured(name string) bool {
	cfg, ok := s.oauthProviders[name]
	return ok && cfg.Provider != nil && s.oauthStateCache != nil && cfg.ClientID != "" && cfg.CallbackURL != ""
}

func (s *Service) OAuthProviderDisplayName(name string) string {
	if cfg, ok := s.oauthProviders[name]; ok && cfg.Provider != nil {
		return cfg.Provider.DisplayName()
	}
	return name
}

func (s *Service) StartOAuth(ctx context.Context, name, callbackURL string) (string, error) {
	cfg, ok := s.oauthProviders[name]
	if !ok || !s.OAuthProviderConfigured(name) {
		return "", fmt.Errorf("%s auth provider is not configured", name)
	}
	if callbackURL == "" {
		callbackURL = cfg.CallbackURL
	}
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", err
	}
	state := hex.EncodeToString(stateBytes)
	location, stateData, ttlSeconds, err := cfg.Provider.Begin(ctx, state, callbackURL)
	if err != nil {
		return "", err
	}
	if stateData == nil {
		stateData = map[string]string{}
	}
	stateData["callback_url"] = callbackURL
	if ttlSeconds <= 0 {
		ttlSeconds = 1200
	}
	if err := s.oauthStateCache.Set(ctx, "auth:"+name+":state:"+state, stateData, ttlSeconds); err != nil {
		return "", err
	}
	return location, nil
}

func (s *Service) CompleteOAuth(ctx context.Context, name, state, code, actorType string, params map[string]string) (LoginResult, error) {
	cfg, ok := s.oauthProviders[name]
	if !ok || !s.OAuthProviderConfigured(name) {
		return LoginResult{}, fmt.Errorf("%s auth provider is not configured", name)
	}
	if state == "" || code == "" {
		return LoginResult{}, errors.New("No code provided")
	}
	key := "auth:" + name + ":state:" + state
	value, found, err := s.oauthStateCache.Get(ctx, key)
	if err != nil {
		return LoginResult{}, err
	}
	if !found || value == nil {
		return LoginResult{}, errors.New("No state provided, or session expired")
	}
	if err = s.oauthStateCache.Invalidate(ctx, key); err != nil {
		return LoginResult{}, err
	}
	stateData, ok := value.(map[string]string)
	if !ok || stateData["callback_url"] == "" {
		return LoginResult{}, errors.New("No state provided, or session expired")
	}
	if params["state"] != "" && params["state"] != state {
		return LoginResult{}, errors.New("No state provided, or session expired")
	}
	entityID, providerMetadata, userMetadata, err := cfg.Provider.Exchange(ctx, code, stateData["callback_url"], stateData, params)
	if err != nil {
		return LoginResult{}, err
	}
	identity, err := s.repository.FindOAuthIdentity(ctx, name, entityID)
	if errors.Is(err, domain.ErrIdentityNotFound) {
		identity, err = s.repository.CreateOAuthIdentity(ctx, name, entityID, providerMetadata, userMetadata)
	} else if err == nil && cfg.Provider.UpdateExistingIdentity() {
		if cfg.Provider.UpdateUserMetadataOnly() {
			identity, err = s.repository.UpdateOAuthUserMetadata(ctx, name, entityID, userMetadata)
		} else {
			identity, err = s.repository.UpdateOAuthIdentity(ctx, name, entityID, providerMetadata, userMetadata)
		}
	}
	if err != nil {
		return LoginResult{}, err
	}
	appMetadata, userData := map[string]any{}, map[string]any{}
	if err = json.Unmarshal(identity.AppMetadata, &appMetadata); err != nil {
		return LoginResult{}, err
	}
	if err = json.Unmarshal(identity.UserMetadata, &userData); err != nil {
		return LoginResult{}, err
	}
	challenge, err := s.CreateAuthMFAChallenge(ctx, identity.IdentityID, name)
	if err != nil {
		return LoginResult{}, err
	}
	if challenge != nil {
		signer, ok := s.tokens.(domain.RefreshTokenManager)
		if !ok {
			return LoginResult{}, errors.New("token signing does not support MFA claims")
		}
		delete(appMetadata, actorType+"_id")
		token, expires, issueErr := signer.IssueRefreshed(identity.IdentityID, actorType, name, "", appMetadata, userData, true, nil)
		if issueErr != nil {
			return LoginResult{}, issueErr
		}
		return LoginResult{Token: token, AuthIdentityID: identity.IdentityID, ExpiresAt: expires, MFARequired: true, MFAChallenge: challenge}, nil
	}
	actorID, _ := appMetadata[actorType+"_id"].(string)
	token, expires, err := s.tokens.Issue(identity.IdentityID, actorType, name, actorID, appMetadata, userData)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, AuthIdentityID: identity.IdentityID, ActorID: actorID, ExpiresAt: expires}, nil
}

func (s *Service) CreateIdentity(ctx context.Context, email, password string) (string, error) {
	return s.CreateIdentityWithProvider(ctx, "emailpass", email, password)
}

func (s *Service) CreateIdentityWithProvider(ctx context.Context, provider, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", errors.New("email and password are required")
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return "", err
	}
	return s.repository.CreateIdentityWithProvider(ctx, provider, email, hash)
}

func (s *Service) UpdateProviderPasswordForEntity(ctx context.Context, provider, actorType, entityID, password string) error {
	credential, err := s.repository.FindProviderCredential(ctx, provider, entityID, actorType)
	if err != nil {
		return err
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return err
	}
	return s.repository.UpdateProviderPassword(ctx, credential.ProviderIdentityID, hash)
}

func (s *Service) DeleteIdentity(ctx context.Context, id string) error {
	return s.repository.DeleteIdentity(ctx, id)
}

func (s *Service) FindIdentityByActor(ctx context.Context, actorType, actorID string) (domain.AuthIdentityRecord, error) {
	return s.repository.FindIdentityByActor(ctx, actorType, actorID)
}

func (s *Service) ListProviderNamesByActor(ctx context.Context, actorType, actorID string) ([]string, error) {
	return s.repository.ListProviderNamesByActor(ctx, actorType, actorID)
}

func (s *Service) ListProviderNamesByIdentity(ctx context.Context, identityID string) ([]string, error) {
	return s.repository.ListProviderNamesByIdentity(ctx, identityID)
}

func (s *Service) CreatePasswordResetToken(ctx context.Context, provider, entityID string, ttl time.Duration) (string, time.Time, error) {
	return s.repository.CreatePasswordResetToken(ctx, provider, entityID, ttl)
}
func (s *Service) ConsumePasswordResetToken(ctx context.Context, jti, provider, entityID string) error {
	return s.repository.ConsumePasswordResetToken(ctx, jti, provider, entityID)
}
func (s *Service) IssuePasswordResetToken(entityID, provider, actorType, jti string, expiresAt time.Time) (string, error) {
	signer, ok := s.tokens.(domain.PasswordResetTokenSigner)
	if !ok {
		return "", errors.New("password reset token signing is not supported")
	}
	return signer.IssuePasswordResetToken(entityID, provider, actorType, jti, expiresAt)
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	return s.Authenticate(ctx, email, password, "customer")
}

func (s *Service) Authenticate(ctx context.Context, email, password, actorType string) (LoginResult, error) {
	return s.AuthenticateProvider(ctx, "emailpass", email, password, actorType)
}

func (s *Service) AuthenticateProvider(ctx context.Context, provider, email, password, actorType string) (LoginResult, error) {
	credential, err := s.repository.FindProviderCredential(ctx, provider, email, actorType)
	if err != nil {
		return LoginResult{}, errors.New("invalid email or password")
	}
	valid, legacy, err := s.passwords.Verify(credential.PasswordHash, password)
	if err != nil || !valid {
		return LoginResult{}, errors.New("invalid email or password")
	}
	if legacy {
		updated, err := s.passwords.Hash(password)
		if err != nil {
			return LoginResult{}, err
		}
		if err := s.repository.UpdateProviderPassword(ctx, credential.ProviderIdentityID, updated); err != nil {
			return LoginResult{}, err
		}
	}
	appMetadata, userMetadata := map[string]any{}, map[string]any{}
	_ = json.Unmarshal(credential.AppMetadata, &appMetadata)
	_ = json.Unmarshal(credential.UserMetadata, &userMetadata)
	challenge, err := s.CreateAuthMFAChallenge(ctx, credential.IdentityID, provider)
	if err != nil {
		return LoginResult{}, err
	}
	if challenge != nil {
		signer, ok := s.tokens.(domain.RefreshTokenManager)
		if !ok {
			return LoginResult{}, errors.New("token signing does not support MFA claims")
		}
		delete(appMetadata, actorType+"_id")
		token, expires, issueErr := signer.IssueRefreshed(credential.IdentityID, actorType, provider, "", appMetadata, userMetadata, true, nil)
		if issueErr != nil {
			return LoginResult{}, issueErr
		}
		return LoginResult{Token: token, AuthIdentityID: credential.IdentityID, ExpiresAt: expires, MFARequired: true, MFAChallenge: challenge}, nil
	}
	token, expires, err := s.tokens.Issue(credential.IdentityID, actorType, provider, credential.ActorID, appMetadata, userMetadata)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, AuthIdentityID: credential.IdentityID, ActorID: credential.ActorID, ExpiresAt: expires}, nil
}

func (s *Service) RegistrationToken(identityID, actorType, provider string) (LoginResult, error) {
	token, expires, err := s.tokens.Issue(identityID, actorType, provider, "", map[string]any{}, map[string]any{})
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, AuthIdentityID: identityID, ExpiresAt: expires}, nil
}

func (s *Service) ParseToken(token string) (*domain.TokenClaims, error) {
	return s.tokens.Parse(token)
}

func (s *Service) RefreshActorToken(ctx context.Context, claims *domain.TokenClaims) (LoginResult, error) {
	if claims == nil || claims.AuthIdentityID == "" || claims.ActorID == "" || claims.ActorType == "" || claims.Provider == "" {
		return LoginResult{}, errors.New("actor-linked token and auth provider are required")
	}
	identity, err := s.repository.RetrieveAuthIdentity(ctx, claims.AuthIdentityID, claims.Provider)
	if err != nil {
		return LoginResult{}, err
	}
	appMetadata, userMetadata := map[string]any{}, map[string]any{}
	if err = json.Unmarshal(identity.AppMetadata, &appMetadata); err != nil {
		return LoginResult{}, err
	}
	if err = json.Unmarshal(identity.UserMetadata, &userMetadata); err != nil {
		return LoginResult{}, err
	}
	signer, ok := s.tokens.(domain.RefreshTokenManager)
	if !ok {
		return LoginResult{}, errors.New("token refresh is not supported")
	}
	token, expiresAt, err := signer.IssueRefreshed(identity.ID, claims.ActorType, claims.Provider, claims.ActorID, appMetadata, userMetadata, claims.MFAEnabled, claims.MFAChallengeCompletedAt)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, AuthIdentityID: identity.ID, ActorID: claims.ActorID, ExpiresAt: expiresAt}, nil
}

func (s *Service) RequestAuthVerification(ctx context.Context, input models.RequestAuthVerificationInput, ttl time.Duration) (models.AuthVerification, error) {
	if input.CodeProvider != "token" {
		return models.AuthVerification{}, errors.New("verification provider is not registered")
	}
	return s.repository.RequestAuthVerification(ctx, input, ttl)
}

func (s *Service) ConfirmAuthVerification(ctx context.Context, input models.ConfirmAuthVerificationInput, ttl time.Duration) (models.AuthVerification, error) {
	if input.CodeProvider != "" && input.CodeProvider != "token" {
		return models.AuthVerification{}, errors.New("verification provider is not registered")
	}
	return s.repository.ConfirmAuthVerification(ctx, input, ttl)
}

func (s *Service) ListAuthMFA(ctx context.Context, identityID string) ([]models.AuthMFAFactor, error) {
	factors, err := s.repository.ListAuthMFA(ctx, identityID, "")
	for i := range factors {
		factors[i].ProviderMetadata = nil
	}
	return factors, err
}

func (s *Service) GetAuthMFA(ctx context.Context, factorID, identityID string) (models.AuthMFAFactor, error) {
	factor, err := s.repository.RetrieveAuthMFA(ctx, factorID, identityID)
	factor.ProviderMetadata = nil
	return factor, err
}

func (s *Service) CreateAuthMFAChallenge(ctx context.Context, identityID, provider string) (*models.AuthMFAChallenge, error) {
	factors, err := s.repository.ListAuthMFA(ctx, identityID, "enabled")
	if err != nil {
		return nil, err
	}
	methods := []string{}
	seen := map[string]bool{}
	for _, factor := range factors {
		if factor.Provider == "totp" && s.totp != nil && !seen[factor.Provider] {
			methods = append(methods, factor.Provider)
			seen[factor.Provider] = true
		}
	}
	if len(methods) == 0 {
		return nil, nil
	}
	if s.challengeCache == nil {
		return nil, errors.New("Cache module dependency is required when using MFA challenges")
	}
	recoveryCodes, err := s.repository.ListAuthMFARecoveryCodes(ctx, identityID)
	if err != nil {
		return nil, err
	}
	if len(recoveryCodes) > 0 {
		methods = append(methods, "recovery_code")
	}
	now := time.Now().UTC()
	challenge := &models.AuthMFAChallenge{ID: entityid.New("authmfachal"), AuthIdentityID: identityID, AuthProvider: provider, Methods: methods, ExpiresAt: now.Add(time.Duration(s.challengeTTL) * time.Second), Attempts: 0, MaxAttempts: s.challengeAttempts, Metadata: json.RawMessage("null")}
	if err = s.challengeCache.Set(ctx, "auth:mfa:challenge:"+challenge.ID, challenge, s.challengeTTL); err != nil {
		return nil, err
	}
	return challenge, nil
}

func (s *Service) VerifyAuthMFAChallenge(ctx context.Context, id, identityID, method, code string) (models.AuthMFAChallenge, error) {
	if s.challengeCache == nil {
		return models.AuthMFAChallenge{}, errors.New("Cache module dependency is required when using MFA challenges")
	}
	key := "auth:mfa:challenge:" + id
	value, found, err := s.challengeCache.Get(ctx, key)
	if err != nil {
		return models.AuthMFAChallenge{}, err
	}
	if !found || value == nil {
		return models.AuthMFAChallenge{}, fmt.Errorf("MFA challenge with id %q was not found", id)
	}
	serialized, err := json.Marshal(value)
	if err != nil {
		return models.AuthMFAChallenge{}, err
	}
	var challenge models.AuthMFAChallenge
	if err = json.Unmarshal(serialized, &challenge); err != nil {
		return models.AuthMFAChallenge{}, err
	}
	if identityID != "" && challenge.AuthIdentityID != identityID {
		return models.AuthMFAChallenge{}, fmt.Errorf("MFA challenge with id %q was not found", id)
	}
	if challenge.CompletedAt != nil {
		return models.AuthMFAChallenge{}, errors.New("MFA challenge has already been completed")
	}
	if !challenge.ExpiresAt.After(time.Now()) {
		return models.AuthMFAChallenge{}, errors.New("MFA challenge has expired")
	}
	if challenge.Attempts >= challenge.MaxAttempts {
		return models.AuthMFAChallenge{}, errors.New("MFA challenge has too many failed attempts")
	}
	supported := false
	for _, supportedMethod := range challenge.Methods {
		if supportedMethod == method {
			supported = true
			break
		}
	}
	if !supported {
		return models.AuthMFAChallenge{}, fmt.Errorf("MFA challenge does not support method %q", method)
	}
	valid, err := s.VerifyAuthMFACode(ctx, challenge.AuthIdentityID, method, code)
	if err != nil {
		return models.AuthMFAChallenge{}, err
	}
	if !valid {
		challenge.Attempts++
		if err = s.saveMFAChallenge(ctx, key, challenge); err != nil {
			return models.AuthMFAChallenge{}, err
		}
		if challenge.Attempts >= challenge.MaxAttempts {
			return models.AuthMFAChallenge{}, errors.New("MFA challenge has too many failed attempts")
		}
		return models.AuthMFAChallenge{}, errors.New("Invalid MFA challenge code")
	}
	now := time.Now().UTC()
	challenge.CompletedAt = &now
	if err = s.saveMFAChallenge(ctx, key, challenge); err != nil {
		return models.AuthMFAChallenge{}, err
	}
	return challenge, nil
}

func (s *Service) saveMFAChallenge(ctx context.Context, key string, challenge models.AuthMFAChallenge) error {
	seconds := int(time.Until(challenge.ExpiresAt).Seconds())
	if time.Duration(seconds)*time.Second < time.Until(challenge.ExpiresAt) {
		seconds++
	}
	if seconds < 1 {
		return nil
	}
	return s.challengeCache.Set(ctx, key, challenge, seconds)
}

func (s *Service) IssueMFAChallengeToken(ctx context.Context, identityID, actorType, provider string, challenge models.AuthMFAChallenge) (LoginResult, error) {
	if provider == "" {
		provider = challenge.AuthProvider
	}
	if provider == "" || challenge.AuthIdentityID != identityID || challenge.CompletedAt == nil {
		return LoginResult{}, errors.New("completed MFA challenge and auth provider are required")
	}
	identity, err := s.repository.RetrieveAuthIdentity(ctx, identityID, provider)
	if err != nil {
		return LoginResult{}, err
	}
	appMetadata, userMetadata := map[string]any{}, map[string]any{}
	if err = json.Unmarshal(identity.AppMetadata, &appMetadata); err != nil {
		return LoginResult{}, err
	}
	if err = json.Unmarshal(identity.UserMetadata, &userMetadata); err != nil {
		return LoginResult{}, err
	}
	actorID, _ := appMetadata[actorType+"_id"].(string)
	signer, ok := s.tokens.(domain.RefreshTokenManager)
	if !ok {
		return LoginResult{}, errors.New("token signing does not support MFA claims")
	}
	token, expiresAt, err := signer.IssueRefreshed(identity.ID, actorType, provider, actorID, appMetadata, userMetadata, true, challenge.CompletedAt)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, AuthIdentityID: identity.ID, ActorID: actorID, ExpiresAt: expiresAt}, nil
}

func (s *Service) StartAuthMFA(ctx context.Context, input models.AuthMFAStartInput) (models.AuthMFAEnrollment, error) {
	if s.totp == nil || input.Provider != "totp" {
		return models.AuthMFAEnrollment{}, errors.New("MFA provider is not registered")
	}
	for _, status := range []string{"pending", "enabled"} {
		factors, err := s.repository.ListAuthMFA(ctx, input.AuthIdentityID, status)
		if err != nil {
			return models.AuthMFAEnrollment{}, err
		}
		for _, factor := range factors {
			if factor.Provider == "totp" {
				return models.AuthMFAEnrollment{}, errors.New("An active TOTP factor already exists for this auth identity")
			}
		}
	}
	encrypted, secret, issuer, uri, err := s.totp.Start(input)
	if err != nil {
		return models.AuthMFAEnrollment{}, err
	}
	providerMetadata, err := json.Marshal(map[string]string{"secret": encrypted, "issuer": issuer})
	if err != nil {
		return models.AuthMFAEnrollment{}, err
	}
	metadata := map[string]any{}
	if len(input.Metadata) > 0 && string(input.Metadata) != "null" {
		if err = json.Unmarshal(input.Metadata, &metadata); err != nil {
			return models.AuthMFAEnrollment{}, err
		}
	}
	if input.Label != nil {
		metadata["label"] = *input.Label
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return models.AuthMFAEnrollment{}, err
	}
	factor, err := s.repository.CreateAuthMFA(ctx, models.AuthMFAFactor{AuthIdentityID: input.AuthIdentityID, Provider: input.Provider, Status: "pending", ProviderMetadata: providerMetadata, Metadata: metadataJSON})
	if err != nil {
		return models.AuthMFAEnrollment{}, err
	}
	factor.ProviderMetadata = nil
	return models.AuthMFAEnrollment{MFA: factor, Secret: secret, OTPAuthURL: uri}, nil
}

func (s *Service) VerifyAuthMFASetup(ctx context.Context, factorID, identityID, code string) (models.AuthMFAFactor, error) {
	factor, err := s.repository.RetrieveAuthMFA(ctx, factorID, identityID)
	if err != nil {
		return models.AuthMFAFactor{}, err
	}
	if factor.Provider != "totp" {
		return models.AuthMFAFactor{}, errors.New("Only TOTP MFA factors can be verified with this method")
	}
	if factor.Status == "disabled" {
		return models.AuthMFAFactor{}, errors.New("Disabled MFA factors cannot be verified")
	}
	var providerMetadata struct {
		Secret string `json:"secret"`
	}
	if err = json.Unmarshal(factor.ProviderMetadata, &providerMetadata); err != nil {
		return models.AuthMFAFactor{}, err
	}
	valid, err := s.totp.Verify(providerMetadata.Secret, code)
	if err != nil {
		return models.AuthMFAFactor{}, err
	}
	if !valid {
		return models.AuthMFAFactor{}, errors.New("Invalid TOTP code")
	}
	if factor.Status == "pending" {
		factor, err = s.repository.UpdateAuthMFA(ctx, factorID, identityID, "enabled")
		if err != nil {
			return models.AuthMFAFactor{}, err
		}
	}
	factor.ProviderMetadata = nil
	return factor, nil
}

func (s *Service) DisableAuthMFA(ctx context.Context, factorID, identityID, method, code string) (models.AuthMFAFactor, error) {
	factor, err := s.repository.RetrieveAuthMFA(ctx, factorID, identityID)
	if err != nil {
		return models.AuthMFAFactor{}, err
	}
	if factor.Status == "enabled" && s.mfaDisablePolicy == "challenge" {
		if method == "" || code == "" {
			return models.AuthMFAFactor{}, errors.New("MFA verification code is required to disable MFA")
		}
		if method != factor.Provider {
			return models.AuthMFAFactor{}, errors.New("Invalid MFA verification method")
		}
		var providerMetadata struct {
			Secret string `json:"secret"`
		}
		if err = json.Unmarshal(factor.ProviderMetadata, &providerMetadata); err != nil {
			return models.AuthMFAFactor{}, err
		}
		valid, verifyErr := s.totp.Verify(providerMetadata.Secret, code)
		if verifyErr != nil {
			return models.AuthMFAFactor{}, verifyErr
		}
		if !valid {
			return models.AuthMFAFactor{}, errors.New("Invalid MFA verification code")
		}
	}
	factor, err = s.repository.UpdateAuthMFA(ctx, factorID, identityID, "disabled")
	factor.ProviderMetadata = nil
	return factor, err
}

func (s *Service) VerifyAuthMFACode(ctx context.Context, identityID, method, code string) (bool, error) {
	if method == "recovery_code" {
		err := s.UseAuthMFARecoveryCode(ctx, identityID, code)
		if errors.Is(err, domain.ErrInvalidMFARecoveryCode) {
			return false, nil
		}
		return err == nil, err
	}
	factors, err := s.repository.ListAuthMFA(ctx, identityID, "enabled")
	if err != nil {
		return false, err
	}
	for _, factor := range factors {
		if factor.Provider != method {
			continue
		}
		var providerMetadata struct {
			Secret string `json:"secret"`
		}
		if err = json.Unmarshal(factor.ProviderMetadata, &providerMetadata); err != nil {
			return false, err
		}
		return s.totp.Verify(providerMetadata.Secret, code)
	}
	return false, nil
}

func (s *Service) GenerateAuthMFARecoveryCodes(ctx context.Context, identityID string, count int) ([]string, error) {
	if count == 0 {
		count = 10
	}
	if count < 1 || count > 50 {
		return nil, errors.New("Recovery code count must be between 1 and 50")
	}
	factors, err := s.repository.ListAuthMFA(ctx, identityID, "enabled")
	if err != nil {
		return nil, err
	}
	if len(factors) == 0 {
		return nil, errors.New("Recovery codes require an enabled MFA factor")
	}
	codes := make([]string, count)
	hashes := make([]string, count)
	for i := range codes {
		randomBytes := make([]byte, 16)
		if _, err = rand.Read(randomBytes); err != nil {
			return nil, err
		}
		raw := hex.EncodeToString(randomBytes)
		codes[i] = fmt.Sprintf("%s-%s-%s-%s-%s-%s-%s-%s", raw[0:4], raw[4:8], raw[8:12], raw[12:16], raw[16:20], raw[20:24], raw[24:28], raw[28:32])
		hashes[i], err = s.passwords.Hash(normalizeRecoveryCode(codes[i]))
		if err != nil {
			return nil, err
		}
	}
	if err = s.repository.ReplaceAuthMFARecoveryCodes(ctx, identityID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *Service) UseAuthMFARecoveryCode(ctx context.Context, identityID, code string) error {
	codes, err := s.repository.ListAuthMFARecoveryCodes(ctx, identityID)
	if err != nil {
		return err
	}
	for _, candidate := range codes {
		valid, _, verifyErr := s.passwords.Verify(candidate.CodeHash, normalizeRecoveryCode(code))
		if verifyErr != nil {
			return verifyErr
		}
		if !valid {
			continue
		}
		deleted, deleteErr := s.repository.DeleteAuthMFARecoveryCode(ctx, candidate.ID)
		if deleteErr != nil {
			return deleteErr
		}
		if !deleted {
			return domain.ErrInvalidMFARecoveryCode
		}
		return nil
	}
	return domain.ErrInvalidMFARecoveryCode
}

func normalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func (s *Service) SetActorMetadata(ctx context.Context, identityID, actorType, actorID string) (string, error) {
	return s.repository.SetActorMetadata(ctx, identityID, actorType, actorID)
}

func (s *Service) UnsetActorMetadata(ctx context.Context, identityID, key string) error {
	return s.repository.UnsetActorMetadata(ctx, identityID, key)
}

var _ domain.IdentityMetadata = (*Service)(nil)
