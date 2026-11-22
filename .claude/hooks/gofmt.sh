#!/usr/bin/env bash
set -euo pipefail

# PostToolUse hook: format Go files after Edit/Write.
# Reads the hook payload from stdin and does nothing for other file types.

payload=$(cat)

if command -v jq >/dev/null 2>&1; then
  file=$(printf '%s' "$payload" | jq -r '.tool_input.file_path // empty')
else
  file=$(printf '%s' "$payload" | python3 -I -c '
import json, sys
try:
    data = json.load(sys.stdin)
except Exception:
    print("")
else:
    print(data.get("tool_input", {}).get("file_path", ""))
' 2>/dev/null || true)
fi

[ -n "$file" ] || exit 0

case "$file" in
  *.go) ;;
  *) exit 0 ;;
esac

[ -f "$file" ] || exit 0

if gofmt -l "$file" | grep -q .; then
  gofmt -w "$file"
  echo "gofmt: formatted $file"
fi
