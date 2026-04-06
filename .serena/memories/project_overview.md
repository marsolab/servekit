# Project Overview

**Name**: servekit  
**Module**: `github.com/marsolab/servekit`  
**Purpose**: A collection of reusable Go components for building HTTP and gRPC APIs. Pre-v1.0.0, may have breaking changes.  
**Language**: Go 1.25  
**Platform**: Darwin (macOS)

## Tech Stack
- **HTTP Router**: go-chi/chi v5
- **gRPC**: google.golang.org/grpc
- **Databases**: PostgreSQL (pgx/v5), SQLite (mattn/go-sqlite3), MongoDB, Redis, ClickHouse
- **Auth**: JWT (cristalhq/jwt), OAuth (Kinde), bcrypt hashing
- **Logging**: slog with tint
- **Metrics**: VictoriaMetrics
- **Error Reporting**: Sentry
- **Testing**: go-testdeep
- **IDs**: ULID (oklog/ulid), XID (rs/xid)
- **Migrations**: jackc/tern
- **Email**: Resend API
- **Linting**: golangci-lint v2

## Package Structure
- `server.go` / `error.go` - Core server orchestrator and error types
- `httpkit/` - HTTP listener, router, middlewares, response helpers, status page
- `grpckit/` - gRPC listener, interceptors, response helpers
- `dbkit/` - Database connections: `pgkit`, `litekit`, `mongokit`, `rediskit`, `chkit`
- `dbkit/pgkit/pgmigrate` - PostgreSQL migrations with tern
- `errkit/` - Sentinel errors, error reporter, Sentry integration (`sentrykit`)
- `authkit/` - Auth: `jwtkit`, `hashkit`, `oauthkit/kinde`
- `logkit/` - Structured logging wrapper
- `idkit/` - ULID/XID generation
- `retry/` - Retry with backoff
- `ctxkit/` - Context utilities
- `mailkit/` - Email via `resendkit`
- `slackkit/` - Slack notifications
- `tern/` - Ternary operator helper

## Key Patterns
- **Listener-based architecture**: Server registers and runs multiple listeners concurrently
- **Functional options pattern**: Used extensively for configuration
- **Health check integration**: via `heartwilltell/hc`
- **Graceful shutdown**: Context-based with errgroup
