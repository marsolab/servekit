package kinde

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/marsolab/servekit/errkit"
)

const expiryBuffer = 30 * time.Second

// M2MClient handles the client credentials OAuth flow for machine-to-machine
// authentication. It caches the token and automatically re-requests when expired.
type M2MClient struct {
	mu       sync.RWMutex
	client   *Client
	audience string
	token    *m2mToken
}

type m2mToken struct {
	accessToken string
	expiresAt   time.Time
}

// M2MOption configures the M2M client.
type M2MOption func(*M2MClient)

// WithM2MAudience sets the audience for the M2M token request.
func WithM2MAudience(aud string) M2MOption {
	return func(m *M2MClient) {
		m.audience = aud
	}
}

// NewM2MClient creates a new M2M client that uses the parent Client's
// credentials and discovered token endpoint.
func (c *Client) NewM2MClient(opts ...M2MOption) *M2MClient {
	m := &M2MClient{
		client:   c,
		audience: c.domain + "/api",
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Token returns a valid access token, requesting a new one if the cached
// token is expired or absent. Thread-safe.
func (m *M2MClient) Token(ctx context.Context) (string, error) {
	m.mu.RLock()

	if m.token != nil && time.Now().Before(m.token.expiresAt) {
		t := m.token.accessToken
		m.mu.RUnlock()

		return t, nil
	}

	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if m.token != nil && time.Now().Before(m.token.expiresAt) {
		return m.token.accessToken, nil
	}

	if err := m.tokenRequest(ctx); err != nil {
		return "", err
	}

	return m.token.accessToken, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"` //nolint:tagliatelle // OAuth2 standard field.
	TokenType   string `json:"token_type"`   //nolint:tagliatelle // OAuth2 standard field.
	ExpiresIn   int    `json:"expires_in"`   //nolint:tagliatelle // OAuth2 standard field.
}

func (m *M2MClient) tokenRequest(ctx context.Context) error {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {m.client.clientID},
		"client_secret": {m.client.clientSecret},
		"audience":      {m.audience},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		m.client.oidc.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("kinde m2m: create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kinde m2m: token request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("kinde m2m: token request: %w: status %d",
			errkit.ErrUnauthenticated, resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("kinde m2m: decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return errors.New("kinde m2m: token response missing access_token")
	}

	if tokenResp.ExpiresIn <= 0 {
		return errors.New("kinde m2m: token response missing or invalid expires_in")
	}

	lifetime := time.Duration(tokenResp.ExpiresIn) * time.Second
	buffer := min(expiryBuffer, lifetime/2)

	m.token = &m2mToken{
		accessToken: tokenResp.AccessToken,
		expiresAt:   time.Now().Add(lifetime - buffer),
	}

	return nil
}
