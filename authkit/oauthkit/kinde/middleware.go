package kinde

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/marsolab/servekit/ctxkit"
	"github.com/marsolab/servekit/errkit"
	"github.com/marsolab/servekit/httpkit"
)

const (
	// accessTokenClaimsKey is the context key for AccessTokenClaims.
	accessTokenClaimsKey ctxkit.Key = "kinde.access_token_claims"

	// subjectKey is the context key for the token subject (user ID).
	subjectKey ctxkit.Key = "kinde.subject"

	bearerPrefix = "Bearer "
)

// AuthMiddleware returns an HTTP middleware that verifies Kinde access tokens.
// It extracts the Bearer token from the Authorization header, verifies it
// using the JWKS provider, parses claims into AccessTokenClaims, and stores
// them in the request context. On failure it responds with 401.
func (c *Client) AuthMiddleware() httpkit.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				httpkit.ErrorHTTP(w, r, errkit.ErrUnauthenticated)

				return
			}

			var claims AccessTokenClaims

			//nolint:contextcheck // JWKSProvider API does not accept context.
			if err := c.jwks.ParseVerifyClaims(token, &claims); err != nil {
				httpkit.ErrorHTTP(w, r, errkit.ErrUnauthenticated)

				return
			}

			if !c.validateIssuerAudience(&claims) {
				httpkit.ErrorHTTP(w, r, errkit.ErrUnauthenticated)

				return
			}

			ctx := r.Context()
			ctx = ctxkit.Set(ctx, accessTokenClaimsKey, &claims)
			ctx = ctxkit.Set(ctx, subjectKey, claims.Subject)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// PermissionMiddleware returns an HTTP middleware that checks whether the
// already-parsed AccessTokenClaims (from AuthMiddleware) contain the required
// permissions. On failure it responds with 403. This middleware must be placed
// after AuthMiddleware in the middleware chain.
func PermissionMiddleware(perms ...string) httpkit.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetAccessTokenClaims(r.Context())
			if claims == nil {
				httpkit.ErrorHTTP(w, r, errkit.ErrUnauthenticated)

				return
			}

			for _, perm := range perms {
				if !claims.HasPermission(perm) {
					httpkit.ErrorHTTP(w, r, errkit.ErrUnauthorized)

					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAccessTokenClaims retrieves the AccessTokenClaims from the request context.
// Returns nil if the AuthMiddleware has not run.
func GetAccessTokenClaims(ctx context.Context) *AccessTokenClaims {
	return ctxkit.Get[*AccessTokenClaims](ctx, accessTokenClaimsKey)
}

// GetSubject retrieves the authenticated user's subject (Kinde user ID) from context.
// Returns an empty string if the AuthMiddleware has not run.
func GetSubject(ctx context.Context) string {
	return ctxkit.Get[string](ctx, subjectKey)
}

// validateIssuerAudience checks that the token's issuer matches the discovered
// OIDC issuer and, if an audience is configured, that the token includes it.
func (c *Client) validateIssuerAudience(claims *AccessTokenClaims) bool {
	if claims.Issuer != c.oidc.Issuer {
		return false
	}

	if c.audience != "" && !slices.Contains(claims.Audience, c.audience) {
		return false
	}

	return true
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, bearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(auth, bearerPrefix)
}
