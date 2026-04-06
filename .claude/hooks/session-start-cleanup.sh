#!/usr/bin/env bash
# SessionStart hook: clean stale markers from previous sessions.

set -euo pipefail
cat > /dev/null  # consume stdin

rm -f .claude/.go-dirty .claude/.checks-passed .claude/.session-start .claude/.go-snapshot .claude/.go-snapshot-start
touch .claude/.session-start
find . -maxdepth 5 -name "*.go" -not -path "./.git/*" | sort > .claude/.go-snapshot-start
echo '{}'
