#!/usr/bin/env bash
# Runs golangci-lint and go test, only marks checks as passed if BOTH succeed.

set -euo pipefail

echo "Running golangci-lint run ..."
if ! golangci-lint run; then
  echo "FAIL: golangci-lint reported issues. Fix them and re-run."
  exit 1
fi

echo "Running go test -race -cover ./... ..."
if ! go test -race -cover ./...; then
  echo "FAIL: go test reported failures. Fix them and re-run."
  exit 1
fi

touch .claude/.checks-passed
find . -maxdepth 5 -name "*.go" -not -path "./.git/*" | sort > .claude/.go-snapshot
echo "All checks passed. Marker set."
