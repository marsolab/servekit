package errkit

import (
	"errors"
	"fmt"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestError_Error(t *testing.T) {
	f := func(name string, err error, want string) {
		t.Helper()

		t.Run(name, func(t *testing.T) {
			td.Cmp(t, err.Error(), want)
		})
	}

	f("ErrAlreadyExist", ErrAlreadyExists, "already exists")
	f("ErrInvalidArgument", ErrInvalidArgument, "invalid argument")
	f("ErrUnauthenticated", ErrUnauthenticated, "authentication failed")
	f("ErrUnauthorized", ErrUnauthorized, "permission denied")
	f("ErrUnavailable", ErrUnavailable, "temporarily unavailable")
	f("ErrConnFailed", ErrConnFailed, "connection failed")
	f("ErrNotFound", ErrNotFound, "not found")
	f("ErrPasswordIncorrect", ErrPasswordIncorrect, "password incorrect")
	f("ErrTokenInvalid", ErrTokenInvalid, "invalid token")
	f("ErrTokenExpired", ErrTokenExpired, "expired token")
	f("ErrTokenNotBefore", ErrTokenNotBefore, "token is not valid yet")
	f("ErrTokenIssuedAt", ErrTokenIssuedAt, "token is not valid at the current time")
	f("ErrInvalidID", ErrInvalidID, "invalid identifier")
	f("ErrValidation", ErrValidation, "validation failed")
	f("Custom", Error("test error"), "test error")
}

func TestError_Is(t *testing.T) {
	// Test that sentinel errors match themselves via errors.Is.
	sentinels := []struct {
		name string
		err  Error
	}{
		{"ErrNotFound", ErrNotFound},
		{"ErrAlreadyExists", ErrAlreadyExists},
		{"ErrInvalidArgument", ErrInvalidArgument},
		{"ErrUnauthenticated", ErrUnauthenticated},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrUnavailable", ErrUnavailable},
		{"ErrConnFailed", ErrConnFailed},
		{"ErrPasswordIncorrect", ErrPasswordIncorrect},
		{"ErrTokenInvalid", ErrTokenInvalid},
		{"ErrTokenExpired", ErrTokenExpired},
		{"ErrTokenNotBefore", ErrTokenNotBefore},
		{"ErrTokenIssuedAt", ErrTokenIssuedAt},
		{"ErrInvalidID", ErrInvalidID},
		{"ErrValidation", ErrValidation},
	}

	for _, s := range sentinels {
		t.Run(s.name+"/self", func(t *testing.T) {
			td.CmpTrue(t, errors.Is(s.err, s.err))
		})
	}

	// Test that a wrapped sentinel error is still matched by errors.Is.
	t.Run("wrapped", func(t *testing.T) {
		wrapped := fmt.Errorf("something went wrong: %w", ErrNotFound)
		td.CmpTrue(t, errors.Is(wrapped, ErrNotFound))
	})

	// Test that different sentinel errors do not match each other.
	t.Run("different_errors_do_not_match", func(t *testing.T) {
		td.CmpFalse(t, errors.Is(ErrNotFound, ErrAlreadyExists))
		td.CmpFalse(t, errors.Is(ErrUnauthenticated, ErrUnauthorized))
	})

	// Test that a doubly-wrapped sentinel error is still found.
	t.Run("double_wrapped", func(t *testing.T) {
		first := fmt.Errorf("first: %w", ErrTokenExpired)
		second := fmt.Errorf("second: %w", first)
		td.CmpTrue(t, errors.Is(second, ErrTokenExpired))
	})
}

func TestError_As(t *testing.T) {
	// Test that errors.As can extract the Error type from a wrapped error.
	wrapped := fmt.Errorf("context: %w", ErrValidation)

	var target Error

	td.CmpTrue(t, errors.As(wrapped, &target))
	td.Cmp(t, target, ErrValidation)
}

func TestError_ErrorInterface(t *testing.T) {
	// Verify that Error satisfies the error interface.
	var err error = ErrNotFound

	td.Cmp(t, err.Error(), "not found")
}
