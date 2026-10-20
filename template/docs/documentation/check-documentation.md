---
type: how-to
---

# Check documentation

Run the documentation checks on a page before you finish a change, so it meets the [documentation standards](../standards/README.md). Agents in Claude Code run these checks automatically after each edit.

## Before you begin

On your workstation, install these tools:

- [Vale](https://vale.sh/docs/install) 3.24.0 or later, on your `PATH`.
- Node.js 22 or later, with `npm`, which installs markdownlint and its link rule the first time the checks run.
- `jq`, for the Claude Code hooks.

To install Vale 3.24.0 to `~/.local/bin`, run:

```bash
curl -sL https://github.com/errata-ai/vale/releases/download/v3.24.0/vale_3.24.0_Linux_64-bit.tar.gz | tar xz -C ~/.local/bin vale
```

## Check pages

From the repository root, run:

```bash
scripts/docs-lint.sh docs/documentation/check-documentation.md
```

To check every page in `docs/`, run the script with no arguments:

```bash
scripts/docs-lint.sh
```

The first run downloads the pinned Google style for Vale into `.vale/styles/Google/`, and installs the markdownlint packages locked in `.markdownlint/package-lock.json` into `.markdownlint/node_modules/`. Later runs use the installed copies, and install the packages again only when the lockfile changes. To install them without running the checks, run `scripts/docs-lint.sh --install`.

## Read the results

The script runs four checks, and prints a section for each:

- **Placement and type:** the page is in an area directory, and declares its [Diátaxis type](../standards/structure.md#declare-the-type) in frontmatter.
- **Template instructions:** no `<!-- Template: -->` comments remain from a [template](../standards/templates/).
- **Vale:** the prose follows the [style guide](../standards/style-guide.md). Each alert names its rule, such as `Google.Passive`.
- **markdownlint:** the Markdown syntax follows `.markdownlint-cli2.jsonc`, and each relative link or image points to a file that exists. An anchor, such as `style-guide.md#dates`, must match a heading in that file.

Fix every error. Fix warnings and suggestions unless you've decided they don't apply.

If a check prints `SKIPPED`, its tool isn't installed, or couldn't run, such as when `npm` can't download markdownlint. Install the tool or fix the network problem, then run the script again. When no check reports an error but a check was skipped, the script exits with status 3.

## Verify

Run the script again. A clean run ends with this output, and the script exits with status 0:

```text
== markdownlint
OK
```

## Related

- [Documentation standards](../standards/README.md)
- [Review checklist](../standards/review-checklist.md)
- [Why the documentation system works this way](why-the-documentation-system-works-this-way.md)
