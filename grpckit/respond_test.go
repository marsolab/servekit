package grpckit

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/marsolab/servekit/ctxkit"
	"github.com/marsolab/servekit/errkit"
	"github.com/maxatome/go-testdeep/td"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewResponseOptions_Defaults(t *testing.T) {
	// NewResponseOptions with no options should return defaults.
	opts := NewResponseOptions()

	td.Cmp(t, opts.statusCode, codes.Unknown, "default status code should be Unknown")
	td.Cmp(t, opts.reportError, false, "default reportError should be false")
}

func TestWithStatus(t *testing.T) {
	// WithStatus should set the status code.
	opts := NewResponseOptions(WithStatus(codes.NotFound))

	td.Cmp(t, opts.statusCode, codes.NotFound)
}

func TestNewResponseOptions_MultipleOptions(t *testing.T) {
	// Multiple options should all be applied.
	opts := NewResponseOptions(
		WithStatus(codes.InvalidArgument),
	)

	td.Cmp(t, opts.statusCode, codes.InvalidArgument)
}

func TestZero(t *testing.T) {
	// zero should return the zero value for any type.
	td.Cmp(t, zero[int](), 0)
	td.Cmp(t, zero[string](), "")
	td.Cmp(t, zero[bool](), false)
	td.CmpNil(t, zero[*int]())
	td.CmpNil(t, zero[error]())
}

func TestZero_Struct(t *testing.T) {
	// zero should return a zeroed struct.
	type testStruct struct {
		Name  string
		Value int
	}

	got := zero[testStruct]()

	td.Cmp(t, got.Name, "")
	td.Cmp(t, got.Value, 0)
}

func TestErrorGRPC_ErrorMapping(t *testing.T) {
	// ErrorGRPC should map errkit errors to the correct gRPC status codes.
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "ErrAlreadyExists maps to AlreadyExists",
			err:      errkit.ErrAlreadyExists,
			wantCode: codes.AlreadyExists,
		},
		{
			name:     "ErrNotFound maps to NotFound",
			err:      errkit.ErrNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "ErrUnauthenticated maps to Unauthenticated",
			err:      errkit.ErrUnauthenticated,
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "ErrUnauthorized maps to PermissionDenied",
			err:      errkit.ErrUnauthorized,
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "ErrInvalidArgument maps to InvalidArgument",
			err:      errkit.ErrInvalidArgument,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrUnavailable maps to Unavailable",
			err:      errkit.ErrUnavailable,
			wantCode: codes.Unavailable,
		},
		{
			name:     "unknown error maps to Internal",
			err:      errors.New("something went wrong"),
			wantCode: codes.Internal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ErrorGRPC[string](context.Background(), tc.err)

			td.Cmp(t, result, "", "returned value should be zero value")
			td.CmpError(t, err)

			s, ok := status.FromError(err)
			td.CmpTrue(t, ok, "error should be a gRPC status error")
			td.Cmp(t, s.Code(), tc.wantCode)
		})
	}
}

func TestErrorGRPC_WrappedErrors(t *testing.T) {
	// ErrorGRPC should correctly map wrapped errkit errors.
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "wrapped ErrNotFound",
			err:      fmt.Errorf("user lookup: %w", errkit.ErrNotFound),
			wantCode: codes.NotFound,
		},
		{
			name:     "wrapped ErrAlreadyExists",
			err:      fmt.Errorf("create user: %w", errkit.ErrAlreadyExists),
			wantCode: codes.AlreadyExists,
		},
		{
			name:     "wrapped ErrUnauthenticated",
			err:      fmt.Errorf("auth check: %w", errkit.ErrUnauthenticated),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "double wrapped ErrInvalidArgument",
			err:      fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", errkit.ErrInvalidArgument)),
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ErrorGRPC[int](context.Background(), tc.err)

			td.Cmp(t, result, 0)
			td.CmpError(t, err)

			s, ok := status.FromError(err)
			td.CmpTrue(t, ok)
			td.Cmp(t, s.Code(), tc.wantCode)
		})
	}
}

func TestErrorGRPC_LogErrHook(t *testing.T) {
	// ErrorGRPC should call the log error hook if one is set on the context.
	var hookedErr error

	ctx := ctxkit.SetLogErrHook(context.Background(), func(err error) {
		hookedErr = err
	})

	testErr := errkit.ErrNotFound

	_, _ = ErrorGRPC[string](ctx, testErr)

	td.CmpNotNil(t, hookedErr, "hook should have been called")
	td.CmpTrue(t, errors.Is(hookedErr, testErr), "hooked error should match the original error")
}

func TestErrorGRPC_NoLogErrHook(t *testing.T) {
	// ErrorGRPC should not panic when no log error hook is set.
	_, err := ErrorGRPC[string](context.Background(), errors.New("test"))

	td.CmpError(t, err, "should still return an error")
}

func TestErrorGRPC_ZeroValueTypes(t *testing.T) {
	// ErrorGRPC should return zero values for different generic types.
	t.Run("int type", func(t *testing.T) {
		val, err := ErrorGRPC[int](context.Background(), errkit.ErrNotFound)
		td.Cmp(t, val, 0)
		td.CmpError(t, err)
	})

	t.Run("bool type", func(t *testing.T) {
		val, err := ErrorGRPC[bool](context.Background(), errkit.ErrNotFound)
		td.Cmp(t, val, false)
		td.CmpError(t, err)
	})

	t.Run("struct type", func(t *testing.T) {
		type resp struct {
			ID   string
			Name string
		}

		val, err := ErrorGRPC[resp](context.Background(), errkit.ErrNotFound)
		td.Cmp(t, val, resp{})
		td.CmpError(t, err)
	})

	t.Run("pointer type", func(t *testing.T) {
		val, err := ErrorGRPC[*string](context.Background(), errkit.ErrNotFound)
		td.CmpNil(t, val)
		td.CmpError(t, err)
	})
}

func TestErrorGRPC_StatusMessage(t *testing.T) {
	// ErrorGRPC should set the status message to the code string.
	tests := []struct {
		name        string
		err         error
		wantMessage string
	}{
		{
			name:        "NotFound message",
			err:         errkit.ErrNotFound,
			wantMessage: codes.NotFound.String(),
		},
		{
			name:        "Internal message for unknown error",
			err:         errors.New("unknown"),
			wantMessage: codes.Internal.String(),
		},
		{
			name:        "AlreadyExists message",
			err:         errkit.ErrAlreadyExists,
			wantMessage: codes.AlreadyExists.String(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ErrorGRPC[string](context.Background(), tc.err)

			s, ok := status.FromError(err)
			td.CmpTrue(t, ok)
			td.Cmp(t, s.Message(), tc.wantMessage)
		})
	}
}
