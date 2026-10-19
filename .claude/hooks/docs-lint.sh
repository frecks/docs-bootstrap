#!/usr/bin/env bash
# PostToolUse hook: check a Markdown file in docs/ after Claude edits it.
# Errors are sent back to Claude with exit code 2 so it fixes them.
# Warnings and suggestions are passed along as context without blocking.

set -uo pipefail

root="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
file="$(jq -r '.tool_input.file_path // empty')"

case "$file" in
  "$root"/docs/*.md) ;;
  *) exit 0 ;;
esac
[ -f "$file" ] || exit 0

context() {
  jq -n --arg ctx "$1" '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $ctx}}'
}

if ! command -v vale > /dev/null; then
  context "Documentation checks skipped: Vale isn't installed. Tell the user; see docs/documentation/check-documentation.md."
  exit 0
fi

if out="$("$root/scripts/docs-lint.sh" "$file" 2>&1)"; then
  if grep -q -E ' (warning|suggestion) ' <<< "$out"; then
    context "scripts/docs-lint.sh passed with alerts. Fix them unless they don't apply:
$out"
  fi
  exit 0
fi

echo "scripts/docs-lint.sh found errors in $file. Fix them, following docs/standards/:
$out" >&2
exit 2
