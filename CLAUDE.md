# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`servekit` is a collection of reusable Go components for building HTTP and gRPC APIs. The project is under active development (pre-v1.0.0) and may have breaking changes.

## Table of Contents

- [Project Overview](#project-overview)
- [Table of Contents](#table-of-contents)
- [Building and Testing](#building-and-testing)
- [Architecture](#architecture)

## Code Style

### Comments

- All code comments should end with a period.
- Comments should be clear and concise, explaining the purpose and functionality of the code.
- Comments should be written in a way that is easy to understand and maintain.

## Building and Testing

### Build Commands

```bash
# Build all packages.
go build ./...
```

### Testing Commands

```bash
# Run all tests.
go test ./...

# Run tests with verbose output.
go test -v ./...

# Run specific test.
go test -run TestServer_GracefulShutdown ./...
```

### Linting

```bash
# Run golangci-lint.
golangci-lint run

# Run golangci-lint with auto-fix.
golangci-lint run --fix
```

The project uses golangci-lint with multiple enabled linters (see `.golangci.yml`). Key linters include: errcheck, govet, gosec, staticcheck, revive, and errorlint.

## Architecture

### Core Server Pattern

The foundation is a **Listener-based architecture** where:

1. **Server** (`server.go`) - Central orchestrator that manages multiple listeners
   - Registers listeners by name via `RegisterListener()`
   - Runs all listeners concurrently using `errgroup`
   - Handles graceful shutdown across all listeners

2. **Listener Interface** - Any component implementing `Serve(ctx context.Context) error`
   - `ListenerHTTP` (`httpkit/http_listener.go`) - HTTP server wrapper around chi router
   - `ListenerGRPC` (`grpckit/grpc_listener.go`) - gRPC server wrapper

3. **Graceful Shutdown** - Built-in context-based shutdown handling
   - Listeners monitor context cancellation
   - Default 5-second shutdown timeout
   - Server waits for all listeners to stop cleanly

### HTTP Architecture (httpkit)

- **Router**: Built on `go-chi/chi` for routing and middleware
- **Functional Options Pattern**: Used extensively for configuration (`ListenerOption[T]`)
- **Built-in Endpoints**: Health, metrics (Prometheus), and pprof profiler
- **Middleware Chain**: Logging, metrics collection, recovery, CORS support
- **Response Helpers**: Standardized response functions in `respond.go` (JSON, HTML, TEXT, ErrorHTTP)
- **Error Mapping**: Automatic HTTP status code mapping from `errkit.Error` types

### gRPC Architecture (grpckit)

- **Interceptor Chain**: Unary and stream interceptors configured via options
- **Graceful Shutdown**: Attempts graceful stop, falls back to forced stop after timeout
- **Mount Pattern**: Services registered via `GRPCEndpointRegistrator` interface

### Database Integrations (dbkit)

- **pgkit**: PostgreSQL via pgx/v5 with connection pooling
- **litekit**: SQLite with Litestream backup support (S3 or file-based)
- **mongokit**: MongoDB connections
- **rediskit**: Redis client wrapper
- **pgmigrate**: Database migration using jackc/tern

All database connections implement `hc.HealthChecker` interface for health checks.

### Error Handling (errkit)

- **Sentinel Errors**: Package defines standard errors (`ErrNotFound`, `ErrAlreadyExists`, `ErrUnauthenticated`, etc.)
- **Error Reporting**: Pluggable error reporter interface with Sentry integration (`errkit/sentrykit`)
- **Context Integration**: Errors can be logged via context hooks (`ctxkit.SetLogErrHook`)

### Authentication (authkit)

- **jwtkit**: JWT token management using cristalhq/jwt
  - Token signing and verification
  - Claims validation (expiry, not-before, issued-at)
- **hashkit**: Password hashing utilities
- **oauthkit**: OAuth integrations (e.g., Kinde provider)

### Utilities

- **idkit**: ULID generation for unique identifiers
- **logkit**: Structured logging wrapper around slog
- **retry**: Retry mechanisms with backoff strategies
- **ctxkit**: Context management utilities (including error logging hooks)
- **mailkit**: Email sending via Resend API
- **slackkit**: Slack notification utilities
- **tern**: Ternary operator helper

## Key Patterns

### Functional Options Pattern

Configuration throughout the codebase uses functional options with generic constraints:

```go
type ListenerOption[T ListenerOptionConstraint] func(o *T)

// Usage
listener := NewListenerHTTP(":8080",
    WithLogger(logger),
    WithHealthCheck(),
    WithMetrics(),
)
```

### Health Check Integration

Components implement `hc.HealthChecker` interface from `github.com/heartwilltell/hc`:

- Database connections automatically support health checks
- HTTP listener exposes health endpoint with configurable formats (text/JSON/HTML)
- Multi-service health checking with `MultiServiceChecker`

### Middleware Composition

HTTP middleware uses chi's standard `func(http.Handler) http.Handler` pattern:

- `LoggingMiddleware`: Structured logging with request details
- `MetricsMiddleware`: Prometheus metrics collection
- `RecoveryMiddleware`: Panic recovery
- `CORSMiddleware`: CORS configuration

### Response Handling

Use `httpkit.JSON()`, `httpkit.HTML()`, `httpkit.ErrorHTTP()` for standardized responses:

- Automatic error-to-status-code mapping
- Context-aware error logging
- Configurable headers and status codes via `ResponseOption`

## Testing Patterns

- Mock implementations of interfaces (e.g., `mockListener` in `server_test.go`)
- Table-driven tests where applicable
- Use `github.com/maxatome/go-testdeep` for assertions
- Test graceful shutdown scenarios with context cancellation
- Background goroutines with error channels for async behavior testing

## Common Development Workflows

### Adding a New HTTP Endpoint

1. Create handler function with signature `func(w http.ResponseWriter, r *http.Request)`
2. Use `httpkit.JSON()` or `httpkit.ErrorHTTP()` for responses
3. Mount to listener: `listener.Mount("/api", handler, middlewares...)`
4. Add tests covering success and error cases

### Adding a New gRPC Service

1. Define protobuf service and generate code
2. Implement service interface
3. Create `GRPCEndpointRegistrator` implementation with `Mount(*grpc.Server)` method
4. Register with listener: `listener.Mount(serviceRegistrator)`

### Database Schema Evolution (SQLite)

Use the schema evolution helper in `litekit/schema_evolution.go` for managing SQLite schema changes with proper transaction handling.
