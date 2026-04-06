# Suggested Commands

## Build
```bash
go build ./...
```

## Test
```bash
# Run all tests
go test ./...

# Verbose
go test -v ./...

# Specific test
go test -run TestServer_GracefulShutdown ./...
```

## Lint & Format
```bash
# Run linter
golangci-lint run

# Run linter with auto-fix
golangci-lint run --fix
```

## Git (Darwin)
```bash
git status
git log --oneline -10
git diff
git blame <file>
```

## System Utilities (Darwin)
```bash
ls, find, grep, cd, cat, head, tail
```
