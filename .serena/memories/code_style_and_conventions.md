# Code Style and Conventions

## General
- Go 1.25, standard Go conventions
- All code comments must end with a period
- Max line length: 140 characters (enforced by golines)
- Use `any` instead of `interface{}` (enforced by gofmt rewrite rule)
- Formatters: gofmt, goimports, golines

## Naming
- Standard Go naming conventions (camelCase for unexported, PascalCase for exported)
- Sentinel errors prefixed with `Err` (enforced by errname linter)
- Printf-like functions named with `f` suffix

## Patterns
- **Functional options**: `type Option[T Constraint] func(o *T)` for configuration
- **Health checks**: Components implement `hc.HealthChecker` interface
- **Middleware**: chi standard `func(http.Handler) http.Handler`
- **Response helpers**: Use `httpkit.JSON()`, `httpkit.HTML()`, `httpkit.ErrorHTTP()`

## Testing
- Use `github.com/maxatome/go-testdeep` for assertions
- Table-driven tests where applicable
- Test graceful shutdown with context cancellation
- Mock implementations of interfaces

## Linting
- golangci-lint v2 with extensive linter set
- Key linters: errcheck, govet, gosec, staticcheck, revive, errorlint, cyclop, gocognit
- No init functions (gochecknoinits)
- No TODO/FIXME comments (godox)
- Exhaustive enum switches required
