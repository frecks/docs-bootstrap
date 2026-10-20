# Style guide

Follow the [Google developer documentation style guide](https://developers.google.com/style). This file lists only where we differ from it or add to it. If a topic isn't here, Google's guidance applies.

Each change gives its reason, so it can be revisited when our needs change. The **Enforced by** line names the check that catches a violation. Where it says **review**, only a reviewer catches it.

For quick reference, the Google rules that matter most here are:

- [Address the reader as "you"](https://developers.google.com/style/person), in an active voice and present tense.
- [Use sentence case for headings](https://developers.google.com/style/capitalization).
- [Put conditions before instructions](https://developers.google.com/style/procedures): "To restart the service, run…", not "Run… to restart the service."
- [Write link text that says where the link goes](https://developers.google.com/style/link-text), not "here" or "this page".
- [Use code font](https://developers.google.com/style/code-in-text) for commands, file names, paths, and values the reader types.

## Audience

Write for a technical reader who knows the common tools and technologies this project uses, but doesn't know this project or why it's built and set up the way it is. That includes your future self, and agents.

**Why:** Google writes for a broad, global audience, which leads to explaining concepts our readers already know. Explaining this project's own design and choices is what's missing when someone needs to change it or something breaks.

**Enforced by:** review.

## Real values and placeholders

When the reader needs to copy a value that exists in this project, such as a hostname, IP address, path, or port, use the real value, not `example.com` or a documentation IP range. When the reader supplies their own value, use a placeholder in uppercase with underscores, such as `YOUR_DOMAIN`, and say what to replace it with.

**Why:** a page about one specific system is only useful if the reader can copy its real values. Google's placeholder rules are for docs that many people adapt, which is where a placeholder belongs.

**Enforced by:** review.

## Secrets

Never put a private key, password, or token in a page. Use a placeholder in uppercase with underscores, such as `API_TOKEN`, and say where the real value is stored.

```bash
curl -H "Authorization: Bearer API_TOKEN" https://api.example.com/v1/status
```

**Why:** the repository may be shared or published, and a secret in history is hard to remove.

**Enforced by:** Vale (`{{STYLE}}.Secrets`, error).

## Say where a command runs

When a command runs somewhere other than the reader's own machine, such as on a server, in a container, or as another user, name the host and user before the procedure, and again before any command that runs somewhere different from the previous one:

> On `HOST`, as `root`:

**Why:** a command run on the wrong host can do damage. Google's guide assumes a single machine.

**Enforced by:** review.

## Commands and output

- Put commands in a code block with a language, usually `bash`.
- Don't include a prompt such as `$` or `#`, so the block can be copied and pasted.
- Put one command on each line.
- Show expected output in its own code block, marked `text`, after the sentence that introduces it.

**Why:** this follows Google's [code samples](https://developers.google.com/style/code-samples) and [command-line syntax](https://developers.google.com/style/code-syntax) guidance. The only addition is a default language for shell commands, so every block is highlighted the same way.

**Enforced by:** markdownlint (code blocks must be fenced and have a language), and review.

## Product names

Write product names with their exact capitalization, such as GitHub, PostgreSQL, and Terraform. Add new names to `.vale/styles/config/vocabularies/{{STYLE}}/accept.txt`.

**Why:** this matches Google's guidance. The vocabulary makes it enforceable.

**Enforced by:** Vale (`Vale.Terms`, error).

## Dates

Use ISO 8601 dates, such as 2026-10-19.

**Why:** Google prefers "October 19, 2026" for a global audience. Our audience is small, and ISO dates sort correctly and match log timestamps.

**Enforced by:** review. Google's `DateFormat` rule still blocks ambiguous formats such as `10/19/2026`.

## Units

Put a space between a number and its unit, as in `16 GB`. A regular space is fine.

**Why:** Google asks for a nonbreaking space, which is hard to type in Markdown and doesn't change how these pages are read.

**Enforced by:** Vale (`Google.Units`, error). Its message mentions a nonbreaking space, but a regular space passes.

## Minimizing words

Don't use words that say a task is easy, such as "simply", "just", "easy", or "obviously".

**Why:** this is in Google's word list, and a common point in the Write the Docs community. If the reader is stuck, these words make it worse. The Google Vale package doesn't check for them.

**Enforced by:** Vale (`{{STYLE}}.Minimizing`, warning).

## Drafts

Don't leave `TODO`, `TBD`, `FIXME`, or placeholder text in a page. If something is unknown, say what is unknown and how to find out.

**Why:** agents and people both leave markers they mean to come back to, and readers can't act on them.

**Enforced by:** Vale (`{{STYLE}}.DraftMarkers`, error), and `scripts/docs-lint.sh` for `<!-- Template: -->` comments left from a template.

## Serial commas

Use a serial comma, as Google does: "names, paths, and values".

**Why:** the Google Vale rule also flags commas that aren't in a list, so it's lowered to a suggestion, and review catches missing serial commas.

**Enforced by:** Vale (`Google.OxfordComma`, suggestion), and review.

## "We"

Use "we" only for the people who maintain this project, as in "We chose PostgreSQL because…", and only where it's clear who that is. Never use "we" to mean the reader and the writer together, as in "Next, we create a user". Address the reader as "you".

**Why:** Google says to [use first-person plural pronouns carefully](https://developers.google.com/style/person#use-first-person-plural-pronouns-carefully), not to avoid them. "We" fits best in explanation pages, which record decisions and who made them.

**Enforced by:** Vale (`Google.We`, suggestion), and review. The rule points out each "we" without blocking it, so a reviewer can check who it refers to.

## Spelling

Use American spelling, as Google does. Vale's spell checker is off.

**Why:** hostnames, package names, and command names make it report too many false errors. Product names are still checked through the vocabulary.

**Enforced by:** Vale (`Google.Spelling`, warning) for British spellings only.

## Markdown

- Start each page with frontmatter that declares its type, then a single `#` heading, which is the page title. See [structure.md](structure.md#declare-the-type).
- Use `#` headings, not underlines, and don't skip levels.
- Use `-` for bulleted lists and `1.` for every item in a numbered list.
- Use `*` for italic and `**` for bold. Use bold only for UI elements and the label of a definition list item.
- Write each paragraph on one line.
- Use relative links between pages, such as `../api/settings.md`.

**Why:** Google doesn't set Markdown syntax. These choices make pages consistent and keep diffs small: one-line paragraphs change one line per edit, and `1.` numbering doesn't renumber a list when a step is added.

**Enforced by:** markdownlint, and `scripts/check-links.py` for links to missing files. Review checks that links are relative and that bold is used correctly.
