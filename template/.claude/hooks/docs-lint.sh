#!/usr/bin/env bash
# PostToolUse hook: check a Markdown file in docs/ after Claude edits it.
# Errors are sent back to Claude with exit code 2 so it fixes them.
# Warnings, suggestions, and checks that couldn't run are passed along as
# context without blocking.

set -uo pipefail

# The hook reads its input with jq; without it, there's nothing to check.
command -v jq > /dev/null || exit 0

file="$(jq -r '.tool_input.file_path // empty')"
[ -f "$file" ] || exit 0
file="$(realpath -- "$file")" || exit 0

project="$(realpath -- "${CLAUDE_PROJECT_DIR:-$(dirname "${BASH_SOURCE[0]}")/../..}")" || exit 0

# common_dir prints the git directory that a checkout's worktrees share.
common_dir() {
  git -C "$1" rev-parse --path-format=absolute --git-common-dir 2> /dev/null
}

# Check the checkout that holds the edited file, if it's a worktree of this
# project: $CLAUDE_PROJECT_DIR can name the main checkout while the session
# works in a worktree. Any other repository, such as one that this one
# contains, isn't checked.
root="$project"
if top="$(git -C "$(dirname "$file")" rev-parse --show-toplevel 2> /dev/null)" &&
  [ "$(common_dir "$top")" = "$(common_dir "$project")" ]; then
  root="$top"
fi

case "$file" in
  "$root"/docs/*.md) ;;
  *) exit 0 ;;
esac
[ -x "$root/scripts/docs-lint.sh" ] || exit 0

context() {
  jq -n --arg ctx "$1" '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $ctx}}'
}

# An edited script kept as a conflict may predate --with-referrers.
args=(--with-referrers)
grep -q -e '--with-referrers' "$root/scripts/docs-lint.sh" || args=()
out="$("$root/scripts/docs-lint.sh" "${args[@]}" "$file" 2>&1)"
case $? in
  0)
    if grep -q -E ' (warning|suggestion) ' <<< "$out"; then
      context "scripts/docs-lint.sh passed with alerts. Fix them unless they don't apply:
$out"
    fi
    exit 0
    ;;
  3)
    # A tool is missing or failed, which the page can't fix.
    context "Some documentation checks couldn't run, so $file wasn't fully checked. Fix any alerts below, and tell the user which checks were skipped; see docs/documentation/check-documentation.md.
$out"
    exit 0
    ;;
esac

echo "scripts/docs-lint.sh found errors in $file, or in links to it from other pages. Fix them, following docs/standards/:
$out" >&2
exit 2
