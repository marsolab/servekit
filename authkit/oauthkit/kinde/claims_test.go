package kinde

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestAccessTokenClaims_HasPermission(t *testing.T) {
	claims := &AccessTokenClaims{
		Permissions: []string{"read:users", "write:users", "admin"},
	}

	td.Cmp(t, claims.HasPermission("read:users"), true)
	td.Cmp(t, claims.HasPermission("admin"), true)
	td.Cmp(t, claims.HasPermission("delete:users"), false)
	td.Cmp(t, claims.HasPermission(""), false)

	emptyClaims := &AccessTokenClaims{}
	td.Cmp(t, emptyClaims.HasPermission("read:users"), false)
}
