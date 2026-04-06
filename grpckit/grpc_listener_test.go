package grpckit

import (
	"context"
	"log/slog"
	"testing"

	"github.com/maxatome/go-testdeep/td"
	"google.golang.org/grpc"
)

func TestApplyOptionsGRPC_Defaults(t *testing.T) {
	// Applying no options should return a config with default values.
	cfg := applyOptionsGRPC()

	td.CmpNotNil(t, cfg.logger, "default logger should not be nil")
	td.Cmp(t, len(cfg.unaryInterceptors), 0, "default unary interceptors should be empty")
	td.Cmp(t, len(cfg.streamInterceptors), 0, "default stream interceptors should be empty")
}

func TestWithLogger_grpc(t *testing.T) {
	// WithLogger should override the default logger.
	customLogger := slog.Default()
	cfg := applyOptionsGRPC(WithLogger(customLogger))

	td.Cmp(t, cfg.logger, customLogger, "logger should be the custom one")
}

func TestWithLogger_grpc_Nil(t *testing.T) {
	// Passing nil to WithLogger should keep the default logger.
	cfg := applyOptionsGRPC(WithLogger(nil))

	td.CmpNotNil(t, cfg.logger, "nil logger should not override the default")
}

func TestWithUnaryInterceptors(t *testing.T) {
	// WithUnaryInterceptors should add interceptors to the config.
	interceptor := func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (any, error) {
		return nil, nil
	}

	cfg := applyOptionsGRPC(WithUnaryInterceptors(interceptor))

	td.Cmp(t, len(cfg.unaryInterceptors), 1, "should have one unary interceptor")
}

func TestWithUnaryInterceptors_Multiple(t *testing.T) {
	// Multiple calls to WithUnaryInterceptors should append interceptors.
	interceptor := func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (any, error) {
		return nil, nil
	}

	cfg := applyOptionsGRPC(
		WithUnaryInterceptors(interceptor),
		WithUnaryInterceptors(interceptor, interceptor),
	)

	td.Cmp(t, len(cfg.unaryInterceptors), 3, "should have three unary interceptors")
}

func TestWithStreamInterceptors(t *testing.T) {
	// WithStreamInterceptors should add interceptors to the config.
	interceptor := func(
		srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
	) error {
		return nil
	}

	cfg := applyOptionsGRPC(WithStreamInterceptors(interceptor))

	td.Cmp(t, len(cfg.streamInterceptors), 1, "should have one stream interceptor")
}

func TestWithStreamInterceptors_Multiple(t *testing.T) {
	// Multiple calls to WithStreamInterceptors should append interceptors.
	interceptor := func(
		srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
	) error {
		return nil
	}

	cfg := applyOptionsGRPC(
		WithStreamInterceptors(interceptor),
		WithStreamInterceptors(interceptor, interceptor),
	)

	td.Cmp(t, len(cfg.streamInterceptors), 3, "should have three stream interceptors")
}

func TestApplyOptionsGRPC_AllOptions(t *testing.T) {
	// Applying all options at once should set all fields correctly.
	customLogger := slog.Default()

	unary := func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (any, error) {
		return nil, nil
	}

	stream := func(
		srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
	) error {
		return nil
	}

	cfg := applyOptionsGRPC(
		WithLogger(customLogger),
		WithUnaryInterceptors(unary),
		WithStreamInterceptors(stream),
	)

	td.Cmp(t, cfg.logger, customLogger)
	td.Cmp(t, len(cfg.unaryInterceptors), 1)
	td.Cmp(t, len(cfg.streamInterceptors), 1)
}
