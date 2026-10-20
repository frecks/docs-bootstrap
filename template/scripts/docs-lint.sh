#!/usr/bin/env bash
# Check documentation against docs/standards.
#
# Usage: scripts/docs-lint.sh [--with-referrers] [FILE...]
#        scripts/docs-lint.sh --install
# With no FILE, checks every Markdown file in docs/. FILE paths are relative to
# the current directory.
#
# --install installs the tools the checks use, and exits: Vale's Google style,
# and the markdownlint packages locked in .markdownlint/package-lock.json. A
# check installs them itself when they're missing or out of date.
#
# --with-referrers also runs markdownlint on the pages in docs/ that mention a
# FILE's name, so a link to a FILE that an edit broke, such as an anchor to a
# renamed heading, is caught. It costs about the same however large docs/ is.
#
# Runs four checks:
#   1. Placement and type: each page is in an area directory, and declares
#      its Diátaxis type in frontmatter.
#   2. Template instructions: no leftover comments from the templates.
#   3. Vale: prose style (.vale.ini).
#   4. markdownlint: Markdown syntax, and relative links that point to files
#      and headings that exist (.markdownlint-cli2.jsonc).
# Exits 1 if any check reports an error, 2 if a FILE doesn't exist, and 3 if
# no check reports an error but a check couldn't run, such as when a tool
# isn't installed. Vale warnings and suggestions are printed but don't fail
# the run.

set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
markdownlint=.markdownlint/node_modules/.bin/markdownlint-cli2

# install_vale_styles downloads the Google style pinned in .vale.ini, if it's
# missing. It prints SKIPPED and returns 1 when Vale can't run.
install_vale_styles() {
  if ! command -v vale > /dev/null; then
    echo "SKIPPED: vale is not installed. See docs/documentation/check-documentation.md."
    return 1
  fi
  if [ ! -d .vale/styles/Google ] && ! vale sync > /dev/null; then
    echo "SKIPPED: vale sync failed, so Vale can't run."
    return 1
  fi
}

# install_markdownlint installs the packages locked in
# .markdownlint/package-lock.json, when they're missing or the lockfile has
# changed since they were installed. npm ci checks each package against its
# integrity hash, and --ignore-scripts runs no package code while installing.
# It prints SKIPPED and returns 1 when markdownlint can't run.
install_markdownlint() {
  local stamp=.markdownlint/node_modules/.docs-lint-lock
  if [ -x "$markdownlint" ] && cmp -s .markdownlint/package-lock.json "$stamp"; then
    return 0
  fi
  if ! command -v npm > /dev/null; then
    echo "SKIPPED: npm is not installed, so markdownlint can't be installed. See docs/documentation/check-documentation.md."
    return 1
  fi
  local out
  if ! out="$(npm ci --prefix .markdownlint --ignore-scripts --no-audit --no-fund --loglevel=error 2>&1)"; then
    echo "$out"
    echo "SKIPPED: npm ci failed, so markdownlint can't run."
    return 1
  fi
  cp .markdownlint/package-lock.json "$stamp"
}

if [ "${1:-}" = "--install" ]; then
  cd "$root" || exit 2
  ok=0
  install_vale_styles || ok=3
  install_markdownlint || ok=3
  exit "$ok"
fi

with_referrers=0
if [ "${1:-}" = "--with-referrers" ]; then
  with_referrers=1
  shift
fi

# Resolve FILE arguments from the caller's directory, before moving to root.
files=()
for f in "$@"; do
  if [ ! -f "$f" ]; then
    echo "$f: no such file." >&2
    exit 2
  fi
  files+=("$(realpath --relative-to="$root" -- "$f")")
done

cd "$root" || exit 2

if [ "$#" -eq 0 ]; then
  mapfile -t files < <(find docs -name '*.md' -type f | sort)
fi

if [ "${#files[@]}" -eq 0 ]; then
  echo "No Markdown files to check."
  exit 0
fi

# markdownlint also checks the links in pages that mention a FILE by name.
lint_files=("${files[@]}")
if [ "$with_referrers" -eq 1 ]; then
  for f in "${files[@]}"; do
    while IFS= read -r ref; do
      lint_files+=("$ref")
    done < <(grep -rlF --include='*.md' -- "$(basename "$f")" docs | grep -vxF -- "$f")
  done
  mapfile -t lint_files < <(printf '%s\n' "${lint_files[@]}" | sort -u)
fi

status=0
skipped=0

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
if ! install_vale_styles; then
  skipped=1
else
  # Vale exits 1 when it reports errors, and 2 when it can't run.
  vale --no-wrap "${files[@]}"
  case $? in
    0) ;;
    1) status=1 ;;
    *)
      echo "SKIPPED: Vale couldn't run."
      skipped=1
      ;;
  esac
fi

echo "== markdownlint"
if ! install_markdownlint; then
  skipped=1
# The config's globs are ignored when files are passed; ":" prefixes make
# these literal paths.
elif out="$("$markdownlint" --no-globs "${lint_files[@]/#/:}" 2>&1)"; then
  echo "OK"
elif grep -q '^Summary:' <<< "$out"; then
  grep -v -E '^(markdownlint-cli2 v|Finding:|Linting:|Summary:)' <<< "$out"
  status=1
else
  # No summary means markdownlint didn't run.
  echo "$out"
  echo "SKIPPED: markdownlint couldn't run."
  skipped=1
fi

if [ "$status" -ne 0 ]; then
  exit 1
fi
if [ "$skipped" -ne 0 ]; then
  exit 3
fi
exit 0
