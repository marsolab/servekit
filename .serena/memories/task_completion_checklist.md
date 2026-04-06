# Task Completion Checklist

After completing a coding task, run the following:

1. **Build**: `go build ./...` — Ensure no compilation errors
2. **Test**: `go test ./...` — Ensure all tests pass
3. **Lint**: `golangci-lint run` — Ensure no lint violations
4. **Fix lint issues if any**: `golangci-lint run --fix` — Auto-fix what's possible, then manually fix the rest

Remember:
- All comments must end with a period
- Use `any` not `interface{}`
- Max line length is 140 chars
- No TODO/FIXME comments
- Exhaustive switch statements on enums
