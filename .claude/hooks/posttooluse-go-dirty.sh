#!/usr/bin/env bash
# PostToolUse hook: mark session as dirty when .go files are edited.
# Deletes .checks-passed so checks must be re-run.

set -euo pipefail

input=$(cat)

# Extract file_path from JSON input.
file_path=$(echo "$input" | python3 -c "
import sys, json
d = json.load(sys.stdin)
ti = d.get('tool_input', {})
print(ti.get('file_path', ''))
" 2>/dev/null || echo "")

# Check if the edited file is a .go file.
if echo "$file_path" | grep -q '\.go$'; then
  touch .claude/.go-dirty
  rm -f .claude/.checks-passed
fi

echo '{}'
