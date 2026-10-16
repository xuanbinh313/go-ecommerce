package oidc

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain"
	"github.com/golang-jwt/jwt/v5"
)

type Options struct {
	Issuer                string            `json:"issuer"`
	ClientID              string            `json:"client_id"`
	ClientSecret          string            `json:"client_secret,omitempty"`
	CallbackURL           string            `json:"callback_url"`
	AllowedCallbackURLs   []string          `json:"allowed_callback_urls,omitempty"`
	Scopes                []string          `json:"scopes,omitempty"`
	AuthorizationEndpoint string            `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string            `json:"token_endpoint,omitempty"`
	JWKSURI               string            `json:"jwks_uri,omitempty"`
	EndSessionEndpoint    string            `json:"end_session_endpoint,omitempty"`
	ClaimMappings         map[string]string `json:"claim_mappings,omitempty"`
	RequireVerifiedEmail  *bool             `json:"require_verified_email,omitempty"`
	AllowedEmailDomains   []string          `json:"allowed_email_domains,omitempty"`
	ClockToleranceSeconds int               `json:"clock_tolerance_seconds,omitempty"`
	StateTTLSeconds       int               `json:"state_ttl_seconds,omitempty"`
	DisplayName           string            `json:"display_name,omitempty"`
}

type Provider struct {
	options       Options
	client        *http.Client
	production    bool
	mu            sync.Mutex
	metadata      map[string]string
	metadataUntil time.Time
	keys          map[string]crypto.PublicKey
	keysUntil     time.Time
}

func New(options Options, production bool) (*Provider, error) {
	if strings.TrimSpace(options.Issuer) == "" || strings.TrimSpace(options.ClientID) == "" || strings.TrimSpace(options.CallbackURL) == "" {
		return nil, errors.New("OIDC provider requires issuer, client_id, and callback_url")
	}
	if err := assertSecureURL(options.Issuer, "issuer", production); err != nil {
		return nil, err
	}
	for label, endpoint := range map[string]string{"authorization_endpoint": options.AuthorizationEndpoint, "token_endpoint": options.TokenEndpoint, "jwks_uri": options.JWKSURI, "end_session_endpoint": options.EndSessionEndpoint} {
		if endpoint != "" {
			if err := assertSecureURL(endpoint, label, production); err != nil {
				return nil, err
			}
		}
	}
	if options.StateTTLSeconds <= 0 {
		options.StateTTLSeconds = 600
	}
	if options.ClockToleranceSeconds == 0 {
		options.ClockToleranceSeconds = 5
	}
	return &Provider{options: options, production: production, client: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (p *Provider) Begin(ctx context.Context, state, callbackURL string) (location string, stateData map[string]string, ttlSeconds int, err error) {
	if callbackURL == "" {
		callbackURL = p.options.CallbackURL
	}
	if len(p.options.AllowedCallbackURLs) > 0 && !contains(p.options.AllowedCallbackURLs, callbackURL) {
		return "", nil, 0, errors.New("The provided callback URL is not allowed")
	}
	defer func() {
		if err != nil {
			err = errors.New("Authentication failed")
		}
	}()
	metadata, err := p.resolveMetadata(ctx)
	if err != nil {
		return "", nil, 0, err
	}
	verifierBytes, nonceBytes := make([]byte, 32), make([]byte, 32)
	if _, err = rand.Read(verifierBytes); err != nil {
		return "", nil, 0, err
	}
	if _, err = rand.Read(nonceBytes); err != nil {
		return "", nil, 0, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	challengeDigest := sha256.Sum256([]byte(verifier))
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	scopes := p.options.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}
	u, err := url.Parse(metadata["authorization_endpoint"])
	if err != nil {
		return "", nil, 0, err
	}
	query := u.Query()
	query.Set("client_id", p.options.ClientID)
	query.Set("redirect_uri", callbackURL)
	query.Set("response_type", "code")
	query.Set("scope", strings.Join(scopes, " "))
	query.Set("state", state)
	query.Set("nonce", nonce)
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challengeDigest[:]))
	query.Set("code_challenge_method", "S256")
	u.RawQuery = query.Encode()
	return u.String(), map[string]string{"nonce": nonce, "code_verifier": verifier}, p.options.StateTTLSeconds, nil
}

func (p *Provider) Exchange(ctx context.Context, code, callbackURL string, stateData, responseParams map[string]string) (entityID string, providerMetadata, userMetadata json.RawMessage, err error) {
	if responseParams["state"] == "" {
		return "", nil, nil, errors.New("No state provided, or session expired")
	}
	defer func() {
		if err != nil {
			err = errors.New("Authentication failed")
		}
	}()
	if callbackURL == "" {
		callbackURL = p.options.CallbackURL
	}
	metadata, err := p.resolveMetadata(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {callbackURL}, "client_id": {p.options.ClientID}, "code_verifier": {stateData["code_verifier"]}}
	if p.options.ClientSecret != "" {
		form.Set("client_secret", p.options.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata["token_endpoint"], strings.NewReader(form.Encode()))
	if err != nil {
		return "", nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, nil, fmt.Errorf("Could not exchange token, %d, %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	var tokenResponse struct {
		IDToken string `json:"id_token"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", nil, nil, err
	}
	if tokenResponse.IDToken == "" {
		return "", nil, nil, errors.New("The identity provider did not return an ID token; ensure the 'openid' scope is requested")
	}
	claims, err := p.verifyIDToken(ctx, tokenResponse.IDToken, stateData["nonce"])
	if err != nil {
		return "", nil, nil, err
	}
	mappings := map[string]string{"entity_id": "sub", "email": "email", "name": "name"}
	for target, claim := range p.options.ClaimMappings {
		if claim == "" {
			delete(mappings, target)
		} else {
			mappings[target] = claim
		}
	}
	entityClaim := mappings["entity_id"]
	if entityClaim == "" {
		entityClaim = "sub"
	}
	entityClaimValue := claims[entityClaim]
	if entityClaimValue == nil || entityClaimValue == "" {
		return "", nil, nil, fmt.Errorf("The identity provider's ID token is missing the '%s' claim used to identify the user", entityClaim)
	}
	requireVerified := p.options.RequireVerifiedEmail == nil || *p.options.RequireVerifiedEmail
	if requireVerified && claims["email_verified"] != true {
		return "", nil, nil, errors.New("The identity provider did not confirm a verified email address")
	}
	if len(p.options.AllowedEmailDomains) > 0 {
		emailClaim := mappings["email"]
		if emailClaim == "" {
			emailClaim = "email"
		}
		email, _ := claims[emailClaim].(string)
		parts := strings.Split(email, "@")
		if len(parts) != 2 || !containsFold(p.options.AllowedEmailDomains, parts[1]) {
			return "", nil, nil, errors.New("The email domain is not allowed to authenticate with this provider")
		}
	}
	mappedUserMetadata := map[string]any{}
	for target, claim := range mappings {
		if target != "entity_id" && claim != "" && claims[claim] != nil {
			mappedUserMetadata[target] = claims[claim]
		}
	}
	metadataJSON, err := json.Marshal(mappedUserMetadata)
	if err != nil {
		return "", nil, nil, err
	}
	userMetadata = metadataJSON
	providerMetadata = json.RawMessage(`{}`)
	return fmt.Sprint(entityClaimValue), providerMetadata, userMetadata, nil
}

func (p *Provider) resolveMetadata(ctx context.Context) (map[string]string, error) {
	p.mu.Lock()
	if p.metadata != nil && time.Now().Before(p.metadataUntil) {
		metadata := p.metadata
		p.mu.Unlock()
		return metadata, nil
	}
	p.mu.Unlock()
	metadata := map[string]string{"issuer": p.options.Issuer, "authorization_endpoint": p.options.AuthorizationEndpoint, "token_endpoint": p.options.TokenEndpoint, "jwks_uri": p.options.JWKSURI}
	if metadata["authorization_endpoint"] == "" || metadata["token_endpoint"] == "" || metadata["jwks_uri"] == "" {
		issuerURL, err := url.Parse(strings.TrimRight(p.options.Issuer, "/"))
		if err != nil {
			return nil, err
		}
		issuerURL.Path = strings.TrimRight(issuerURL.Path, "/") + "/.well-known/openid-configuration"
		issuerURL.RawQuery, issuerURL.Fragment = "", ""
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuerURL.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("OIDC discovery request failed: %s", resp.Status)
		}
		var doc map[string]any
		if err = json.NewDecoder(resp.Body).Decode(&doc); err != nil {
			return nil, err
		}
		if discoveredIssuer, _ := doc["issuer"].(string); discoveredIssuer != p.options.Issuer {
			return nil, errors.New("OIDC discovery issuer does not match configured issuer")
		}
		for key := range metadata {
			if metadata[key] == "" {
				metadata[key], _ = doc[key].(string)
			}
		}
	}
	for _, key := range []string{"authorization_endpoint", "token_endpoint", "jwks_uri"} {
		if metadata[key] == "" {
			return nil, fmt.Errorf("OIDC discovery did not provide %s", key)
		}
		if err := assertSecureURL(metadata[key], key, p.production); err != nil {
			return nil, err
		}
	}
	p.mu.Lock()
	p.metadata, p.metadataUntil = metadata, time.Now().Add(time.Hour)
	p.mu.Unlock()
	return metadata, nil
}

func (p *Provider) verifyIDToken(ctx context.Context, raw, expectedNonce string) (map[string]any, error) {
	validMethods := []string{"RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512"}
	leeway := time.Duration(p.options.ClockToleranceSeconds) * time.Second
	token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("ID token is missing 'kid' header")
		}
		return p.signingKey(ctx, kid)
	}, jwt.WithValidMethods(validMethods), jwt.WithIssuer(p.options.Issuer), jwt.WithAudience(p.options.ClientID), jwt.WithLeeway(leeway), jwt.WithIssuedAt())
	if err != nil || token == nil || !token.Valid {
		return nil, fmt.Errorf("Could not validate the identity provider's response: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("Invalid ID token claims")
	}
	expiration, expirationErr := claims.GetExpirationTime()
	issuedAt, issuedAtErr := claims.GetIssuedAt()
	subject, _ := claims["sub"].(string)
	if expirationErr != nil || issuedAtErr != nil || expiration == nil || issuedAt == nil || subject == "" {
		return nil, errors.New("ID token is missing required claims")
	}
	if azp, ok := claims["azp"].(string); ok && azp != p.options.ClientID {
		return nil, errors.New("Invalid ID token authorized party")
	}
	if audiences, ok := claims["aud"].([]any); ok && len(audiences) > 1 && claims["azp"] != p.options.ClientID {
		return nil, errors.New("Invalid ID token authorized party")
	}
	if expectedNonce == "" || claims["nonce"] != expectedNonce {
		return nil, errors.New("Invalid ID token nonce")
	}
	result := map[string]any{}
	for key, value := range claims {
		result[key] = value
	}
	return result, nil
}

func (p *Provider) signingKey(ctx context.Context, kid string) (any, error) {
	p.mu.Lock()
	if time.Now().Before(p.keysUntil) {
		key := p.keys[kid]
		p.mu.Unlock()
		if key != nil {
			return key, nil
		}
	} else {
		p.mu.Unlock()
	}
	metadata, err := p.resolveMetadata(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadata["jwks_uri"], nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OIDC JWKS request failed: %s", resp.Status)
	}
	var jwks struct {
		Keys []struct {
			Kid string   `json:"kid"`
			Kty string   `json:"kty"`
			N   string   `json:"n"`
			E   string   `json:"e"`
			Crv string   `json:"crv"`
			X   string   `json:"x"`
			Y   string   `json:"y"`
			X5C []string `json:"x5c"`
		} `json:"keys"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}
	keys := map[string]crypto.PublicKey{}
	for _, key := range jwks.Keys {
		if key.Kid == "" {
			continue
		}
		if len(key.X5C) > 0 {
			der, decodeErr := base64.StdEncoding.DecodeString(key.X5C[0])
			if decodeErr == nil {
				cert, parseErr := x509.ParseCertificate(der)
				if parseErr == nil {
					keys[key.Kid] = cert.PublicKey
				}
			}
			continue
		}
		switch key.Kty {
		case "RSA":
			modulus, e1 := base64.RawURLEncoding.DecodeString(key.N)
			exponentBytes, e2 := base64.RawURLEncoding.DecodeString(key.E)
			if e1 == nil && e2 == nil {
				exponent := 0
				for _, b := range exponentBytes {
					exponent = exponent<<8 | int(b)
				}
				if exponent > 0 {
					keys[key.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: exponent}
				}
			}
		case "EC":
			curve := elliptic.P256()
			switch key.Crv {
			case "P-384":
				curve = elliptic.P384()
			case "P-521":
				curve = elliptic.P521()
			case "P-256":
			default:
				continue
			}
			x, e1 := base64.RawURLEncoding.DecodeString(key.X)
			y, e2 := base64.RawURLEncoding.DecodeString(key.Y)
			if e1 == nil && e2 == nil {
				keys[key.Kid] = &ecdsa.PublicKey{Curve: curve, X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}
			}
		}
	}
	p.mu.Lock()
	p.keys, p.keysUntil = keys, time.Now().Add(time.Hour)
	key := p.keys[kid]
	p.mu.Unlock()
	if key == nil {
		return nil, fmt.Errorf("unable to resolve OIDC signing key %q", kid)
	}
	return key, nil
}

func (p *Provider) UpdateExistingIdentity() bool { return true }
func (p *Provider) UpdateUserMetadataOnly() bool { return true }
func (p *Provider) DisplayName() string {
	if p.options.DisplayName != "" {
		return p.options.DisplayName
	}
	return "OpenID Connect"
}

func assertSecureURL(raw, label string, production bool) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("OIDC '%s' must be a valid URL", label)
	}
	localhost := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1" || u.Hostname() == "[::1]"
	if u.Scheme != "https" && !(u.Scheme == "http" && localhost && !production) {
		return fmt.Errorf("OIDC '%s' must use https (http is only allowed for localhost outside of production)", label)
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

var _ domain.OAuthProvider = (*Provider)(nil)
