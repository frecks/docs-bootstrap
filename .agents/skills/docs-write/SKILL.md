---
name: docs-write
description: Write or edit Docs bootstrap documentation in docs/ to our standards, using Diátaxis for structure, our Google-based style guide, and Vale. Use when asked to document something, write a runbook, how-to, guide, tutorial, reference page, or explanation, record a decision, or update a page in docs/.
---

# Write documentation

The rules live in `doc-standards/`. This skill is the process for applying them. Read the standards files named below; don't work from memory of the Google guide alone, because ours differs.

## Process

1. **Choose the type.** Read `doc-standards/structure.md`, then decide whether the page is a tutorial, how-to guide, reference, or explanation. If the request covers more than one, plan one page for each and link them. If you can't tell what the reader needs, ask before writing.

1. **Check what exists.** Read `docs/README.md`, and search `docs/` for pages on the same subject. Update an existing page rather than adding a second one. Link to facts on other pages instead of copying them.

1. **Gather facts from the source.** Get names, paths, commands, options, values, hostnames, addresses, and ports from the code, config files, a running build or system, or the user. Don't guess. If a fact can't be confirmed, say in the page what's unknown, and tell the user.

1. **Start from the template.** Copy `doc-standards/templates/<type>.md` to `docs/how-to/<name>.md` for a how-to guide, `docs/tutorials/<name>.md` for a tutorial, or `docs/<system>/<name>.md` for a reference or explanation page, where `<system>` is the path to the system's directory, such as `infrastructure/network`. The system directory is the one a reader would look in first, as described in `doc-standards/structure.md`. Use a lowercase, hyphenated file name that matches the title. Keep the `type` frontmatter from the template.

1. **Write.** Follow `doc-standards/style-guide.md`. The points agents miss most often:
   - When a command runs somewhere other than the reader's own machine, name the host and user first.
   - Never include a secret. Use an uppercase placeholder, and say where the value is stored.
   - Say why any unusual setting exists.
   - Give each how-to guide and tutorial a way to verify it worked.
   - Delete every `<!-- Template: -->` comment.

1. **Add the page to `docs/README.md`**, in the table for its heading, with its type. If the page's directory has a `README.md`, add it there too. Add a how-to guide to the `README.md` of the system it works on, if there is one. If the page starts a new system in a group, add the system to the group's `README.md`.

1. **Link across the split.** Link a how-to guide to the reference and explanation pages it depends on, and add it under **Related** on those pages.

1. **Run the checks.**

   ```bash
   scripts/docs-lint.sh docs/<path>/<name>.md
   ```

   Fix every error. Fix warnings unless you've decided they don't apply, and tell the user which ones you kept and why. Fix suggestions, such as `Google.Passive`, when the rewrite is clearer; otherwise leave them.

   If the output says a check was `SKIPPED`, its tool isn't installed, isn't on `PATH`, or couldn't run. Tell the user, and point them to `docs/how-to/check-documentation.md`. Don't report the page as checked.

   In Claude Code, hooks run this script after each edit in `docs/` or `doc-standards/`, and on every page before you finish if either has uncommitted changes. Run it yourself anyway, so you see every alert.

1. **Get a review.** If you can start a subagent, have one with a fresh context run the `docs-review` skill on the page, then fix what it reports. Otherwise, run `docs-review` yourself.

## Finish

Report the pages you created or changed, the result of `scripts/docs-lint.sh`, and any facts you couldn't confirm.
