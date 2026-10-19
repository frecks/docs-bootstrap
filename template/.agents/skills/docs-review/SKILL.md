---
name: docs-review
description: Review {{PROJECT}} documentation in docs/ against our standards, running the linters and then checking structure, accuracy, and the review checklist. Use when asked to review, check, or audit a doc or the docs, decide if a page is ready, or after writing a page with docs-write.
---

# Review documentation

Review one or more pages in `docs/`. If no page is named, review the pages changed in the current work, or ask.

Linters handle what can be checked mechanically. Your job is what they can't check. Don't repeat linter findings as your own.

## Process

1. **Run the checks** from the repository root, on each page:

   ```bash
   scripts/docs-lint.sh <file>...
   ```

   If a check prints `SKIPPED`, its tool isn't installed or isn't on `PATH`. The script exits non-zero for that reason, not because of the page. Report which checks were skipped. The page can't be **Ready** until they run.

1. **Read the standards:** `docs/standards/structure.md`, `docs/standards/style-guide.md`, and `docs/standards/review-checklist.md`.

1. **Check the type.** Decide what type the page is from its content, then compare it with the `type` in its frontmatter. Report a mismatch, and any part that belongs to a different type, and say where it should go. Check that the page is in the directory of the {{AREA}} a reader would look in first.

1. **Check accuracy.** For each command, path, name, address, value, and claim about how a tool or the code behaves, find the source and confirm it matches. Sources are, in order of preference: files in the repository, the tool's own documentation, then the running system. If the only evidence is weak, such as a file date, or there's none, list the fact as unconfirmed. Don't run commands that change a system to check a page.

1. **Go through `docs/standards/review-checklist.md`** item by item. Skip items that don't apply to the page's type, without listing them.

1. **Rank and report** using the severity levels and the definition of ready in `docs/standards/review-checklist.md`, in the format below. Don't edit the page unless you were asked to fix it as well.

## Report format

````markdown
## Review: <file>

**Verdict:** Ready | Needs changes | Not checked (a check was skipped)

### Automated checks

```text
<docs-lint.sh output, verbatim>
```

Alerts to keep: <each warning or suggestion that should stay, and why, or "None">

### Findings

1. **Must fix**, <checklist item or style-guide section> (line N): <problem>.
   Suggested: <rewrite or action>.

### Problems in other files

- `<file>`: <problem found while checking this page, such as a standard or config that's wrong>.

### Unconfirmed facts

- Line N: <fact>. <Where you looked.>
````

Order findings by severity. Line numbers are lines in the file. Leave out any section that would be empty, except **Findings**, where you write "None" if there are none. Say plainly when a page is ready, and don't invent findings to fill the report.
