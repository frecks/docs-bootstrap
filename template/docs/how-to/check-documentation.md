---
type: how-to
---

# Check documentation

Run the documentation checks on a page before you finish a change, so it meets the [documentation standards](../../doc-standards/README.md). In Claude Code, hooks run these checks after each edit to a page in `docs/` or `doc-standards/`, and on every page before the agent finishes if either has uncommitted changes. A git pre-commit hook can also run them before each commit, and check every commit for secrets.

## Before you begin

On your workstation, install these tools:

- [Vale](https://vale.sh/docs/install) 3.24.0 or later, on your `PATH`.
- Node.js 22 or later, with `npm`, which installs markdownlint and its link rule the first time the checks run.
- `jq`, for the Claude Code hooks.
- [gitleaks](https://github.com/gitleaks/gitleaks) 8.30.1 or later, on your `PATH`, for the pre-commit hook's check for secrets.

To install Vale 3.24.0 to `~/.local/bin`, run:

```bash
mkdir -p ~/.local/bin
curl -sL https://github.com/errata-ai/vale/releases/download/v3.24.0/vale_3.24.0_Linux_64-bit.tar.gz | tar xz -C ~/.local/bin vale
```

To install gitleaks 8.30.1 to `~/.local/bin`, checking the download against the release's checksums first, run:

```bash
dir="$(mktemp -d)"
gh release download v8.30.1 --repo gitleaks/gitleaks --dir "$dir" --pattern 'gitleaks_8.30.1_linux_x64.tar.gz' --pattern 'gitleaks_8.30.1_checksums.txt'
(cd "$dir" && grep ' gitleaks_8.30.1_linux_x64.tar.gz$' gitleaks_8.30.1_checksums.txt | sha256sum --check)
tar xzf "$dir/gitleaks_8.30.1_linux_x64.tar.gz" -C "$dir" gitleaks
install -m 0755 "$dir/gitleaks" ~/.local/bin/gitleaks
rm -rf "$dir"
```

`sha256sum` prints `gitleaks_8.30.1_linux_x64.tar.gz: OK`.

## Check pages

From the repository root, run the script with the page's path. A path is relative to the directory you run the script in:

```bash
scripts/docs-lint.sh docs/how-to/check-documentation.md
```

To check every page in `docs/` and `doc-standards/`, run the script with no arguments:

```bash
scripts/docs-lint.sh
```

The first run downloads the pinned Google style for Vale into `.vale/styles/Google/`, and installs the markdownlint packages locked in `.markdownlint/package-lock.json` into `.markdownlint/node_modules/`. Later runs use the installed copies, and install the packages again only when the lockfile changes. To install them without running the checks, run `scripts/docs-lint.sh --install`.

## Check before each commit

To check each commit for secrets, and the pages before each commit that changes them, turn on the pre-commit hook in `.githooks/`. From the repository root, run:

```bash
scripts/docs-lint.sh --install
```

The script sets the repository's `core.hooksPath` to `.githooks`, and prints this line:

```text
Turned on the pre-commit hook in .githooks/.
```

The setting applies to every worktree of the clone. When Claude Code creates a worktree, it rewrites the setting to the absolute path of the main checkout's `.githooks`. The hook stays on, and every worktree then runs the main checkout's copy of it, which checks that worktree's pages. If `core.hooksPath` is already set to another directory, the script doesn't change it, and says so.

On every commit, the hook runs gitleaks on the staged changes, with the settings in `.gitleaks.toml`, and stops the commit if gitleaks finds what looks like a secret. Its output hides the value. If the finding is a placeholder or another value that isn't secret, add it to the allowlists in `.gitleaks.toml`.

When a commit changes a page in `docs/` or `doc-standards/`, or the configuration of the checks, the hook also checks every page, and stops the commit if a check reports an error. It checks the files on disk, so it includes changes you haven't staged.

If gitleaks, Vale, or `npm` isn't installed or can't run, or gitleaks is older than 8.25.0, the hook skips that check with a message, and doesn't stop the commit.

To confirm that the hook is on, run `git config --get core.hooksPath`. It prints `.githooks`, or the absolute path to the main checkout's `.githooks` once Claude Code has rewritten it.

## Read the results

The script runs four checks, and prints a section for each:

- **Placement and type:** the page declares its [Diátaxis type](../../doc-standards/structure.md#declare-the-type) in frontmatter, and is in the directory for it: a tutorial or how-to guide in `docs/tutorials/` or `docs/how-to/`, and any other page in a system directory.
- **Template instructions:** no `<!-- Template: -->` comments remain from a [template](../../doc-standards/templates/).
- **Vale:** the prose follows the [style guide](../../doc-standards/style-guide.md). Each alert names its rule, such as `Google.We`.
- **markdownlint:** the Markdown syntax follows `.markdownlint-cli2.jsonc`, and each relative link or image points to a file that exists. An anchor, such as `style-guide.md#dates`, must match a heading in that file. A link that starts with `/` is checked from the repository root. With `--links-as-warnings`, which the Claude Code edit hook uses, a link to a file or heading that doesn't exist is a warning instead of an error, because a later edit may add it. The Stop hook and the pre-commit hook still stop on it.

Fix every error. Fix warnings and suggestions unless you've decided they don't apply.

If a check prints `SKIPPED`, its tool isn't installed, isn't on `PATH`, or couldn't run, such as when `npm` can't download markdownlint. Install the tool or fix the network problem, then run the script again. The script exits with status 1 when a check reports an error, 2 when a page you named doesn't exist, and 3 when no check reports an error but a check was skipped.

## Verify

Run the script again. A clean run ends with this output, and the script exits with status 0:

```text
== markdownlint
OK
```

## Related

- [Documentation standards](../../doc-standards/README.md)
- [Review checklist](../../doc-standards/review-checklist.md)
- [Why the documentation system works this way](../../doc-standards/rationale.md)
