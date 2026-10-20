#!/usr/bin/env bash
# Stop hook: before Claude finishes, check every page in docs/ if any changed.
# The edit hook checks each page as it's edited, and the pages that link to
# it, but it doesn't see pages that a command moved or deleted. Errors are sent
# back to Claude with exit code 2, once per stop, so it fixes them.

set -uo pipefail

# Without jq, the hook can't tell a second stop from the first, so it could
# block every stop. Skip the check instead.
command -v jq > /dev/null || exit 0

root="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
input="$(cat)"

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
