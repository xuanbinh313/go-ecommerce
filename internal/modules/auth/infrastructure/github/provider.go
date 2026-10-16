package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/auth/domain"
)

type Provider struct {
	clientID     string
	clientSecret string
	client       *http.Client
}

func New(clientID, clientSecret string) *Provider {
	return &Provider{clientID: clientID, clientSecret: clientSecret, client: &http.Client{Timeout: 20 * time.Second}}
}

func (p *Provider) AuthorizationURL(state, callbackURL string) string {
	u, _ := url.Parse("https://github.com/login/oauth/authorize")
	q := u.Query()
	q.Set("redirect_uri", callbackURL)
	q.Set("client_id", p.clientID)
	q.Set("response_type", "code")
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

func (p *Provider) UpdateExistingIdentity() bool { return true }
func (p *Provider) UpdateUserMetadataOnly() bool { return false }
func (p *Provider) DisplayName() string          { return "Github Authentication" }

func (p *Provider) Begin(_ context.Context, state, callbackURL string) (string, map[string]string, int, error) {
	return p.AuthorizationURL(state, callbackURL), map[string]string{}, 1200, nil
}

func (p *Provider) Exchange(ctx context.Context, code, callbackURL string, _ map[string]string, _ map[string]string) (string, json.RawMessage, json.RawMessage, error) {
	u, _ := url.Parse("https://github.com/login/oauth/access_token")
	q := u.Query()
	q.Set("client_id", p.clientID)
	q.Set("client_secret", p.clientSecret)
	q.Set("code", code)
	q.Set("redirect_uri", callbackURL)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return "", nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, nil, fmt.Errorf("Could not exchange token, %d, %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	var token map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", nil, nil, err
	}
	accessToken, _ := token["access_token"].(string)
	if accessToken == "" {
		return "", nil, nil, fmt.Errorf("No access token found")
	}
	providerMetadata := map[string]any{"access_token": accessToken, "refresh_token": token["refresh_token"]}
	providerMetadata["access_token_expires_at"] = expiresAt(token["expires_in"])
	providerMetadata["refresh_token_expires_at"] = expiresAt(token["refresh_token_expires_in"])
	providerJSON, err := json.Marshal(providerMetadata)
	if err != nil {
		return "", nil, nil, err
	}
	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", nil, nil, err
	}
	userReq.Header.Set("Accept", "application/json")
	userReq.Header.Set("Authorization", "Bearer "+accessToken)
	userResp, err := p.client.Do(userReq)
	if err != nil {
		return "", nil, nil, err
	}
	defer userResp.Body.Close()
	var user struct {
		ID        int64  `json:"id"`
		URL       string `json:"url"`
		AvatarURL string `json:"avatar_url"`
		Email     any    `json:"email"`
		Name      any    `json:"name"`
		Company   any    `json:"company"`
		TwoFactor *bool  `json:"two_factor_authentication"`
	}
	if err = json.NewDecoder(userResp.Body).Decode(&user); err != nil {
		return "", nil, nil, err
	}
	if user.ID == 0 {
		return "", nil, nil, fmt.Errorf("Github user response did not contain an id")
	}
	twoFactor := false
	if user.TwoFactor != nil {
		twoFactor = *user.TwoFactor
	}
	userJSON, err := json.Marshal(map[string]any{"profile_url": user.URL, "avatar": user.AvatarURL, "email": user.Email, "name": user.Name, "company": user.Company, "two_factor_authentication": twoFactor})
	if err != nil {
		return "", nil, nil, err
	}
	return strconv.FormatInt(user.ID, 10), providerJSON, userJSON, nil
}

func expiresAt(value any) any {
	seconds, ok := value.(float64)
	if !ok {
		return nil
	}
	return time.Now().UTC().Add(time.Duration(seconds * float64(time.Second))).Format(time.RFC3339Nano)
}

var _ domain.OAuthProvider = (*Provider)(nil)
