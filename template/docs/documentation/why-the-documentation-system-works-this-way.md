---
type: explanation
---

<!-- vale Google.We = NO -->

# Why the documentation system works this way

This page explains why {{PROJECT}} documentation uses Diátaxis for structure, a style guide based on Google's, Vale and markdownlint for checks, and a Claude Code hook that runs those checks for agents. Read it before you change any part of the system, so you know what each part protects against. The rules themselves are in the [documentation standards](../standards/README.md), and this page doesn't repeat them.

## Context

The documentation has to work under these conditions:

- **Readers:** {{AUDIENCE}}. That includes a future self and agents. See [Audience](../standards/style-guide.md#audience).
- **Writers:** people and agents both write pages, so the rules need one written source that both can read and check against.
{{#infrastructure}}
- **Environment:** the pages describe one specific environment with many hosts. A command run on the wrong host can do damage.
{{/infrastructure}}
{{#software}}
- **Codebase:** the pages describe code that changes often, so pages go stale unless they change with it.
{{/software}}
- **Sharing:** the repository may end up shared or published, and a secret in its history is hard to remove.

## Decision

We set up the system on {{DATE}}. It has six parts, and each one answers a different question about a page or about the system itself.

### Diátaxis decides what a page is for

Every page serves one of four reader needs: learning by doing, completing a task, looking up a fact, or understanding a decision. [Diátaxis](https://diataxis.fr/) names these needs as four types: tutorial, how-to guide, reference, and explanation. A page that tries to serve two needs serves neither well, so a page that drifts gets split. See [Documentation structure](../standards/structure.md).

Each page declares its type in frontmatter, and directories group pages by the {{AREA}} they describe, such as `docs/{{EXAMPLE_AREA}}/`. A reader looks for help by {{AREA}}, so everything about one {{AREA}} is in one place. The type stays visible on each page and in the index.

### The style guide starts from Google's and records each difference

Pages follow the [Google developer documentation style guide](https://developers.google.com/style), and [style-guide.md](../standards/style-guide.md) lists only where this {{NOUN}} differs or adds to it. Google covers the general questions of voice, headings, procedures, and code, so the local file stays short.

Google writes for a broad, global audience. This {{NOUN}}'s pages have a small, known audience, so some Google rules don't fit. Examples include ISO 8601 dates such as 2026-10-19{{#infrastructure}}, real hostnames instead of placeholders, and a named host and user before each procedure{{/infrastructure}}{{#software}}, and a regular space between a number and its unit{{/software}}. Each difference states its reason, so that someone can revisit it when needs change, and names the check that enforces it.

The [review checklist](../standards/review-checklist.md) draws on Diátaxis, the [Write the Docs guide](https://www.writethedocs.org/guide/), and the style guide.

### Vale checks prose

[Vale](https://vale.sh/) turns style rules into checks that run on every page, no matter who wrote it. It runs three styles:

- **Google:** the [errata-ai/Google](https://github.com/errata-ai/Google) package, pinned to a release in `.vale.ini`. Pinning means a new package release can't change the results of a check without a deliberate update. `vale sync` downloads the package into `.vale/styles/Google/`, which `.gitignore` excludes.
- **{{STYLE}}:** local rules in `.vale/styles/{{STYLE}}/` for what Google's package doesn't check, such as secrets, draft markers, and minimizing words.
- **Vale:** Vale's built-in rules, used with the {{STYLE}} vocabulary in `.vale/styles/config/vocabularies/{{STYLE}}/` to enforce the capitalization of product names.

Three settings in `.vale.ini` change what the styles check:

- `Vale.Spelling = NO` turns off the spell checker. Hostnames, package names, and tool names make it report too many false errors. Product names are still checked through the vocabulary.
- `BasedOnStyles =` is empty for `docs/standards/`, so Vale doesn't check the standards. They quote bad examples on purpose, and the templates hold placeholders.
- `Google.OxfordComma = suggestion` lowers the serial comma rule, because it also flags commas that aren't in a list. Review checks serial commas instead.

Explanation pages also allow the word "we" for the people who made a decision. Vale can't read a page's type from frontmatter, so the explanation template turns off `Google.We` with a `<!-- vale Google.We = NO -->` comment instead of a setting in `.vale.ini`.

### A Markdown linter checks syntax

[markdownlint](https://github.com/DavidAnson/markdownlint) checks Markdown syntax, which Vale doesn't. `.markdownlint-cli2.jsonc` enables the default rules, and picks one style wherever Markdown offers a choice:

- `#` for headings.
- `-` for bullets.
- `1.` for every item in a numbered list.
- Fenced code blocks.
- `*` for emphasis.

It turns off the line-length rule, so each paragraph is one line and editors wrap it. It allows no inline HTML except `<kbd>`.

`scripts/docs-lint.sh` pins markdownlint-cli2 to a version and runs it through `npx`, so the check needs no separate install beyond Node.js.

### One script runs every check

`scripts/docs-lint.sh` runs five checks in order: page placement and type, leftover template comments, links, Vale, and markdownlint. The first check confirms that each page is in a {{AREA}} directory and that its frontmatter declares one of the four types. Vale skips HTML comments, so the script checks for `<!-- Template: -->` comments itself. Neither Vale nor markdownlint checks that a linked file exists, so `scripts/check-links.py` does. The script fails on any error, and prints Vale warnings and suggestions without failing.

Having one script means people, agents, and the hook all run the same checks. To run it, see [Check documentation](check-documentation.md).

A `PostToolUse` hook in `.claude/settings.json` runs the script after each `Edit`, `MultiEdit`, or `Write` in Claude Code, on Markdown files in `docs/`. Errors go back to the agent to fix before it moves on. Warnings and suggestions go to the agent as context, without blocking. If Vale isn't installed, the hook asks the agent to tell the user, and doesn't block. The hook catches problems at the edit that introduced them, instead of in review.

### A bootstrap tool installs and upgrades the system

The docs-bootstrap tool installed these files, and `.docs-bootstrap.json` records the version and a hash of each file it manages. Running the tool again upgrades every file that no one here has edited, and reports the files that someone has. Edits made here always win, so this {{NOUN}} can adapt its standards and still take improvements to the shared ones.

## Alternatives considered

We considered and rejected these alternatives:

- **A directory for each Diátaxis type, such as `how-to/`:** it splits each {{AREA}}'s pages across four directories, and pushes you to create empty structure before there's content, which [Diátaxis advises against](https://diataxis.fr/how-to-use-diataxis/). The [Discord API docs](https://github.com/discord/discord-api-docs/tree/main/developers) show the alternative: group by subject, and keep the types within each subject.
{{#infrastructure}}
- **Google's placeholder values, such as `example.com`:** rejected because readers here need to copy the real value. See [Real names and addresses](../standards/style-guide.md#real-names-and-addresses).
{{/infrastructure}}
- **Google's date format, such as "October 19, 2026":** rejected because ISO 8601 dates sort correctly and match log timestamps. See [Dates](../standards/style-guide.md#dates).
- **A nonbreaking space between a number and its unit:** rejected because it's hard to type in Markdown and doesn't change how pages read. See [Units](../standards/style-guide.md#units).

## Consequences

The system makes these things easier:

- A reader finds everything about one {{AREA}} in one directory, and each page says what kind of help it gives.
- A rule changes in one place, `docs/standards/`, and `AGENTS.md` and the agent skills link there. The `docs-write` skill repeats only a short list of the rules agents miss most.
- Agents fix most style and syntax problems themselves, before a person reviews the page.

It also has costs and limits:

- Some standards need judgment that no linter has, such as {{#infrastructure}}naming the host before a procedure or using real addresses{{/infrastructure}}{{#software}}keeping one type per page or giving the reason for a design choice{{/software}}. Only [review](../standards/review-checklist.md) catches those.
- The Google Vale package has its own maintainers, separate from the guide, and can lag behind it. Updating the package is a manual task. See [Track the Google style guide](../standards/README.md#track-the-google-style-guide).
- The checks need Vale, Node.js, Python 3, and jq on the workstation. Without Vale, the hook doesn't block, so an agent can finish without checks.
- Spell checking is off. Vale still flags a few British spellings, such as `colour` and `-ise` endings, and product names with the wrong capitalization, but no other misspellings.

Revisit a rule when reviewers keep flagging the same problem. The standards say to turn a problem flagged more than twice into a Vale rule, so that over time tools enforce more of the standard than review does.

## Related

- [Documentation standards](../standards/README.md)
- [Check documentation](check-documentation.md)
