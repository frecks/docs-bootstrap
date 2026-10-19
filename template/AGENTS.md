## Documentation

Documentation lives in `docs/`, grouped in directories by {{AREA}}, such as `docs/{{EXAMPLE_AREA}}/`. Each page declares its [Diátaxis](https://diataxis.fr/) type in frontmatter: `tutorial`, `how-to`, `reference`, or `explanation`. `docs/README.md` indexes every page.

The rules live in `docs/standards/`, starting at `docs/standards/README.md`. They're the single source of truth, so don't restate them elsewhere. Our style is the Google developer documentation style guide plus the changes in `docs/standards/style-guide.md`.

Use the skills:

- `docs-write` to write or update a page.
- `docs-review` to review a page.

Check any page you change:

```bash
scripts/docs-lint.sh docs/path/to/page.md
```

In Claude Code, a hook in `.claude/settings.json` runs this after each edit to `docs/`, and returns errors for you to fix.

When you change the {{NOUN}}, update the pages that describe it in the same change.
