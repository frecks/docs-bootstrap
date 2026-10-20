---
type: explanation
---

# Why the documentation system works this way

This page explains why Docs bootstrap documentation uses Diátaxis for structure, a style guide based on Google's, Vale and markdownlint for checks, and Claude Code hooks that run those checks for agents. Read it before you change any part of the system, so you know what each part protects against. The rules themselves are in the [documentation standards](../standards/README.md), and this page doesn't repeat them.

## Context

The documentation has to work under these conditions:

- **Readers:** technical readers who know the common tools and technologies the project uses, but not this project or why it's built and set up the way it is. That includes a future self and agents. See [Audience](../standards/style-guide.md#audience).
- **Writers:** people and agents both write pages, so the rules need one written source that both can read and check against.
- **Change:** the pages describe code and systems that change often, so pages go stale unless they change with them.
- **Sharing:** the repository may end up shared or published, and a secret in its history is hard to remove.

## Decision

We set up the system on 2026-10-19. It has six parts, and each one answers a different question about a page or about the system itself.

### Diátaxis decides what a page is for

Every page serves one of four reader needs: learning by doing, completing a task, looking up a fact, or understanding a decision. [Diátaxis](https://diataxis.fr/) names these needs as four types: tutorial, how-to guide, reference, and explanation. A page that tries to serve two needs serves neither well, so a page that drifts gets split. See [Documentation structure](../standards/structure.md).

Each page declares its type in frontmatter, and directories group pages by the area they describe, such as a component or a system, as in `docs/api/`. A reader looks for help by area, so everything about one area is in one place. The type stays visible on each page and in the index.

### The style guide starts from Google's and records each difference

Pages follow the [Google developer documentation style guide](https://developers.google.com/style), and [style-guide.md](../standards/style-guide.md) lists only where this project differs or adds to it. Google covers the general questions of voice, headings, procedures, and code, so the local file stays short.

Google writes for a broad, global audience. This project's pages have a small, known audience, so some Google rules don't fit. Examples include ISO 8601 dates such as 2026-10-19, real values instead of placeholders where the reader must copy them, and a regular space between a number and its unit. Each difference states its reason, so that someone can revisit it when needs change, and names the check that enforces it.

The [review checklist](../standards/review-checklist.md) draws on Diátaxis, the [Write the Docs guide](https://www.writethedocs.org/guide/), and the style guide.

### Vale checks prose

[Vale](https://vale.sh/) turns style rules into checks that run on every page, no matter who wrote it. It runs three styles:

- **Google:** the [errata-ai/Google](https://github.com/errata-ai/Google) package, pinned to a release in `.vale.ini`. Pinning means a new package release can't change the results of a check without a deliberate update. `vale sync` downloads the package into `.vale/styles/Google/`, which `.gitignore` excludes.
- **DocsBootstrap:** local rules in `.vale/styles/DocsBootstrap/` for what Google's package doesn't check, such as secrets, draft markers, and minimizing words.
- **Vale:** Vale's built-in rules, used with the DocsBootstrap vocabulary in `.vale/styles/config/vocabularies/DocsBootstrap/` to enforce the capitalization of product names.

Four settings in `.vale.ini` change what the styles check:

- `Vale.Spelling = NO` turns off the spell checker. Hostnames, package names, and tool names make it report too many false errors. Product names are still checked through the vocabulary.
- `BasedOnStyles =` is empty for `docs/standards/`, so Vale doesn't check the standards. They quote bad examples on purpose, and the templates hold placeholders.
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

`scripts/docs-lint.sh` runs four checks in order: page placement and type, leftover template comments, Vale, and markdownlint. The first check confirms that each page is in an area directory and that its frontmatter declares one of the four types. Vale skips HTML comments, so the script checks for `<!-- Template: -->` comments itself. The script fails on any error, and prints Vale warnings and suggestions without failing.

Having one script means people, agents, and the hooks all run the same checks. To run it, see [Check documentation](check-documentation.md).

Two hooks in `.claude/settings.json` run the script for agents in Claude Code:

- **After each edit:** a `PostToolUse` hook checks a Markdown file in `docs/` after each `Edit`, `MultiEdit`, or `Write`. It also runs markdownlint on the pages that mention the file's name, so a link that the edit broke elsewhere, such as an anchor to a renamed heading, is caught too. Checking only those pages keeps each edit fast however large `docs/` grows.
- **Before the agent finishes:** a `Stop` hook checks every page in `docs/`, if any page there has uncommitted changes. This catches what the edit hook can't see, such as a page that a command moved or deleted. It blocks the agent once, so it fixes the errors, and then lets it finish.

Errors go back to the agent to fix before it moves on. Warnings and suggestions go to the agent as context, without blocking. If a check can't run, such as when Vale isn't installed or npm can't download markdownlint, the hooks ask the agent to tell the user, and don't block. The hooks catch problems while the agent is still working, instead of in review.

### A bootstrap tool installs and upgrades the system

The docs-bootstrap tool installed these files, and `.docs-bootstrap.json` records the version and a hash of each file it manages. Running the tool again upgrades every file that no one here has edited, and reports the files that someone has. Edits made here always win, so this project can adapt its standards and still take improvements to the shared ones.

## Alternatives considered

We considered and rejected these alternatives:

- **A directory for each Diátaxis type, such as `how-to/`:** it splits each area's pages across four directories, and pushes you to create empty structure before there's content, which [Diátaxis advises against](https://diataxis.fr/how-to-use-diataxis/). The [Discord API docs](https://github.com/discord/discord-api-docs/tree/main/developers) show the alternative: group by subject, and keep the types within each subject.
- **Google's placeholder values, such as `example.com`, for every value:** rejected because a reader who needs to copy a real value can't copy a placeholder. See [Real values and placeholders](../standards/style-guide.md#real-values-and-placeholders).
- **Google's date format, such as "October 19, 2026":** rejected because ISO 8601 dates sort correctly and match log timestamps. See [Dates](../standards/style-guide.md#dates).
- **A nonbreaking space between a number and its unit:** rejected because it's hard to type in Markdown and doesn't change how pages read. See [Units](../standards/style-guide.md#units).

## Consequences

The system makes these things easier:

- A reader finds everything about one area in one directory, and each page says what kind of help it gives.
- A rule changes in one place, `docs/standards/`, and `AGENTS.md` and the agent skills link there. The `docs-write` skill repeats only a short list of the rules agents miss most.
- Agents fix most style and syntax problems themselves, before a person reviews the page.

It also has costs and limits:

- Some standards need judgment that no linter has, such as keeping one type per page, naming the host before a procedure, or giving the reason for a design choice. Only [review](../standards/review-checklist.md) catches those.
- The Google Vale package has its own maintainers, separate from the guide, and can lag behind it. Updating the package is a manual task. See [Track the Google style guide](../standards/README.md#track-the-google-style-guide).
- The checks need Vale, Node.js 22 or later with npm, and jq on the workstation. The first check in a new clone needs the network, to install Vale's Google style and the markdownlint packages. When a check can't run, the hooks don't block, so an agent can finish without that check.
- Spell checking is off. Vale still flags a few British spellings, such as `colour` and `-ise` endings, and product names with the wrong capitalization, but no other misspellings.

Revisit a rule when reviewers keep flagging the same problem. The standards say to turn a problem flagged more than twice into a Vale rule, so that over time tools enforce more of the standard than review does.

## Related

- [Documentation standards](../standards/README.md)
- [Check documentation](check-documentation.md)
