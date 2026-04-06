#!/usr/bin/env bash
# Stop hook: block if Go files changed this session without passing checks.
# Self-contained — detects changes directly, no dependency on .go-dirty.
#
# Markers:
#   .claude/.session-start       — timestamp for session start
#   .claude/.go-snapshot-start   — .go file list at session start
#   .claude/.checks-passed       — set by mark-checks-passed.sh
#   .claude/.go-snapshot         — .go file list at time of last check pass

set -euo pipefail
cat > /dev/null  # consume stdin

BLOCK_MSG='{
  "decision": "block",
  "reason": "Go files were modified this session. Before stopping you MUST:\n1. Run `bash .claude/hooks/mark-checks-passed.sh`\n2. Fix ALL lint errors and test failures — even pre-existing ones"
}'

# No session-start marker → not a tracked session, allow.
if [ ! -f .claude/.session-start ]; then
  echo '{}'
  exit 0
fi

# --- Detect whether .go files changed since session start ---

go_changed=false

# 1. Any .go file newer than session start?
newer=$(find . -maxdepth 5 -name "*.go" -not -path "./.git/*" -newer .claude/.session-start -print -quit 2>/dev/null)
if [ -n "$newer" ]; then
  go_changed=true
fi

# 2. .go file list differs from session-start snapshot?
#    Missing snapshot → fail closed (assume changed).
if [ "$go_changed" = "false" ]; then
  if [ ! -f .claude/.go-snapshot-start ]; then
    go_changed=true
  else
    current=$(find . -maxdepth 5 -name "*.go" -not -path "./.git/*" | sort)
    saved=$(cat .claude/.go-snapshot-start)
    if [ "$current" != "$saved" ]; then
      go_changed=true
    fi
  fi
fi

# No Go changes this session → allow.
if [ "$go_changed" = "false" ]; then
  echo '{}'
  exit 0
fi

# --- Go files changed — validate .checks-passed marker ---

# No marker → block.
if [ ! -f .claude/.checks-passed ]; then
  echo "$BLOCK_MSG"
  exit 0
fi

# Snapshot must exist alongside marker.
if [ ! -f .claude/.go-snapshot ]; then
  rm -f .claude/.checks-passed
  echo "$BLOCK_MSG"
  exit 0
fi

# Current .go file list must match check-time snapshot.
current=${current:-$(find . -maxdepth 5 -name "*.go" -not -path "./.git/*" | sort)}
check_saved=$(cat .claude/.go-snapshot)
if [ "$current" != "$check_saved" ]; then
  rm -f .claude/.checks-passed
  echo "$BLOCK_MSG"
  exit 0
fi

# No .go file may be newer than the marker.
newer_than_marker=$(find . -maxdepth 5 -name "*.go" -not -path "./.git/*" -newer .claude/.checks-passed -print -quit 2>/dev/null)
if [ -n "$newer_than_marker" ]; then
  rm -f .claude/.checks-passed
  echo "$BLOCK_MSG"
  exit 0
fi

# Marker is valid — allow stop.
echo '{}'
