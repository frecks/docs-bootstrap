#!/usr/bin/env bash
# Check documentation against doc-standards.
#
# Usage: scripts/docs-lint.sh [--with-referrers] [FILE...]
#        scripts/docs-lint.sh --install
# With no FILE, checks every Markdown file in docs/ and doc-standards/. FILE
# paths are relative to the current directory.
#
# --install installs the tools the checks use, and exits: Vale's Google style,
# and the markdownlint packages locked in .markdownlint/package-lock.json. A
# check installs them itself when they're missing or out of date.
#
# --with-referrers also runs markdownlint on the pages in docs/ and
# doc-standards/ that mention a FILE's name, so a link to a FILE that an edit
# broke, such as an anchor to a renamed heading, is caught. It costs about the
# same however large docs/ is.
#
# Runs four checks:
#   1. Placement and type: each page is in a directory under docs/, and
#      declares its Diátaxis type in frontmatter. Tutorials and how-to guides
#      are in docs/tutorials/ and docs/how-to/, and no other type is.
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

# install_vale_styles downloads the styles that .vale.ini's Packages line pins,
# when they're missing or the line has changed since they were downloaded. If
# the download fails, it keeps the styles it had. It prints SKIPPED and
# returns 1 when Vale can't run.
install_vale_styles() {
  if ! command -v vale > /dev/null; then
    echo "SKIPPED: vale is not installed. See docs/how-to/check-documentation.md."
    return 1
  fi
  local styles=.vale/styles/Google
  local stamp="$styles/.docs-lint-packages"
  local packages
  packages="$(grep -E '^Packages[[:space:]]*=' .vale.ini)"
  # A run that stopped during a sync can leave only the backup.
  if [ ! -d "$styles" ] && [ -d "$styles.old" ]; then
    mv "$styles.old" "$styles"
  fi
  if [ -d "$styles" ] && [ "$(cat "$stamp" 2> /dev/null)" = "$packages" ]; then
    return 0
  fi
  # Sync into an empty directory, so rules removed upstream don't linger.
  rm -rf "$styles.old"
  if [ -d "$styles" ]; then
    mv "$styles" "$styles.old"
  fi
  if vale sync > /dev/null; then
    rm -rf "$styles.old"
    printf '%s\n' "$packages" > "$stamp"
    return 0
  fi
  rm -rf "$styles"
  if [ -d "$styles.old" ]; then
    mv "$styles.old" "$styles"
    echo "vale sync failed, so Vale is using the styles from before .vale.ini changed."
    return 0
  fi
  echo "SKIPPED: vale sync failed, so Vale can't run."
  return 1
}

# install_markdownlint installs the packages locked in
# .markdownlint/package-lock.json, when they're missing or the lockfile has
# changed since they were installed. npm ci checks each package against its
# integrity hash, and --ignore-scripts runs no package code while installing.
# A failed install keeps the packages that were there. It prints SKIPPED and
# returns 1 when markdownlint can't run.
install_markdownlint() {
  local stamp=.markdownlint/node_modules/.docs-lint-lock
  if [ -x "$markdownlint" ] && cmp -s .markdownlint/package-lock.json "$stamp"; then
    return 0
  fi
  if ! command -v npm > /dev/null; then
    echo "SKIPPED: npm is not installed, so markdownlint can't be installed. See docs/how-to/check-documentation.md."
    return 1
  fi
  # Install next to the current packages, and swap them in once npm ci works.
  local next=.markdownlint/.next out
  rm -rf "$next"
  mkdir -p "$next"
  cp .markdownlint/package.json .markdownlint/package-lock.json "$next/"
  if ! out="$(npm ci --prefix "$next" --ignore-scripts --no-audit --no-fund --loglevel=error 2>&1)"; then
    rm -rf "$next"
    echo "$out"
    if [ -x "$markdownlint" ]; then
      echo "npm ci failed, so markdownlint is using the packages from before the lockfile changed."
      return 0
    fi
    echo "SKIPPED: npm ci failed, so markdownlint can't run."
    return 1
  fi
  rm -rf .markdownlint/node_modules
  mv "$next/node_modules" .markdownlint/node_modules
  rm -rf "$next"
  cp .markdownlint/package-lock.json "$stamp"
}

# referrer_errors reads markdownlint's error lines, and keeps those for a FILE,
# and those from other pages whose message mentions a FILE's name. A name
# counts only as a whole name, so x.md doesn't match index.md. markdownlint
# percent-encodes links in its messages, so they're decoded first. LC_ALL=C
# decodes them to bytes, which match UTF-8 names.
referrer_errors() {
  FILES="$(printf '%s\n' "${files[@]}")" NAMES="$(printf '%s\n' "${names[@]}")" LC_ALL=C awk '
    function hex(h,   v, i) {
      v = 0
      for (i = 1; i <= 2; i++) v = v * 16 + index("0123456789ABCDEF", toupper(substr(h, i, 1))) - 1
      return v
    }
    function decode(s,   out, i, c) {
      out = ""
      for (i = 1; i <= length(s); i++) {
        c = substr(s, i, 1)
        if (c == "%" && substr(s, i + 1, 2) ~ /^[0-9A-Fa-f][0-9A-Fa-f]$/) {
          out = out sprintf("%c", hex(substr(s, i + 1, 2)))
          i += 2
        } else {
          out = out c
        }
      }
      return out
    }
    function mentions(line, name,   p, before) {
      while ((p = index(line, name)) > 0) {
        before = p > 1 ? substr(line, p - 1, 1) : ""
        if (before !~ /[A-Za-z0-9_.-]/) return 1
        line = substr(line, p + 1)
      }
      return 0
    }
    BEGIN {
      n = split(ENVIRON["FILES"], f, "\n")
      for (i = 1; i <= n; i++) own[f[i]] = 1
      m = split(ENVIRON["NAMES"], name, "\n")
    }
    {
      # Each line is PATH:LINE[:COLUMN] MESSAGE.
      path = $0
      sub(/:[0-9]+(:[0-9]+)? .*/, "", path)
      if (path in own) { print; next }
      message = substr($0, length(path) + 1)
      sub(/^:[0-9]+(:[0-9]+)? /, "", message)
      message = decode(message)
      for (i = 1; i <= m; i++) {
        if (name[i] != "" && mentions(message, name[i])) { print; next }
      }
    }'
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
  mapfile -t files < <(find docs doc-standards -name '*.md' -type f | sort)
fi

if [ "${#files[@]}" -eq 0 ]; then
  echo "No Markdown files to check."
  exit 0
fi

# markdownlint also checks the links in pages that mention a FILE by name, as
# it is or with spaces encoded as %20.
lint_files=("${files[@]}")
names=()
if [ "$with_referrers" -eq 1 ]; then
  for f in "${files[@]}"; do
    name="$(basename "$f")"
    names+=("$name" "${name// /%20}")
    while IFS= read -r ref; do
      lint_files+=("$ref")
    done < <(grep -rlF --include='*.md' -e "$name" -e "${name// /%20}" docs doc-standards | grep -vxF -- "$f")
  done
  mapfile -t lint_files < <(printf '%s\n' "${lint_files[@]}" | sort -u)
fi

status=0
skipped=0

echo "== Placement and type"
# Pages live in a directory under docs/, and declare their Diátaxis type in
# frontmatter. Tutorials and how-to guides live in docs/tutorials/ and
# docs/how-to/, and other types live in a directory for their system, such as
# docs/network/.
# README.md files are indexes and have no type.
# The files in doc-standards/ aren't pages, so they aren't checked.
type_ok=1
for f in "${files[@]}"; do
  case "$f" in
    */README.md) continue ;;
    docs/*/*) ;;
    docs/*)
      echo "$f: put pages in a directory, such as docs/how-to/ or docs/network/. See doc-standards/structure.md."
      type_ok=0
      continue
      ;;
    *) continue ;;
  esac
  # Allow a byte order mark, CRLF line endings, a quoted value, as in
  # type: "how-to", and a comment after the value.
  type="$(awk -v quotes="[\"']" '
    NR == 1 { sub(/^\357\273\277/, "") }
    { sub(/\r$/, "") }
    NR == 1 && $0 != "---" { exit }
    NR > 1 && $0 == "---" { exit }
    NR > 1 && sub(/^type:[[:space:]]*/, "") {
      sub(/[[:space:]]+#.*$/, "")
      sub(/[[:space:]]+$/, "")
      gsub("^" quotes "|" quotes "$", "")
      print
      exit
    }' "$f")"
  case "$type" in
    tutorial | how-to | reference | explanation) ;;
    "")
      echo "$f: missing 'type' in frontmatter. Use tutorial, how-to, reference, or explanation. See doc-standards/structure.md."
      type_ok=0
      continue
      ;;
    *)
      echo "$f: unknown type '$type'. Use tutorial, how-to, reference, or explanation."
      type_ok=0
      continue
      ;;
  esac
  case "$f:$type" in
    docs/how-to/*:how-to | docs/tutorials/*:tutorial) ;;
    docs/how-to/* | docs/tutorials/*)
      echo "$f: a $type page doesn't belong in ${f%/*}/. Put it in the directory of the system it describes. See doc-standards/structure.md."
      type_ok=0
      ;;
    *:how-to)
      echo "$f: put how-to guides in docs/how-to/. See doc-standards/structure.md."
      type_ok=0
      ;;
    *:tutorial)
      echo "$f: put tutorials in docs/tutorials/. See doc-standards/structure.md."
      type_ok=0
      ;;
  esac
done
if [ "$type_ok" -eq 1 ]; then echo "OK"; else status=1; fi

echo "== Template instructions"
# Vale skips HTML comments, so look for leftover template instructions here.
template_ok=1
for f in "${files[@]}"; do
  case "$f" in doc-standards/*) continue ;; esac
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
  errors="$(grep -v -E '^(markdownlint-cli2 v|Finding:|Linting:|Summary:)' <<< "$out")"
  # In pages that link to a FILE, only errors about the FILE count, so an
  # unrelated error there doesn't block an edit to the FILE.
  if [ "$with_referrers" -eq 1 ]; then
    errors="$(referrer_errors <<< "$errors")"
  fi
  if [ -n "$errors" ]; then
    echo "$errors"
    status=1
  else
    echo "OK"
  fi
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
