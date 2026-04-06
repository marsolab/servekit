package jwtkit_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cristalhq/jwt/v5"
	"github.com/marsolab/servekit/authkit/jwtkit"
	"github.com/marsolab/servekit/errkit"
	"github.com/marsolab/servekit/idkit"
	"github.com/maxatome/go-testdeep/td"
)

func TestJWKSProvider(t *testing.T) {
	td.NewT(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	td.CmpNil(t, err)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	if privateKey == nil {
		t.Fatal("generate rsa key: nil key")
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		td.CmpNil(t, json.NewEncoder(w).Encode(keyStore))
	}))
	t.Cleanup(server.Close)

	jwksProvider, err := jwtkit.NewJWKSProvider(server.URL, 1*time.Minute)
	td.CmpNil(t, err)
	if err != nil {
		t.Fatalf("create jwks provider: %v", err)
	}

	if jwksProvider == nil {
		t.Fatal("create jwks provider: nil provider")
	}

	signer, err := jwt.NewSignerRS(jwt.RS256, privateKey)
	td.CmpNil(t, err)

	builder := jwt.NewBuilder(signer, jwt.WithKeyID(kid))
	claims := &jwt.RegisteredClaims{
		ID:        idkit.XID(),
		Subject:   "test-subject",
		Audience:  []string{"test-audience"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		NotBefore: jwt.NewNumericDate(time.Now()),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token, err := builder.Build(claims)
	td.CmpNil(t, err)

	parsedToken, err := jwksProvider.ParseVerify(token.String())
	td.CmpNil(t, err)
	if err != nil {
		t.Fatalf("parse verify token: %v", err)
	}

	if parsedToken == nil {
		t.Fatal("parse verify token: nil token")
	}

	td.Cmp(t, parsedToken.Subject, "test-subject")

	t.Run("Sign returns error", func(t *testing.T) {
		_, err := jwksProvider.Sign(&jwtkit.Token{})
		td.CmpNotNil(t, err)
		td.CmpContains(t, err.Error(), "signing is not supported")
	})

	t.Run("Verify with valid token", func(t *testing.T) {
		err := jwksProvider.Verify(token.String())
		td.CmpNil(t, err)
	})

	t.Run("Verify with malformed token", func(t *testing.T) {
		err := jwksProvider.Verify("not.a.valid.token")
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenInvalid))
	})

	t.Run("Verify with expired token", func(t *testing.T) {
		expiredClaims := &jwt.RegisteredClaims{
			ID:        idkit.XID(),
			Subject:   "test-subject",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		}
		expiredToken, buildErr := builder.Build(expiredClaims)
		td.CmpNil(t, buildErr)

		err := jwksProvider.Verify(expiredToken.String())
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenExpired))
	})

	t.Run("ParseVerifyClaims with valid token", func(t *testing.T) {
		type CustomClaims struct {
			jwt.RegisteredClaims
			Role string `json:"role"`
		}

		customClaims := &CustomClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        idkit.XID(),
				Subject:   "custom-subject",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				NotBefore: jwt.NewNumericDate(time.Now()),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			Role: "admin",
		}
		customToken, buildErr := builder.Build(customClaims)
		td.CmpNil(t, buildErr)

		parsed := CustomClaims{}
		err := jwksProvider.ParseVerifyClaims(customToken.String(), &parsed)
		td.CmpNil(t, err)
		td.Cmp(t, parsed.Role, "admin")
		td.Cmp(t, parsed.Subject, "custom-subject")
	})

	t.Run("ParseVerifyClaims with malformed token", func(t *testing.T) {
		type CustomClaims struct {
			jwt.RegisteredClaims
		}

		parsed := CustomClaims{}
		err := jwksProvider.ParseVerifyClaims("invalid-token", &parsed)
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenInvalid))
	})

	t.Run("ParseVerifyClaims with expired token", func(t *testing.T) {
		type CustomClaims struct {
			jwt.RegisteredClaims
		}

		expiredClaims := &CustomClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        idkit.XID(),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		expiredToken, buildErr := builder.Build(expiredClaims)
		td.CmpNil(t, buildErr)

		parsed := CustomClaims{}
		err := jwksProvider.ParseVerifyClaims(expiredToken.String(), &parsed)
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenExpired))
	})

	t.Run("ParseVerify with missing kid", func(t *testing.T) {
		// Build a token without a kid header.
		noKidSigner, signerErr := jwt.NewSignerRS(jwt.RS256, privateKey)
		td.CmpNil(t, signerErr)

		noKidBuilder := jwt.NewBuilder(noKidSigner)
		noKidClaims := &jwt.RegisteredClaims{
			ID:        idkit.XID(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		}
		noKidToken, buildErr := noKidBuilder.Build(noKidClaims)
		td.CmpNil(t, buildErr)

		_, err := jwksProvider.ParseVerify(noKidToken.String())
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenInvalid))
	})

	t.Run("ParseVerify with unknown kid", func(t *testing.T) {
		// Build a token with an unknown kid.
		unknownKidBuilder := jwt.NewBuilder(signer, jwt.WithKeyID("unknown-kid"))
		unknownKidClaims := &jwt.RegisteredClaims{
			ID:        idkit.XID(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		}
		unknownKidToken, buildErr := unknownKidBuilder.Build(unknownKidClaims)
		td.CmpNil(t, buildErr)

		_, err := jwksProvider.ParseVerify(unknownKidToken.String())
		td.Cmp(t, err, td.ErrorIs(errkit.ErrTokenInvalid))
	})
}

func ExampleNewJWKSProvider() {
	// This is a placeholder for a real JWKS endpoint
	// In a real application, you would use a URL like
	// "https://www.googleapis.com/oauth2/v3/certs"
	// or your own identity provider's JWKS endpoint.
	jwksURL := "http://127.0.0.1:8080/.well-known/jwks.json"

	// Create a new JWKSProvider with a 1-hour refresh interval.
	// The provider will fetch the keys from the URL upon creation
	// and then periodically refresh them.
	provider, err := jwtkit.NewJWKSProvider(jwksURL, 1*time.Hour)
	if err != nil {
		// In a real app, you would likely log this error and exit,
		// as the application cannot verify tokens without the keys.
		fmt.Printf("failed to create JWKS provider: %v", err)
		return
	}

	// The provider can now be used to verify tokens.
	// Typically, you would use this in a middleware to protect your routes.
	// For example:
	// token := "a.jwt.token"
	// parsedToken, err := provider.ParseVerify(token)
	_ = provider
}
