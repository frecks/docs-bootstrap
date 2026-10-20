#!/usr/bin/env bash
# Check documentation against docs/standards.
#
# Usage: scripts/docs-lint.sh [FILE...]
# With no arguments, checks every Markdown file in docs/.
#
# Runs four checks:
#   1. Placement and type: each page is in an area directory, and declares
#      its Diátaxis type in frontmatter.
#   2. Template instructions: no leftover comments from the templates.
#   3. Vale: prose style (.vale.ini).
#   4. markdownlint: Markdown syntax, and relative links that point to files
#      and headings that exist (.markdownlint-cli2.jsonc).
# Exits non-zero if any check reports an error. Vale warnings and suggestions
# are printed but don't fail the run.

set -uo pipefail

MARKDOWNLINT_VERSION=0.23.3
RELATIVE_LINKS_VERSION=5.1.4

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root" || exit 2

if [ "$#" -gt 0 ]; then
  files=()
  for f in "$@"; do
    files+=("$(realpath --relative-to="$root" "$f")")
  done
else
  mapfile -t files < <(find docs -name '*.md' -type f | sort)
fi

if [ "${#files[@]}" -eq 0 ]; then
  echo "No Markdown files to check."
  exit 0
fi

status=0

echo "== Placement and type"
# Pages live in an area directory, such as docs/api/, and declare their
# Diátaxis type in frontmatter. README.md files are indexes and have no type.
type_ok=1
for f in "${files[@]}"; do
  case "$f" in
    docs/standards/* | */README.md) continue ;;
    docs/*/*) ;;
    docs/*)
      echo "$f: put pages in an area directory, such as docs/api/. See docs/standards/structure.md."
      type_ok=0
      continue
      ;;
    *) continue ;;
  esac
  type="$(awk 'NR == 1 && $0 != "---" { exit } NR > 1 && $0 == "---" { exit } NR > 1 && sub(/^type:[[:space:]]*/, "") { print; exit }' "$f")"
  case "$type" in
    tutorial | how-to | reference | explanation) ;;
    "")
      echo "$f: missing 'type' in frontmatter. Use tutorial, how-to, reference, or explanation. See docs/standards/structure.md."
      type_ok=0
      ;;
    *)
      echo "$f: unknown type '$type'. Use tutorial, how-to, reference, or explanation."
      type_ok=0
      ;;
  esac
done
if [ "$type_ok" -eq 1 ]; then echo "OK"; else status=1; fi

echo "== Template instructions"
# Vale skips HTML comments, so look for leftover template instructions here.
template_ok=1
for f in "${files[@]}"; do
  case "$f" in docs/standards/*) continue ;; esac
  if grep -n -E '^\s*<!-- Template:' "$f" | sed "s|^|$f:|"; then
    template_ok=0
  fi
done
if [ "$template_ok" -eq 1 ]; then echo "OK"; else echo "Delete the Template comments above."; status=1; fi

echo "== Vale"
if command -v vale > /dev/null; then
  if [ ! -d .vale/styles/Google ]; then
    vale sync > /dev/null || { echo "vale sync failed"; status=1; }
  fi
  vale --no-wrap "${files[@]}" || status=1
else
  echo "SKIPPED: vale is not installed. See docs/documentation/check-documentation.md."
  status=1
fi

echo "== markdownlint"
if command -v npx > /dev/null; then
  # The config's globs are ignored when files are passed; ":" prefixes make
  # these literal paths. The relative-links rule is a separate package.
  if out="$(npx --yes --package "markdownlint-cli2@${MARKDOWNLINT_VERSION}" \
    --package "markdownlint-rule-relative-links@${RELATIVE_LINKS_VERSION}" \
    markdownlint-cli2 --no-globs "${files[@]/#/:}" 2>&1)"; then
    echo "OK"
  else
    grep -v -E '^(markdownlint-cli2 v|Finding:|Linting:|Summary:)' <<< "$out"
    status=1
  fi
else
  echo "SKIPPED: npx is not installed, so markdownlint can't run."
  status=1
fi

exit "$status"
