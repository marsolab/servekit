package kinde

import (
	"slices"

	"github.com/cristalhq/jwt/v5"
)

// AccessTokenClaims represents the claims in a Kinde access token.
type AccessTokenClaims struct {
	jwt.RegisteredClaims
	OrgCode      string          `json:"org_code,omitempty"` //nolint:tagliatelle // Kinde API uses snake_case.
	Permissions  []string        `json:"permissions,omitempty"`
	FeatureFlags map[string]Flag `json:"feature_flags,omitempty"` //nolint:tagliatelle // Kinde API uses snake_case.
}

// Flag represents a Kinde feature flag value.
type Flag struct {
	Type  string `json:"t"`
	Value any    `json:"v"`
}

// HasPermission checks whether the access token includes the given permission.
func (c *AccessTokenClaims) HasPermission(perm string) bool {
	return slices.Contains(c.Permissions, perm)
}

// IDTokenClaims represents the claims in a Kinde ID token.
type IDTokenClaims struct {
	jwt.RegisteredClaims
	Email         string   `json:"email,omitempty"`
	EmailVerified bool     `json:"email_verified,omitempty"` //nolint:tagliatelle // OIDC standard claim.
	Name          string   `json:"name,omitempty"`
	GivenName     string   `json:"given_name,omitempty"`  //nolint:tagliatelle // OIDC standard claim.
	FamilyName    string   `json:"family_name,omitempty"` //nolint:tagliatelle // OIDC standard claim.
	Picture       string   `json:"picture,omitempty"`
	OrgCodes      []string `json:"org_codes,omitempty"` //nolint:tagliatelle // Kinde API uses snake_case.
}
