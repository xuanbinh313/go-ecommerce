package google

import (
	"context"
	"crypto"
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
)

const jwksURL = "https://www.googleapis.com/oauth2/v3/certs"

type Provider struct {
	clientID     string
	clientSecret string
	client       *http.Client
	mu           sync.Mutex
	keys         map[string]*rsa.PublicKey
	keysExpireAt time.Time
}

func New(clientID, clientSecret string) *Provider {
	return &Provider{clientID: clientID, clientSecret: clientSecret, client: &http.Client{Timeout: 20 * time.Second}}
}

func (p *Provider) AuthorizationURL(state, callbackURL string) string {
	u, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	q := u.Query()
	q.Set("redirect_uri", callbackURL)
	q.Set("client_id", p.clientID)
	q.Set("response_type", "code")
	q.Set("scope", "email profile openid")
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

func (p *Provider) UpdateExistingIdentity() bool { return false }
func (p *Provider) UpdateUserMetadataOnly() bool { return false }
func (p *Provider) DisplayName() string          { return "Google Authentication" }

func (p *Provider) Begin(_ context.Context, state, callbackURL string) (string, map[string]string, int, error) {
	return p.AuthorizationURL(state, callbackURL), map[string]string{}, 1200, nil
}

func (p *Provider) Exchange(ctx context.Context, code, callbackURL string, _ map[string]string, _ map[string]string) (string, json.RawMessage, json.RawMessage, error) {
	u, _ := url.Parse("https://oauth2.googleapis.com/token")
	q := u.Query()
	q.Set("client_id", p.clientID)
	q.Set("client_secret", p.clientSecret)
	q.Set("code", code)
	q.Set("redirect_uri", callbackURL)
	q.Set("grant_type", "authorization_code")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return "", nil, nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, nil, fmt.Errorf("Could not exchange token, %d, %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	var exchange struct {
		IDToken string `json:"id_token"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&exchange); err != nil {
		return "", nil, nil, err
	}
	if exchange.IDToken == "" {
		return "", nil, nil, errors.New("No ID found")
	}
	claims, err := p.verifyIDToken(ctx, exchange.IDToken)
	if err != nil {
		return "", nil, nil, err
	}
	emailVerified, _ := claims["email_verified"].(bool)
	if !emailVerified {
		return "", nil, nil, errors.New("Email not verified, cannot proceed with authentication")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", nil, nil, errors.New("id_token is missing 'sub' claim")
	}
	metadata := map[string]any{}
	for _, field := range []string{"name", "email", "picture", "given_name", "family_name"} {
		if value := claims[field]; value != nil {
			metadata[field] = value
		}
	}
	userMetadata, err := json.Marshal(metadata)
	if err != nil {
		return "", nil, nil, err
	}
	return sub, json.RawMessage(`{}`), userMetadata, nil
}

func (p *Provider) verifyIDToken(ctx context.Context, raw string) (map[string]any, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, errors.New("Could not verify Google id_token: malformed token")
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "RS256" || header.KeyID == "" {
		return nil, errors.New("Could not verify Google id_token: ID token is missing a valid RS256 kid header")
	}
	key, err := p.signingKey(ctx, header.KeyID)
	if err != nil {
		return nil, fmt.Errorf("Could not verify Google id_token: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("Could not verify Google id_token: %w", err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err = rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return nil, fmt.Errorf("Could not verify Google id_token: %w", err)
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	claims := map[string]any{}
	if err = json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}
	if claims["iss"] != "https://accounts.google.com" && claims["iss"] != "accounts.google.com" {
		return nil, errors.New("Could not verify Google id_token: invalid issuer")
	}
	if !audienceContains(claims["aud"], p.clientID) {
		return nil, errors.New("Could not verify Google id_token: invalid audience")
	}
	exp, ok := claims["exp"].(float64)
	if !ok || !time.Unix(int64(exp), 0).After(time.Now()) {
		return nil, errors.New("Could not verify Google id_token: token is expired")
	}
	if nbf, exists := claims["nbf"]; exists {
		issuedAfter, valid := nbf.(float64)
		if !valid || time.Unix(int64(issuedAfter), 0).After(time.Now()) {
			return nil, errors.New("Could not verify Google id_token: token is not valid yet")
		}
	}
	return claims, nil
}

func audienceContains(value any, clientID string) bool {
	switch aud := value.(type) {
	case string:
		return aud == clientID
	case []any:
		for _, item := range aud {
			if item == clientID {
				return true
			}
		}
	}
	return false
}

func (p *Provider) signingKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	p.mu.Lock()
	if time.Now().Before(p.keysExpireAt) {
		key := p.keys[kid]
		p.mu.Unlock()
		if key != nil {
			return key, nil
		}
	}
	p.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Google JWKS request failed: %s", resp.Status)
	}
	var set struct {
		Keys []struct {
			Kid string   `json:"kid"`
			Kty string   `json:"kty"`
			N   string   `json:"n"`
			E   string   `json:"e"`
			X5C []string `json:"x5c"`
		} `json:"keys"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, jwk := range set.Keys {
		if jwk.Kid == "" {
			continue
		}
		if len(jwk.X5C) > 0 {
			certBytes, decodeErr := base64.StdEncoding.DecodeString(jwk.X5C[0])
			if decodeErr != nil {
				continue
			}
			cert, parseErr := x509.ParseCertificate(certBytes)
			if parseErr != nil {
				continue
			}
			if publicKey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
				keys[jwk.Kid] = publicKey
			}
			continue
		}
		if jwk.Kty != "RSA" {
			continue
		}
		modulus, decodeErr := base64.RawURLEncoding.DecodeString(jwk.N)
		if decodeErr != nil {
			continue
		}
		exponentBytes, decodeErr := base64.RawURLEncoding.DecodeString(jwk.E)
		if decodeErr != nil {
			continue
		}
		exponent := 0
		for _, b := range exponentBytes {
			exponent = exponent<<8 | int(b)
		}
		if exponent > 0 {
			keys[jwk.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: exponent}
		}
	}
	p.mu.Lock()
	p.keys, p.keysExpireAt = keys, time.Now().Add(time.Hour)
	key := p.keys[kid]
	p.mu.Unlock()
	if key == nil {
		return nil, fmt.Errorf("unable to resolve signing key %q", kid)
	}
	return key, nil
}

var _ domain.OAuthProvider = (*Provider)(nil)
