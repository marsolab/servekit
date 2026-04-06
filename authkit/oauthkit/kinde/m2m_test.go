package kinde

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maxatome/go-testdeep/td"
)

func TestM2MClient_Token(t *testing.T) {
	var tokenRequestCount atomic.Int32

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequestCount.Add(1)

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		td.Cmp(t, r.FormValue("grant_type"), "client_credentials")
		td.Cmp(t, r.FormValue("client_id"), "test-client-id")
		td.Cmp(t, r.FormValue("client_secret"), "test-client-secret")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "m2m-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	t.Cleanup(tokenServer.Close)

	client := &Client{
		domain:       "https://test.kinde.com",
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   http.DefaultClient,
		oidc: &OIDCConfig{
			TokenEndpoint: tokenServer.URL,
		},
	}

	m2m := client.NewM2MClient()

	// First call should fetch a new token.
	token, err := m2m.Token(context.Background())
	td.CmpNil(t, err)
	td.Cmp(t, token, "m2m-access-token")
	td.Cmp(t, tokenRequestCount.Load(), int32(1))

	// Second call should return cached token.
	token, err = m2m.Token(context.Background())
	td.CmpNil(t, err)
	td.Cmp(t, token, "m2m-access-token")
	td.Cmp(t, tokenRequestCount.Load(), int32(1))
}

func TestM2MClient_Token_ShortLivedNotImmediatelyExpired(t *testing.T) {
	var callCount atomic.Int32

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "short-lived-token",
			TokenType:   "Bearer",
			ExpiresIn:   2, // Short-lived: buffer is capped at half (1s), so token lives ~1s.
		})
	}))
	t.Cleanup(tokenServer.Close)

	client := &Client{
		domain:       "https://test.kinde.com",
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   http.DefaultClient,
		oidc: &OIDCConfig{
			TokenEndpoint: tokenServer.URL,
		},
	}

	m2m := client.NewM2MClient()

	// First call fetches a token.
	token, err := m2m.Token(context.Background())
	td.CmpNil(t, err)
	td.Cmp(t, token, "short-lived-token")

	// Immediate second call should return the cached token, not re-fetch.
	// The buffer is capped at half the lifetime (1s), so the token is still valid.
	_, err = m2m.Token(context.Background())
	td.CmpNil(t, err)
	td.Cmp(t, callCount.Load(), int32(1))
}

func TestM2MClient_Token_Concurrent(t *testing.T) {
	var callCount atomic.Int32

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "concurrent-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	t.Cleanup(tokenServer.Close)

	client := &Client{
		domain:       "https://test.kinde.com",
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   http.DefaultClient,
		oidc: &OIDCConfig{
			TokenEndpoint: tokenServer.URL,
		},
	}

	m2m := client.NewM2MClient()

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			token, err := m2m.Token(context.Background())
			td.CmpNil(t, err)
			td.Cmp(t, token, "concurrent-token")
		})
	}

	wg.Wait()
}

func TestM2MClient_Token_ServerError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(tokenServer.Close)

	client := &Client{
		domain:       "https://test.kinde.com",
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   &http.Client{Timeout: 5 * time.Second},
		oidc: &OIDCConfig{
			TokenEndpoint: tokenServer.URL,
		},
	}

	m2m := client.NewM2MClient()

	_, err := m2m.Token(context.Background())
	td.CmpNotNil(t, err)
}

func TestM2MClient_Token_EmptyAccessToken(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	t.Cleanup(tokenServer.Close)

	client := &Client{
		domain:       "https://test.kinde.com",
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   http.DefaultClient,
		oidc: &OIDCConfig{
			TokenEndpoint: tokenServer.URL,
		},
	}

	m2m := client.NewM2MClient()

	_, err := m2m.Token(context.Background())
	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "missing access_token")
}

func TestM2MClient_Token_ZeroExpiresIn(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{ //nolint:errcheck // Test helper.
			AccessToken: "some-token",
			TokenType:   "Bearer",
			ExpiresIn:   0,
		})
	}))
	t.Cleanup(tokenServer.Close)

	client := &Client{
		domain:       "https://test.kinde.com",
		clientID:     "test-client-id",
		clientSecret: "test-client-secret",
		httpClient:   http.DefaultClient,
		oidc: &OIDCConfig{
			TokenEndpoint: tokenServer.URL,
		},
	}

	m2m := client.NewM2MClient()

	_, err := m2m.Token(context.Background())
	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "invalid expires_in")
}

func TestM2MClient_WithM2MAudience(t *testing.T) {
	client := &Client{
		domain: "https://test.kinde.com",
		oidc:   &OIDCConfig{},
	}

	m2m := client.NewM2MClient(WithM2MAudience("https://custom-api.example.com"))
	td.Cmp(t, m2m.audience, "https://custom-api.example.com")
}
