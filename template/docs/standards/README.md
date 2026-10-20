# Documentation standards

These files define how documentation in this repository is written, structured, and reviewed. They're the single source of truth for people and agents. `AGENTS.md` and the agent skills link here instead of restating rules, so a rule changes in one place.

## What to read when

| Task | Read |
| --- | --- |
| Deciding what kind of page to write, and where it goes | [structure.md](structure.md) |
| Writing or editing sentences | [style-guide.md](style-guide.md) |
| Starting a new page | The matching file in [templates/](templates/) |
| Reviewing a page before it's done | [review-checklist.md](review-checklist.md) |

## Foundations

- **Style:** the [Google developer documentation style guide](https://developers.google.com/style), with the changes listed in [style-guide.md](style-guide.md).
- **Structure:** [Diátaxis](https://diataxis.fr/), which gives each page one of four types: tutorial, how-to guide, reference, or explanation. Pages are grouped by area.
- **Practice:** the [Write the Docs guide](https://www.writethedocs.org/guide/), condensed into [review-checklist.md](review-checklist.md).

## Enforcement

Run the checks on any page you change:

```bash
scripts/docs-lint.sh docs/api/example.md
```

The script runs four checks, and fails on errors:

| Check | Tool | Configuration |
| --- | --- | --- |
| Page is in an area directory, and declares a Diátaxis type | `scripts/docs-lint.sh` | [structure.md](structure.md) |
| No instructions left from a template | `scripts/docs-lint.sh` | [templates/](templates/) |
| Prose style | [Vale](https://vale.sh/) | `.vale.ini`, `.vale/styles/{{STYLE}}/`, and the {{STYLE}} vocabulary |
| Markdown syntax, and relative links that point to files and headings that exist, with links that start with `/` checked from the repository root | [markdownlint](https://github.com/DavidAnson/markdownlint) | `.markdownlint-cli2.jsonc` |

In Claude Code, hooks run the same script after every edit to a file in `docs/`, and on all of `docs/` before the agent finishes if `docs/` has uncommitted changes. They send errors back to the agent to fix.

Vale doesn't check the files in `docs/standards/`, because they quote bad examples on purpose. markdownlint still checks them.

Some standards need judgment that no linter has. [review-checklist.md](review-checklist.md) covers those, and the `docs-review` skill applies it.

## Change a rule

1. Change the rule in the relevant file here, and say why in the same change.
1. If a tool can check the rule, add or change a Vale rule in `.vale/styles/{{STYLE}}/`, or a setting in `.vale.ini` or `.markdownlint-cli2.jsonc`.
1. Run `scripts/docs-lint.sh` with no arguments, and fix or accept what the change flags across existing pages.

When a reviewer flags the same problem more than twice, turn it into a Vale rule. Over time, more of the standard is enforced by tools instead of review.

## Track the Google style guide

The Google rules come from the [errata-ai/Google](https://github.com/errata-ai/Google) Vale package, pinned to a release in `.vale.ini`. To update it:

1. Read the [Google style guide's change log](https://developers.google.com/style/whats-new) and the package's [release notes](https://github.com/errata-ai/Google/releases) since the pinned version.
1. Change the version in the `Packages` URL in `.vale.ini`.
1. Run `scripts/docs-lint.sh`. It downloads the new version, because the `Packages` line changed. Every other clone of the repository downloads it on its next run too. Decide for each new alert whether to fix the pages or record a change in [style-guide.md](style-guide.md).

The Vale package is maintained separately from the guide and can lag behind it, so read the change log even when the package hasn't changed.

The Google developer documentation style guide is licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
