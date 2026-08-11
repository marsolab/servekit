# servekit

A collection of reusable Go components for building HTTP and gRPC APIs. Designed to reduce boilerplate and provide consistent patterns across services.

**Warning:** Pre-v1.0.0 - expect breaking changes.

[![Go Reference](https://pkg.go.dev/badge/github.com/marsolab/servekit.svg)](https://pkg.go.dev/github.com/marsolab/servekit)

## Install

```bash
go get github.com/marsolab/servekit
```

Requires **Go 1.25+**.

## Quick Start

```go
package main

import (
    "github.com/marsolab/servekit"
    "github.com/marsolab/servekit/httpkit"
    "github.com/marsolab/servekit/logkit"
)

func main() {
    logger := logkit.New(logkit.WithJSON())

    // Create a server that manages multiple listeners.
    srv := servekit.NewServer(logger)

    // Create an HTTP listener with built-in health checks and metrics.
    http := httpkit.NewListenerHTTP(":8080",
        httpkit.WithLogger(logger),
        httpkit.WithHealthCheck(),
        httpkit.WithMetrics(),
    )

    srv.RegisterListener("http", http)

    // Serve blocks until all listeners stop or the context is canceled.
    if err := srv.Serve(context.Background()); err != nil {
        logger.Error("server failed", slog.String("error", err.Error()))
        os.Exit(1)
    }
}
```

## Architecture

The foundation is a **Listener-based architecture**:

- **Server** orchestrates multiple listeners, runs them concurrently via `errgroup`, and handles graceful shutdown.
- **Listener** is any component implementing `Serve(ctx context.Context) error`.
- Built-in listeners: `httpkit.ListenerHTTP` and `grpckit.ListenerGRPC`.
- Graceful shutdown propagates through context cancellation with configurable timeouts.

Configuration throughout the codebase uses the **functional options pattern**:

```go
listener := httpkit.NewListenerHTTP(":8080",
    httpkit.WithLogger(logger),
    httpkit.WithHealthCheck(),
    httpkit.WithMetrics(),
    httpkit.WithCORS(corsOpts),
)
```

## Packages

### Core

| Package | Description |
|---------|-------------|
| `servekit` (root) | Server orchestrator - manages and runs multiple listeners concurrently |
| [`httpkit`](httpkit/) | HTTP server built on [chi](https://github.com/go-chi/chi) with routing, middleware (logging, metrics, recovery, CORS), and standardized response helpers |
| [`httpkit/statuspage`](httpkit/statuspage/) | HTML status page template for health check endpoints |
| [`grpckit`](grpckit/) | gRPC server with interceptor chains, graceful shutdown, and response utilities |

### Database

| Package | Description |
|---------|-------------|
| [`dbkit/pgkit`](dbkit/pgkit/) | PostgreSQL via [pgx/v5](https://github.com/jackc/pgx/v5) with connection pooling and health checks |
| [`dbkit/pgkit/pgmigrate`](dbkit/pgkit/pgmigrate/) | PostgreSQL migrations using [tern](https://github.com/jackc/tern/v2) |
| [`dbkit/litekit`](dbkit/litekit/) | SQLite with [Litestream](https://github.com/benbjohnson/litestream) backup support (S3 or file-based) and schema evolution |
| [`dbkit/chkit`](dbkit/chkit/) | ClickHouse client via [clickhouse-go](https://github.com/ClickHouse/clickhouse-go) |
| [`dbkit/mongokit`](dbkit/mongokit/) | MongoDB connections |
| [`dbkit/rediskit`](dbkit/rediskit/) | Redis client wrapper via [go-redis](https://github.com/redis/go-redis/v9) |

All database packages implement the `hc.HealthChecker` interface for health checks.

### Authentication

| Package | Description |
|---------|-------------|
| [`authkit/jwtkit`](authkit/jwtkit/) | JWT token signing, verification, and claims validation using [cristalhq/jwt](https://github.com/cristalhq/jwt/v5) |
| [`authkit/hashkit`](authkit/hashkit/) | Password hashing utilities |
| [`authkit/oauthkit`](authkit/oauthkit/) | OAuth provider integrations (Kinde) |

### Error Handling

| Package | Description |
|---------|-------------|
| [`errkit`](errkit/) | Sentinel errors (`ErrNotFound`, `ErrAlreadyExists`, `ErrUnauthenticated`, etc.) with automatic HTTP status code mapping |
| [`errkit/sentrykit`](errkit/sentrykit/) | Sentry integration for error reporting |

### Utilities

| Package | Description |
|---------|-------------|
| [`logkit`](logkit/) | Structured logging wrapper around `log/slog` with colored terminal output via [tint](https://github.com/lmittmann/tint) |
| [`ctxkit`](ctxkit/) | Context management utilities including error logging hooks |
| [`idkit`](idkit/) | ULID generation using [oklog/ulid](https://github.com/oklog/ulid/v2) and XID via [rs/xid](https://github.com/rs/xid) |
| [`retry`](retry/) | Retry mechanisms with configurable backoff strategies |
| [`mailkit`](mailkit/) | Email sending via [Resend](https://github.com/resend/resend-go/v2) API |
| [`slackkit`](slackkit/) | Slack notifications |
| [`tgkit`](tgkit/) | Telegram bots, channel administration, Mini App authentication, invoices, and subscriptions |
| [`tern`](tern/) | Ternary operator helper |

## HTTP Middleware

`httpkit` provides composable middleware following chi's `func(http.Handler) http.Handler` pattern:

- **LoggingMiddleware** - structured request/response logging
- **MetricsMiddleware** - Prometheus metrics collection via [VictoriaMetrics](https://github.com/VictoriaMetrics/metrics)
- **RecoveryMiddleware** - panic recovery with error reporting
- **CORSMiddleware** - CORS configuration via [go-chi/cors](https://github.com/go-chi/cors)

## Response Helpers

Standardized response functions for both HTTP and gRPC:

```go
// HTTP responses.
httpkit.JSON(w, http.StatusOK, data)
httpkit.HTML(w, http.StatusOK, template, data)
httpkit.TEXT(w, http.StatusOK, "ok")
httpkit.ErrorHTTP(w, r, err) // maps errkit errors to HTTP status codes automatically

// gRPC responses.
grpckit.Error(err) // maps errkit errors to gRPC status codes
```

## Health Checks

Components implement the [`hc.HealthChecker`](https://github.com/heartwilltell/hc) interface. The HTTP listener can expose a health endpoint aggregating checks from all registered services:

```go
listener := httpkit.NewListenerHTTP(":8080",
    httpkit.WithHealthCheck(pgConn, redisConn, mongoConn),
)
// GET /health returns aggregated health status.
```

## Testing

```bash
go test ./...
```

The project uses [`go-testdeep`](https://github.com/maxatome/go-testdeep) for assertions and table-driven tests throughout.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE.md)
