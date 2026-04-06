package httpkit

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/cors"
	td "github.com/maxatome/go-testdeep"
)

func TestLoggingMiddleware_Success(t *testing.T) {
	logger := slog.Default()
	mw := LoggingMiddleware(logger)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(w, r)

	td.Cmp(t, w.Code, http.StatusOK, "handler should return 200.")
}

func TestLoggingMiddleware_ServerError(t *testing.T) {
	logger := slog.Default()
	mw := LoggingMiddleware(logger)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/resource", nil)

	handler.ServeHTTP(w, r)

	td.Cmp(t, w.Code, http.StatusInternalServerError, "handler should return 500.")
}

func TestLoggingMiddleware_ServerErrorWithLogHook(t *testing.T) {
	logger := slog.Default()
	mw := LoggingMiddleware(logger)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The middleware sets a log error hook on the context.
		// ErrorHTTP or direct hook usage should trigger it.
		ErrorHTTP(w, r, errTestInternal)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/fail", nil)

	handler.ServeHTTP(w, r)

	td.Cmp(t, w.Code, http.StatusInternalServerError, "handler should return 500.")
}

func TestLoggingMiddleware_PassesRequestThrough(t *testing.T) {
	logger := slog.Default()
	mw := LoggingMiddleware(logger)

	var called bool

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/resource/1", nil)

	handler.ServeHTTP(w, r)

	td.CmpTrue(t, called, "inner handler should be called.")
	td.Cmp(t, w.Code, http.StatusNoContent, "handler should return 204.")
}

func TestHttpReqDurationStr(t *testing.T) {
	result := httpReqDurationStr("GET", "/api/v1/users", "200")

	expected := `http_request_duration{method="GET", route="/api/v1/users", code="200"}`
	td.Cmp(t, result, expected, "duration metric string should match expected format.")
}

func TestHttpReqTotalStr(t *testing.T) {
	result := httpReqTotalStr("POST", "/api/v1/users", "201")

	expected := `http_requests_total{method="POST", route="/api/v1/users", code="201"}`
	td.Cmp(t, result, expected, "total metric string should match expected format.")
}

func TestHttpReqDurationStr_VariousInputs(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		route    string
		status   string
		expected string
	}{
		{
			name:     "GET request with 200.",
			method:   "GET",
			route:    "/",
			status:   "200",
			expected: `http_request_duration{method="GET", route="/", code="200"}`,
		},
		{
			name:     "POST request with 500.",
			method:   "POST",
			route:    "/api/create",
			status:   "500",
			expected: `http_request_duration{method="POST", route="/api/create", code="500"}`,
		},
		{
			name:     "DELETE request with 404.",
			method:   "DELETE",
			route:    "/items/{id}",
			status:   "404",
			expected: `http_request_duration{method="DELETE", route="/items/{id}", code="404"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := httpReqDurationStr(tt.method, tt.route, tt.status)
			td.Cmp(t, got, tt.expected, "metric string should match.")
		})
	}
}

func TestHttpReqTotalStr_VariousInputs(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		route    string
		status   string
		expected string
	}{
		{
			name:     "GET request with 200.",
			method:   "GET",
			route:    "/health",
			status:   "200",
			expected: `http_requests_total{method="GET", route="/health", code="200"}`,
		},
		{
			name:     "PUT request with 204.",
			method:   "PUT",
			route:    "/api/update",
			status:   "204",
			expected: `http_requests_total{method="PUT", route="/api/update", code="204"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := httpReqTotalStr(tt.method, tt.route, tt.status)
			td.Cmp(t, got, tt.expected, "metric string should match.")
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	mw := RecoveryMiddleware()

	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Should not panic.
	handler.ServeHTTP(w, r)

	td.Cmp(t, w.Code, http.StatusInternalServerError,
		"recovery middleware should return 500 after panic.")
}

func TestRedirectSlashesMiddleware(t *testing.T) {
	mw := RedirectSlashesMiddleware()

	td.CmpNot(t, mw, nil, "redirect slashes middleware should not be nil.")
}

func TestCORSMiddleware(t *testing.T) {
	// Verify CORS middleware can be created without panic.
	mw := CORSMiddleware(corsOptionsForTest())

	td.CmpNot(t, mw, nil, "CORS middleware should not be nil.")
}

func TestProfilerMiddleware(t *testing.T) {
	handler := ProfilerMiddleware()

	td.CmpNot(t, handler, nil, "profiler middleware should not be nil.")
}

// errTestInternal is a test error for middleware tests.
var errTestInternal = http.ErrServerClosed

// corsOptionsForTest returns CORS options suitable for testing.
func corsOptionsForTest() cors.Options {
	return cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
	}
}
