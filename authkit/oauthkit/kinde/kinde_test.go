package kinde

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/marsolab/servekit/authkit/jwtkit"
	"github.com/maxatome/go-testdeep/td"
)

// testEnv holds shared test infrastructure for OIDC + JWKS mock servers.
type testEnv struct {
	privateKey *rsa.PrivateKey
	kid        string
	oidcServer *httptest.Server
	jwksServer *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	kid := "test-kid"
	keyStore := jwtkit.KeyStore{
		Keys: []jwtkit.Key{
			{
				Use: "sig",
				Kty: "RSA",
				Kid: kid,
				Alg: "RS256",
				N:   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
			},
		},
	}

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keyStore) //nolint:errcheck // Test helper.
	}))
	t.Cleanup(jwksServer.Close)

	oidcConfig := OIDCConfig{
		Issuer:                "https://test.kinde.com",
		AuthorizationEndpoint: "https://test.kinde.com/oauth2/auth",
		TokenEndpoint:         "https://test.kinde.com/oauth2/token",
		UserinfoEndpoint:      "https://test.kinde.com/oauth2/userinfo",
		JWKSURI:               jwksServer.URL,
	}

	oidcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != oidcDiscoveryPath {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(oidcConfig) //nolint:errcheck // Test helper.
	}))
	t.Cleanup(oidcServer.Close)

	return &testEnv{
		privateKey: privateKey,
		kid:        kid,
		oidcServer: oidcServer,
		jwksServer: jwksServer,
	}
}

func TestNewClient(t *testing.T) {
	env := newTestEnv(t)

	client, err := NewClient(env.oidcServer.URL, "client-id", "client-secret",
		WithJWKSRefreshInterval(1*time.Minute),
	)
	td.CmpNil(t, err)

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	oidc := client.GetOIDCConfig()
	td.Cmp(t, oidc.Issuer, "https://test.kinde.com")
	td.Cmp(t, oidc.JWKSURI, env.jwksServer.URL)
	td.CmpNotNil(t, client.TokenVerifier())
}

func TestNewClient_DiscoveryFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	_, err := NewClient(server.URL, "client-id", "client-secret")
	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "oidc discovery")
}

func TestNewClient_MissingIssuer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OIDCConfig{JWKSURI: "https://example.com/jwks"}) //nolint:errcheck // Test helper.
	}))
	t.Cleanup(server.Close)

	_, err := NewClient(server.URL, "client-id", "client-secret")
	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "missing issuer")
}

func TestNewClient_MissingJWKSURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OIDCConfig{Issuer: "https://test.kinde.com"}) //nolint:errcheck // Test helper.
	}))
	t.Cleanup(server.Close)

	_, err := NewClient(server.URL, "client-id", "client-secret")
	td.CmpNotNil(t, err)
	td.CmpContains(t, err.Error(), "missing jwks_uri")
}

func TestNewClient_TrailingSlash(t *testing.T) {
	env := newTestEnv(t)

	client, err := NewClient(env.oidcServer.URL+"/", "client-id", "client-secret")
	td.CmpNil(t, err)

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}
