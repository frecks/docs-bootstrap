---
name: docs-write
description: Write or edit Docs bootstrap documentation in docs/ to our standards, using Diátaxis for structure, our Google-based style guide, and Vale. Use when asked to document something, write a runbook, how-to, guide, tutorial, reference page, or explanation, record a decision, or update a page in docs/.
---

# Write documentation

The rules live in `docs/standards/`. This skill is the process for applying them. Read the standards files named below; don't work from memory of the Google guide alone, because ours differs.

## Process

1. **Choose the type.** Read `docs/standards/structure.md`, then decide whether the page is a tutorial, how-to guide, reference, or explanation. If the request covers more than one, plan one page for each and link them. If you can't tell what the reader needs, ask before writing.

1. **Check what exists.** Read `docs/README.md`, and search `docs/` for pages on the same subject. Update an existing page rather than adding a second one. Link to facts on other pages instead of copying them.

1. **Gather facts from the source.** Get names, paths, commands, options, values, hostnames, addresses, and ports from the code, config files, a running build or system, or the user. Don't guess. If a fact can't be confirmed, say in the page what's unknown, and tell the user.

1. **Start from the template.** Copy `docs/standards/templates/<type>.md` to `docs/<area>/<name>.md`. The area directory is the one a reader would look in first, as described in `docs/standards/structure.md`. Use a lowercase, hyphenated file name that matches the title. Keep the `type` frontmatter from the template.

1. **Write.** Follow `docs/standards/style-guide.md`. The points agents miss most often:
   - When a command runs somewhere other than the reader's own machine, name the host and user first.
   - Never include a secret. Use an uppercase placeholder, and say where the value is stored.
   - Say why any unusual setting exists.
   - Give each how-to guide and tutorial a way to verify it worked.
   - Delete every `<!-- Template: -->` comment.

1. **Add the page to `docs/README.md`**, in the table for its area, with its type. If the area directory has a `README.md`, add it there too.

1. **Run the checks.**

   ```bash
   scripts/docs-lint.sh docs/<area>/<name>.md
   ```

   Fix every error. Fix warnings unless you've decided they don't apply, and tell the user which ones you kept and why. Fix suggestions, such as `Google.Passive`, when the rewrite is clearer; otherwise leave them.

   If the output says a check was `SKIPPED`, the tool isn't installed or isn't on `PATH`. Tell the user, and point them to `docs/documentation/check-documentation.md`. Don't report the page as checked.

   In Claude Code, a hook runs this script after each edit in `docs/`, but run it yourself before finishing so you see every alert.

1. **Get a review.** If you can start a subagent, have one with a fresh context run the `docs-review` skill on the page, then fix what it reports. Otherwise, run `docs-review` yourself.

## Finish

Report the pages you created or changed, the result of `scripts/docs-lint.sh`, and any facts you couldn't confirm.
