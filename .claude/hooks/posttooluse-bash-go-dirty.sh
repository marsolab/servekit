#!/usr/bin/env bash
# PostToolUse hook for Bash: detect .go file changes by comparing
# filesystem timestamps AND file-list snapshots against session markers.
# Catches modifications, creations, deletions, and renames.

set -euo pipefail
cat > /dev/null  # consume stdin

# --- Timestamp check: any .go file newer than reference? ---
ref=""
if [ -f .claude/.checks-passed ]; then
  ref=.claude/.checks-passed
elif [ -f .claude/.session-start ]; then
  ref=.claude/.session-start
fi

if [ -n "$ref" ]; then
  newer=$(find . -maxdepth 5 -name "*.go" -not -path "./.git/*" -newer "$ref" -print -quit 2>/dev/null)
  if [ -n "$newer" ]; then
    touch .claude/.go-dirty
    rm -f .claude/.checks-passed
    echo '{}'
    exit 0
  fi
fi

# --- Snapshot check: .go file list changed (catches deletes/renames)? ---
if [ -f .claude/.go-snapshot ]; then
  current=$(find . -maxdepth 5 -name "*.go" -not -path "./.git/*" | sort)
  saved=$(cat .claude/.go-snapshot)
  if [ "$current" != "$saved" ]; then
    touch .claude/.go-dirty
    rm -f .claude/.checks-passed
    echo '{}'
    exit 0
  fi
fi

echo '{}'
