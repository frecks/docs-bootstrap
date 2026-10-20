#!/usr/bin/env bash
# Stop hook: before Claude finishes, check every page in docs/ if any changed.
# The edit hook checks each page as it's edited, and the pages that link to
# it, but it doesn't see pages that a command moved or deleted. Errors are sent
# back to Claude with exit code 2, once per stop, so it fixes them.

set -uo pipefail

# Without jq, the hook can't tell a second stop from the first, so it could
# block every stop. Skip the check instead.
command -v jq > /dev/null || exit 0

input="$(cat)"

project="$(realpath -- "${CLAUDE_PROJECT_DIR:-$(dirname "${BASH_SOURCE[0]}")/../..}")" || exit 0

# common_dir prints the git directory that a checkout's worktrees share.
common_dir() {
  git -C "$1" rev-parse --path-format=absolute --git-common-dir 2> /dev/null
}

# Check the checkout the session works in, if it's a worktree of this project:
# $CLAUDE_PROJECT_DIR can name the main checkout while the session works in a
# worktree. Otherwise, check the project.
root="$project"
cwd="$(jq -r '.cwd // empty' <<< "$input")"
if [ -n "$cwd" ] && top="$(git -C "$cwd" rev-parse --show-toplevel 2> /dev/null)" &&
  [ "$(common_dir "$top")" = "$(common_dir "$project")" ]; then
  root="$top"
fi
[ -x "$root/scripts/docs-lint.sh" ] || exit 0

# After a blocked stop, let Claude finish rather than block again.
if [ "$(jq -r '.stop_hook_active // false' <<< "$input")" = "true" ]; then
  exit 0
fi

# Skip when docs/ has no uncommitted changes. Outside a git repository, there's
# no way to tell, so always check.
if git -C "$root" rev-parse --is-inside-work-tree > /dev/null 2>&1 &&
  [ -z "$(git -C "$root" status --porcelain -- docs)" ]; then
  exit 0
fi

out="$("$root/scripts/docs-lint.sh" 2>&1)"
case $? in
  1) ;;
  3)
    # A tool is missing or failed: tell the user, without blocking.
    jq -n '{systemMessage: "Some documentation checks couldn'"'"'t run before Claude finished. Run scripts/docs-lint.sh to see which, and see docs/documentation/check-documentation.md."}'
    exit 0
    ;;
  *) exit 0 ;;
esac

echo "scripts/docs-lint.sh found errors in docs/. Fix them before you finish, following docs/standards/:
$out" >&2
exit 2
