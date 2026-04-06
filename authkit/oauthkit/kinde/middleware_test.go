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

	"github.com/cristalhq/jwt/v5"
	"github.com/marsolab/servekit/authkit/jwtkit"
	"github.com/marsolab/servekit/ctxkit"
	"github.com/maxatome/go-testdeep/td"
)

// middlewareTestEnv sets up a full Kinde client with mock OIDC + JWKS servers
// and a JWT signer for generating test tokens.
type middlewareTestEnv struct {
	client  *Client
	signer  jwt.Signer
	builder *jwt.Builder
	kid     string
}

func newMiddlewareTestEnv(t *testing.T) *middlewareTestEnv {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	kid := "mw-test-kid"
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
		Issuer:  "https://test.kinde.com",
		JWKSURI: jwksServer.URL,
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

	client, err := NewClient(oidcServer.URL, "client-id", "client-secret")
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	signer, err := jwt.NewSignerRS(jwt.RS256, privateKey)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	builder := jwt.NewBuilder(signer, jwt.WithKeyID(kid))

	return &middlewareTestEnv{
		client:  client,
		signer:  signer,
		builder: builder,
		kid:     kid,
	}
}

func (e *middlewareTestEnv) buildToken(t *testing.T, claims any) string {
	t.Helper()

	token, err := e.builder.Build(claims)
	if err != nil {
		t.Fatalf("build token: %v", err)
	}

	return token.String()
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	env := newMiddlewareTestEnv(t)

	claims := &AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://test.kinde.com",
			Subject:   "kp_user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		OrgCode:     "org_abc",
		Permissions: []string{"read:data", "write:data"},
	}
	token := env.buildToken(t, claims)

	var capturedClaims *AccessTokenClaims
	var capturedSubject string

	handler := env.client.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClaims = GetAccessTokenClaims(r.Context())
		capturedSubject = GetSubject(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusOK)
	td.CmpNotNil(t, capturedClaims)
	td.Cmp(t, capturedSubject, "kp_user123")
	td.Cmp(t, capturedClaims.OrgCode, "org_abc")
	td.Cmp(t, capturedClaims.HasPermission("read:data"), true)
	td.Cmp(t, capturedClaims.HasPermission("delete:data"), false)
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	env := newMiddlewareTestEnv(t)

	handler := env.client.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusUnauthorized)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	env := newMiddlewareTestEnv(t)

	handler := env.client.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusUnauthorized)
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	env := newMiddlewareTestEnv(t)

	claims := &AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "kp_user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := env.buildToken(t, claims)

	handler := env.client.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusUnauthorized)
}

func TestPermissionMiddleware_HasPermission(t *testing.T) {
	claims := &AccessTokenClaims{
		Permissions: []string{"read:data", "write:data"},
	}

	handler := PermissionMiddleware("read:data")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = ctxkit.Set(ctx, accessTokenClaimsKey, claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusOK)
}

func TestPermissionMiddleware_MissingPermission(t *testing.T) {
	claims := &AccessTokenClaims{
		Permissions: []string{"read:data"},
	}

	handler := PermissionMiddleware("delete:data")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = ctxkit.Set(ctx, accessTokenClaimsKey, claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusForbidden)
}

func TestPermissionMiddleware_NoClaims(t *testing.T) {
	handler := PermissionMiddleware("read:data")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusUnauthorized)
}

func TestGetAccessTokenClaims_NilWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	claims := GetAccessTokenClaims(req.Context())
	td.CmpNil(t, claims)
}

func TestGetSubject_EmptyWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	subject := GetSubject(req.Context())
	td.Cmp(t, subject, "")
}

func TestAuthMiddleware_WrongIssuer(t *testing.T) {
	env := newMiddlewareTestEnv(t)

	claims := &AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://wrong-issuer.kinde.com",
			Subject:   "kp_user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := env.buildToken(t, claims)

	handler := env.client.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusUnauthorized)
}

func TestAuthMiddleware_WrongAudience(t *testing.T) {
	env := newMiddlewareTestEnv(t)
	env.client.audience = "https://expected-api.example.com"

	claims := &AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://test.kinde.com",
			Subject:   "kp_user123",
			Audience:  jwt.Audience{"https://other-api.example.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := env.buildToken(t, claims)

	handler := env.client.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusUnauthorized)
}

func TestAuthMiddleware_CorrectAudience(t *testing.T) {
	env := newMiddlewareTestEnv(t)
	env.client.audience = "https://my-api.example.com"

	claims := &AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://test.kinde.com",
			Subject:   "kp_user123",
			Audience:  jwt.Audience{"https://my-api.example.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := env.buildToken(t, claims)

	handler := env.client.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusOK)
}

func TestAuthMiddleware_NoAudienceConfigured(t *testing.T) {
	env := newMiddlewareTestEnv(t)
	// No audience set on client — should accept any audience.

	claims := &AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://test.kinde.com",
			Subject:   "kp_user123",
			Audience:  jwt.Audience{"https://any-api.example.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := env.buildToken(t, claims)

	handler := env.client.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	td.Cmp(t, rec.Code, http.StatusOK)
}
