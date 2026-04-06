package kinde

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/marsolab/servekit/authkit/jwtkit"
)

const (
	defaultJWKSRefreshInterval = 1 * time.Hour
	defaultHTTPClientTimeout   = 10 * time.Second
	oidcDiscoveryPath          = "/.well-known/openid-configuration"
)

// OIDCConfig holds the discovered OpenID Connect configuration.
type OIDCConfig struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"` //nolint:tagliatelle // OIDC standard field.
	TokenEndpoint         string `json:"token_endpoint"`         //nolint:tagliatelle // OIDC standard field.
	UserinfoEndpoint      string `json:"userinfo_endpoint"`      //nolint:tagliatelle // OIDC standard field.
	JWKSURI               string `json:"jwks_uri"`               //nolint:tagliatelle // OIDC standard field.
}

// Client is the central Kinde integration point. It handles OIDC discovery
// and provides access to token verification, M2M authentication, and the
// management API.
type Client struct {
	domain         string
	clientID       string
	clientSecret   string
	audience       string
	httpClient     *http.Client
	jwksRefreshInt time.Duration
	oidc           *OIDCConfig
	jwks           *jwtkit.JWKSProvider
}

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client for all outbound requests.
func WithHTTPClient(c *http.Client) Option {
	return func(o *Client) {
		o.httpClient = c
	}
}

// WithAudience sets the expected audience for token verification.
func WithAudience(aud string) Option {
	return func(o *Client) {
		o.audience = aud
	}
}

// WithJWKSRefreshInterval sets the interval for refreshing the JWKS key cache.
func WithJWKSRefreshInterval(d time.Duration) Option {
	return func(o *Client) {
		o.jwksRefreshInt = d
	}
}

// NewClient creates a new Kinde client. It performs OIDC discovery and
// initializes the JWKS provider for token verification. The domain should
// be the full Kinde tenant URL (e.g., "https://myapp.kinde.com").
func NewClient(domain, clientID, clientSecret string, opts ...Option) (*Client, error) {
	c := &Client{
		domain:         strings.TrimRight(domain, "/"),
		clientID:       clientID,
		clientSecret:   clientSecret,
		jwksRefreshInt: defaultJWKSRefreshInterval,
		httpClient:     &http.Client{Timeout: defaultHTTPClientTimeout},
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := c.discoverOIDC(context.Background()); err != nil {
		return nil, fmt.Errorf("kinde: oidc discovery: %w", err)
	}

	jwks, err := jwtkit.NewJWKSProvider(c.oidc.JWKSURI, c.jwksRefreshInt)
	if err != nil {
		return nil, fmt.Errorf("kinde: jwks provider: %w", err)
	}

	c.jwks = jwks

	return c, nil
}

// TokenVerifier returns the underlying JWKSProvider for direct token verification.
func (c *Client) TokenVerifier() *jwtkit.JWKSProvider {
	return c.jwks
}

// GetOIDCConfig returns the discovered OIDC configuration.
func (c *Client) GetOIDCConfig() OIDCConfig {
	return *c.oidc
}

func (c *Client) discoverOIDC(ctx context.Context) error {
	url := c.domain + oidcDiscoveryPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("create discovery request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch discovery document: %w", err)
	}

	if resp == nil {
		return errors.New("fetch discovery document: empty response")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch discovery document: unexpected status code: %d", resp.StatusCode)
	}

	var oidc OIDCConfig
	if err := json.NewDecoder(resp.Body).Decode(&oidc); err != nil {
		return fmt.Errorf("decode discovery document: %w", err)
	}

	if oidc.Issuer == "" {
		return errors.New("discovery document missing issuer")
	}

	if oidc.JWKSURI == "" {
		return errors.New("discovery document missing jwks_uri")
	}

	c.oidc = &oidc

	return nil
}
