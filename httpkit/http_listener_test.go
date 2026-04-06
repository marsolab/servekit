package httpkit

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/heartwilltell/hc"
	"github.com/marsolab/servekit"
	td "github.com/maxatome/go-testdeep"
)

func TestNewListenerHTTP_DefaultConfig(t *testing.T) {
	l, err := NewListenerHTTP(":8080")

	td.CmpNoError(t, err, "creating listener with defaults should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
	td.CmpNot(t, l.router, nil, "router should not be nil.")
	td.CmpNot(t, l.server, nil, "server should not be nil.")
	td.CmpNot(t, l.logger, nil, "logger should not be nil.")
	td.Cmp(t, l.server.Addr, ":8080", "server address should match.")
	td.Cmp(t, l.enableTLS, false, "TLS should be disabled by default.")
}

func TestNewListenerHTTP_WithLogger(t *testing.T) {
	logger := slog.Default()
	l, err := NewListenerHTTP(":8080", WithLogger(logger))

	td.CmpNoError(t, err, "creating listener with logger should not error.")
	td.Cmp(t, l.logger, logger, "logger should be the one we provided.")
}

func TestNewListenerHTTP_WithNilLogger(t *testing.T) {
	l, err := NewListenerHTTP(":8080", WithLogger(nil))

	td.CmpNoError(t, err, "creating listener with nil logger should not error.")
	td.CmpNot(t, l.logger, nil, "logger should fall back to default, not nil.")
}

func TestNewListenerHTTP_WithHealthCheck(t *testing.T) {
	l, err := NewListenerHTTP(":8080", WithHealthCheck())

	td.CmpNoError(t, err, "creating listener with health check should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
}

func TestNewListenerHTTP_WithHealthCheckOptions(t *testing.T) {
	checker := hc.NewNopChecker()

	l, err := NewListenerHTTP(":8080",
		WithHealthCheck(
			HealthChecker(checker),
			HealthCheckRoute("/healthz"),
			HealthCheckAccessLog(true),
			HealthCheckMetricsForEndpoint(true),
		),
	)

	td.CmpNoError(t, err, "creating listener with health check options should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
}

func TestNewListenerHTTP_WithHealthCheckNilChecker(t *testing.T) {
	// HealthChecker with nil should not overwrite the default nop checker.
	l, err := NewListenerHTTP(":8080", WithHealthCheck(HealthChecker(nil)))

	td.CmpNoError(t, err, "nil checker should not cause error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
}

func TestNewListenerHTTP_WithMetrics(t *testing.T) {
	l, err := NewListenerHTTP(":8080", WithMetrics())

	td.CmpNoError(t, err, "creating listener with metrics should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
}

func TestNewListenerHTTP_WithMetricsOptions(t *testing.T) {
	l, err := NewListenerHTTP(":8080",
		WithMetrics(
			MetricsRoute("/custom-metrics"),
			MetricsAccessLog(true),
			MetricsMetricsForEndpoint(true),
		),
	)

	td.CmpNoError(t, err, "creating listener with metrics options should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
}

func TestNewListenerHTTP_WithProfiler(t *testing.T) {
	l, err := NewListenerHTTP(":8080", WithProfiler(PPROFConfig{}))

	td.CmpNoError(t, err, "creating listener with profiler should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
}

func TestNewListenerHTTP_WithProfilerCustomRoute(t *testing.T) {
	l, err := NewListenerHTTP(":8080", WithProfiler(PPROFConfig{
		route: "/custom-debug",
	}))

	td.CmpNoError(t, err, "creating listener with custom profiler route should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
}

func TestNewListenerHTTP_WithHTTPServerTimeouts(t *testing.T) {
	l, err := NewListenerHTTP(":8080",
		WithHTTPServerTimeouts(
			HTTPServerReadTimeout(5*time.Second),
			HTTPServerReadHeaderTimeout(15*time.Second),
			HTTPServerWriteTimeout(10*time.Second),
			HTTPServerIdleTimeout(30*time.Second),
		),
	)

	td.CmpNoError(t, err, "creating listener with timeouts should not error.")
	td.Cmp(t, l.server.ReadTimeout, 5*time.Second, "read timeout should match.")
	td.Cmp(t, l.server.ReadHeaderTimeout, 15*time.Second, "read header timeout should match.")
	td.Cmp(t, l.server.WriteTimeout, 10*time.Second, "write timeout should match.")
	td.Cmp(t, l.server.IdleTimeout, 30*time.Second, "idle timeout should match.")
}

func TestNewListenerHTTP_WithGlobalMiddlewares(t *testing.T) {
	var called bool
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	l, err := NewListenerHTTP(":8080", WithGlobalMiddlewares(mw))

	td.CmpNoError(t, err, "creating listener with global middlewares should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
	// called will be false until a request is made, just checking construction.
	td.Cmp(t, called, false, "middleware should not be called during construction.")
}

func TestNewListenerHTTP_WithTLS(t *testing.T) {
	l, err := NewListenerHTTP(":8443", WithTLS("/path/to/cert.pem", "/path/to/key.pem"))

	td.CmpNoError(t, err, "creating listener with TLS should not error.")
	td.Cmp(t, l.enableTLS, true, "TLS should be enabled.")
	td.Cmp(t, l.cert, "/path/to/cert.pem", "cert path should match.")
	td.Cmp(t, l.key, "/path/to/key.pem", "key path should match.")
}

func TestNewListenerHTTP_WithTLS_MissingCert(t *testing.T) {
	_, err := NewListenerHTTP(":8443", WithTLS("", "/path/to/key.pem"))

	td.CmpError(t, err, "missing cert should cause an error.")
	td.Cmp(t, err.Error(), "configure TLS: "+servekit.ErrCertPathRequired.Error(),
		"error should indicate missing cert.")
}

func TestNewListenerHTTP_WithTLS_MissingKey(t *testing.T) {
	_, err := NewListenerHTTP(":8443", WithTLS("/path/to/cert.pem", ""))

	td.CmpError(t, err, "missing key should cause an error.")
	td.Cmp(t, err.Error(), "configure TLS: "+servekit.ErrPrivateKeyPathRequired.Error(),
		"error should indicate missing key.")
}

func TestNewListenerHTTP_AllFeatures(t *testing.T) {
	logger := slog.Default()

	l, err := NewListenerHTTP(":9090",
		WithLogger(logger),
		WithHealthCheck(
			HealthCheckRoute("/ready"),
			HealthCheckReportJSON(),
		),
		WithMetrics(
			MetricsRoute("/prometheus"),
		),
		WithProfiler(PPROFConfig{route: "/pprof"}),
		WithHTTPServerTimeouts(
			HTTPServerReadTimeout(3*time.Second),
		),
	)

	td.CmpNoError(t, err, "creating listener with all features should not error.")
	td.CmpNot(t, l, nil, "listener should not be nil.")
	td.Cmp(t, l.server.Addr, ":9090", "server address should match.")
	td.Cmp(t, l.logger, logger, "logger should match.")
}

func TestApplyOptionsHTTP_Defaults(t *testing.T) {
	cfg := applyOptionsHTTP()

	td.Cmp(t, cfg.timeouts.readTimeout, readTimeout, "default read timeout should match.")
	td.Cmp(t, cfg.timeouts.readHeaderTimeout, readHeaderTimeout, "default read header timeout should match.")
	td.Cmp(t, cfg.timeouts.writeTimeout, writeTimeout, "default write timeout should match.")
	td.Cmp(t, cfg.timeouts.idleTimeout, idleTimeout, "default idle timeout should match.")
	td.Cmp(t, cfg.health.enable, false, "health should be disabled by default.")
	td.Cmp(t, cfg.health.route, "/health", "default health route should be /health.")
	td.Cmp(t, cfg.metrics.enable, false, "metrics should be disabled by default.")
	td.Cmp(t, cfg.metrics.route, "/metrics", "default metrics route should be /metrics.")
	td.Cmp(t, cfg.profiler.enable, false, "profiler should be disabled by default.")
	td.Cmp(t, cfg.profiler.route, "/debug", "default profiler route should be /debug.")
	td.CmpNot(t, cfg.logger, nil, "default logger should not be nil.")
	td.Cmp(t, len(cfg.globalMiddlewares), 0, "default global middlewares should be empty.")
}

func TestApplyOptionsHTTP_WithOptions(t *testing.T) {
	cfg := applyOptionsHTTP(
		WithHealthCheck(),
		WithMetrics(),
	)

	td.Cmp(t, cfg.health.enable, true, "health should be enabled.")
	td.Cmp(t, cfg.metrics.enable, true, "metrics should be enabled.")
}

func TestHealthCheckReportJSON(t *testing.T) {
	cfg := HealthConfig{}
	HealthCheckReportJSON()(&cfg)

	td.Cmp(t, cfg.healthReport, healthReportJSON, "health report should be JSON.")
}

func TestHealthCheckReportHTML(t *testing.T) {
	cfg := HealthConfig{}
	HealthCheckReportHTML()(&cfg)

	td.Cmp(t, cfg.healthReport, healthReportHTML, "health report should be HTML.")
}

func TestNewListenerOption(t *testing.T) {
	opts := NewListenerOption(
		WithHealthCheck(),
		WithMetrics(),
	)

	td.Cmp(t, len(opts), 2, "should have 2 options.")
}

func TestMount(t *testing.T) {
	l, err := NewListenerHTTP(":8080")
	td.CmpNoError(t, err, "creating listener should not error.")

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Should not panic.
	l.Mount("/api", handler)
}

func TestMountGroup(t *testing.T) {
	l, err := NewListenerHTTP(":8080")
	td.CmpNoError(t, err, "creating listener should not error.")

	// Should not panic.
	l.MountGroup("/api/v1", func(r chi.Router) {
		r.Get("/users", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})
}
