package grpckit

import (
	"context"
	"errors"
	"testing"

	"github.com/marsolab/servekit/ctxkit"
	"github.com/marsolab/servekit/logkit"
	"github.com/maxatome/go-testdeep/td"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func TestGrpcReqDurationStr(t *testing.T) {
	// grpcReqDurationStr should build a valid metrics string.
	got := grpcReqDurationStr("/test.Service/Method", "0")
	want := `grpc_request_duration{route="/test.Service/Method", code="0"}`

	td.Cmp(t, got, want)
}

func TestGrpcReqTotalStr(t *testing.T) {
	// grpcReqTotalStr should build a valid metrics string.
	got := grpcReqTotalStr("/test.Service/Method", "0")
	want := `grpc_requests_total{route="/test.Service/Method", code="0"}`

	td.Cmp(t, got, want)
}

func TestGrpcReqDurationStr_DifferentCodes(t *testing.T) {
	// Different status codes should produce different metric strings.
	tests := []struct {
		name  string
		route string
		code  string
		want  string
	}{
		{
			name:  "ok code",
			route: "/pkg.Svc/Call",
			code:  "0",
			want:  `grpc_request_duration{route="/pkg.Svc/Call", code="0"}`,
		},
		{
			name:  "not found code",
			route: "/pkg.Svc/Get",
			code:  "5",
			want:  `grpc_request_duration{route="/pkg.Svc/Get", code="5"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := grpcReqDurationStr(tc.route, tc.code)
			td.Cmp(t, got, tc.want)
		})
	}
}

func TestGrpcReqTotalStr_DifferentCodes(t *testing.T) {
	// Different status codes should produce different metric strings.
	tests := []struct {
		name  string
		route string
		code  string
		want  string
	}{
		{
			name:  "ok code",
			route: "/pkg.Svc/Call",
			code:  "0",
			want:  `grpc_requests_total{route="/pkg.Svc/Call", code="0"}`,
		},
		{
			name:  "internal error code",
			route: "/pkg.Svc/Create",
			code:  "13",
			want:  `grpc_requests_total{route="/pkg.Svc/Create", code="13"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := grpcReqTotalStr(tc.route, tc.code)
			td.Cmp(t, got, tc.want)
		})
	}
}

func TestMetricsInterceptor_Success(t *testing.T) {
	// MetricsInterceptor should call the handler and return its result on success.
	interceptor := MetricsInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/Method"}

	resp, err := interceptor(context.Background(), "request", info, handler)

	td.CmpNoError(t, err)
	td.Cmp(t, resp, "response")
}

func TestMetricsInterceptor_Error(t *testing.T) {
	// MetricsInterceptor should propagate errors from the handler.
	interceptor := MetricsInterceptor()

	grpcErr := status.Error(5, "not found")

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, grpcErr
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/Method"}

	resp, err := interceptor(context.Background(), "request", info, handler)

	td.CmpNil(t, resp)
	td.CmpError(t, err)
}

func TestMetricsInterceptor_NonGRPCError(t *testing.T) {
	// MetricsInterceptor should handle non-gRPC errors gracefully.
	interceptor := MetricsInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("plain error")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/Method"}

	resp, err := interceptor(context.Background(), "request", info, handler)

	td.CmpNil(t, resp)
	td.CmpError(t, err)
}

func TestLoggingInterceptor_Success(t *testing.T) {
	// LoggingInterceptor should call the handler and return its result on success.
	logger := logkit.NewNop()
	interceptor := LoggingInterceptor(logger)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/Method"}

	resp, err := interceptor(context.Background(), "request", info, handler)

	td.CmpNoError(t, err)
	td.Cmp(t, resp, "ok")
}

func TestLoggingInterceptor_GRPCError(t *testing.T) {
	// LoggingInterceptor should log and return gRPC status errors.
	logger := logkit.NewNop()
	interceptor := LoggingInterceptor(logger)

	grpcErr := status.Error(5, "not found")

	// The handler must call the log error hook so reqErr is not nil.
	handler := func(ctx context.Context, req any) (any, error) {
		if hook := ctxkit.GetLogErrHook(ctx); hook != nil {
			hook(grpcErr)
		}
		return nil, grpcErr
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/Method"}

	resp, err := interceptor(context.Background(), "request", info, handler)

	td.CmpNil(t, resp)
	td.CmpError(t, err)
}
