# Why the documentation system works this way

This page explains why Docs bootstrap documentation uses Diátaxis for structure, a style guide based on Google's, Vale and markdownlint for checks, and Claude Code hooks that run those checks for agents. Read it before you change any part of the system, so you know what each part protects against. The rules themselves are in the [documentation standards](README.md), and this page doesn't repeat them.

## Context

The documentation has to work under these conditions:

- **Readers:** technical readers who know the common tools and technologies the project uses, but not this project or why it's built and set up the way it is. That includes a future self and agents. See [Audience](style-guide.md#audience).
- **Writers:** people and agents both write pages, so the rules need one written source that both can read and check against.
- **Change:** the pages describe code and systems that change often, so pages go stale unless they change with them.
- **Sharing:** the repository may end up shared or published, and a secret in its history is hard to remove.

## Decision

We set up the system on 2026-10-19. It has six parts, and each one answers a different question about a page or about the system itself.

### Diátaxis decides what a page is for

Every page serves one of four reader needs: learning by doing, completing a task, looking up a fact, or understanding a decision. [Diátaxis](https://diataxis.fr/) names these needs as four types: tutorial, how-to guide, reference, and explanation. A page that tries to serve two needs serves neither well, so a page that drifts gets split. See [Documentation structure](structure.md).

Each page declares its type in frontmatter, and the tree splits the way readers do. Tutorials and how-to guides are in `docs/tutorials/` and `docs/how-to/`, because a reader with a task looks for the task, not for the system that does it. Reference and explanation pages are in a directory for the system they describe, as in `docs/network/`, because a reader looks up a fact or a reason by system. The type stays visible on each page and in the index. See [Organize the pages](structure.md#organize-the-pages).

The split follows a survey of the docs of 39 projects that use Vale. Projects under about 200 pages mostly put the types first, and large product docs, such as Docker's and dbt's, put tasks first by type and reference by product. The standards are in `doc-standards/`, outside `docs/`, so `docs/` holds only pages about the project.

### The style guide starts from Google's and records each difference

Pages follow the [Google developer documentation style guide](https://developers.google.com/style), and [style-guide.md](style-guide.md) lists only where this project differs or adds to it. Google covers the general questions of voice, headings, procedures, and code, so the local file stays short.

Google writes for a broad, global audience. This project's pages have a small, known audience, so some Google rules don't fit. Examples include ISO 8601 dates such as 2026-10-19, real values instead of placeholders where the reader must copy them, and a regular space between a number and its unit. Each difference states its reason, so that someone can revisit it when needs change, and names the check that enforces it.

The [review checklist](review-checklist.md) draws on Diátaxis, the [Write the Docs guide](https://www.writethedocs.org/guide/), and the style guide.

### Vale checks prose

[Vale](https://vale.sh/) turns style rules into checks that run on every page, no matter who wrote it. It runs three styles:

- **Google:** the [errata-ai/Google](https://github.com/errata-ai/Google) package, pinned to a release in `.vale.ini`. Pinning means a new package release can't change the results of a check without a deliberate update. `vale sync` downloads the package into `.vale/styles/Google/`, which `.gitignore` excludes.
- **DocsBootstrap:** local rules in `.vale/styles/DocsBootstrap/` for what Google's package doesn't check, such as secrets, draft markers, and minimizing words.
- **Vale:** Vale's built-in rules, used with the DocsBootstrap vocabulary in `.vale/styles/config/vocabularies/DocsBootstrap/` to enforce the capitalization of product names.

Four settings in `.vale.ini` change what the styles check:

- `Vale.Spelling = NO` turns off the spell checker. Hostnames, package names, and tool names make it report too many false errors. Product names are still checked through the vocabulary.
- `BasedOnStyles =` is empty for `doc-standards/`, so Vale doesn't check the standards. They quote bad examples on purpose, and the templates hold placeholders. This page is the exception, and Vale checks it like any other page.
- `Google.OxfordComma = suggestion` lowers the serial comma rule, because it also flags commas that aren't in a list. Review checks serial commas instead.
- `Google.We = suggestion` lowers the rule against first-person plural words, such as `we` and `our`. Google allows them for the people who maintain a project, so the rule points out each one for review instead of warning.

### A Markdown linter checks syntax

[markdownlint](https://github.com/DavidAnson/markdownlint) checks Markdown syntax, which Vale doesn't. `.markdownlint-cli2.jsonc` enables the default rules, and picks one style wherever Markdown offers a choice:

- `#` for headings.
- `-` for bullets.
- `1.` for every item in a numbered list.
- Fenced code blocks.
- `*` for emphasis.

It turns off the line-length rule, so each paragraph is one line and editors wrap it. It allows no inline HTML except `<kbd>`.

markdownlint also checks links, through the [markdownlint-rule-relative-links](https://github.com/theoludwig/markdownlint-rule-relative-links) rule: each relative link must point to a file that exists, and an anchor must match a heading in that file. A separate link checker would add a dependency, such as Python, and wouldn't check anchors in other pages.

`.markdownlint/package-lock.json` pins markdownlint-cli2, the link rule, and every package they depend on, each with an integrity hash. `scripts/docs-lint.sh` installs them once with `npm ci --ignore-scripts`, which checks each hash and runs no package code while installing, and installs them again only when the lockfile changes. Each check then runs the installed copy. Running packages through `npx` instead would resolve them from the registry on every run, with only the top-level versions pinned, and would add about half a second to each check.

### One script runs every check

`scripts/docs-lint.sh` runs four checks in order: page placement and type, leftover template comments, Vale, and markdownlint. The first check confirms that each page is in a directory under `docs/`, that its frontmatter declares one of the four types, and that tutorials and how-to guides are in their own directories and no other type is. Vale skips HTML comments, so the script checks for `<!-- Template: -->` comments itself. The script fails on any error, and prints Vale warnings and suggestions without failing.

Having one script means people, agents, and the hooks all run the same checks. To run it, see [Check documentation](../docs/how-to/check-documentation.md).

Two hooks in `.claude/settings.json` run the script for agents in Claude Code:

- **After each edit:** a `PostToolUse` hook checks a Markdown file in `docs/` or `doc-standards/` after each `Edit`, `MultiEdit`, or `Write`. It also runs markdownlint on the pages that mention the file's name, so a link that the edit broke elsewhere, such as an anchor to a renamed heading, is caught too. Checking only those pages keeps each edit fast however large `docs/` grows.
- **Before the agent finishes:** a `Stop` hook checks every page in `docs/` and `doc-standards/`, if any page there has uncommitted changes. This catches what the edit hook can't see, such as a page that a command moved or deleted. It blocks the agent once, so it fixes the errors, and then lets it finish.

Errors go back to the agent to fix before it moves on. Warnings and suggestions go to the agent as context, without blocking. If a check can't run, such as when Vale isn't installed or npm can't download markdownlint, the edit hook asks the agent to tell the user, the Stop hook shows the user a message, and neither blocks. The hooks catch problems while the agent is still working, instead of in review.

### A bootstrap tool installs and upgrades the system

The docs-bootstrap tool installed these files, and `.docs-bootstrap.json` records the version and a hash of each file it manages. Running the tool again upgrades every file that no one here has edited, and reports the files that someone has. Edits made here always win, so this project can adapt its standards and still take improvements to the shared ones.

## Alternatives considered

We considered and rejected these alternatives:

- **A directory for each system, holding every type:** it keeps each how-to guide next to the reference it depends on, but a reader with a task has to know which system does it first. The projects in the survey that organize this way, such as [Grafana](https://github.com/grafana/grafana/tree/main/docs/sources) and [Tigera](https://github.com/tigera/docs), have large, reference-heavy docs.
- **A directory for each of the four types:** rejected because it splits a system's facts and reasons across `reference/` and `explanation/`, which readers look up by system. It's the simplest rule, and small projects such as [LXD](https://github.com/canonical/lxd/tree/main/doc) and [nix.dev](https://github.com/nixos/nix.dev/tree/master/source) use it, so it's the fallback if the blend stops working.
- **Google's placeholder values, such as `example.com`, for every value:** rejected because a reader who needs to copy a real value can't copy a placeholder. See [Real values and placeholders](style-guide.md#real-values-and-placeholders).
- **Google's date format, such as "October 19, 2026":** rejected because ISO 8601 dates sort correctly and match log timestamps. See [Dates](style-guide.md#dates).
- **A nonbreaking space between a number and its unit:** rejected because it's hard to type in Markdown and doesn't change how pages read. See [Units](style-guide.md#units).

## Consequences

The system makes these things easier:

- A reader with a task finds it in one directory, a reader looking up a system finds its facts and reasons in one directory, and each page says what kind of help it gives.
- A rule changes in one place, `doc-standards/`, and `AGENTS.md` and the agent skills link there. The `docs-write` skill repeats only a short list of the rules agents miss most.
- Agents fix most style and syntax problems themselves, before a person reviews the page.

It also has costs and limits:

- A how-to guide is in a different directory from the reference it depends on, so the links between them, under **Related**, have to be kept up.
- Some standards need judgment that no linter has, such as keeping one type per page, naming the host before a procedure, or giving the reason for a design choice. Only [review](review-checklist.md) catches those.
- The Google Vale package has its own maintainers, separate from the guide, and can lag behind it. Updating the package is a manual task. See [Track the Google style guide](README.md#track-the-google-style-guide).
- The checks need Vale, Node.js 22 or later with npm, and jq on the workstation. The first check in a new clone needs the network, to install Vale's Google style and the markdownlint packages. When a check can't run, the hooks don't block, so an agent can finish without that check.
- Spell checking is off. Vale still flags product names with the wrong capitalization, and a few British spellings, such as `colour` and `organise`, but no other misspellings.

Revisit a rule when reviewers keep flagging the same problem. The standards say to turn a problem flagged more than twice into a Vale rule, so that over time tools enforce more of the standard than review does.

## Related

- [Documentation standards](README.md)
- [Check documentation](../docs/how-to/check-documentation.md)
