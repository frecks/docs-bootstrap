# Docs bootstrap

The installer in `main.go` installs a documentation system into other repositories. The system itself is in `template/`, and this repository uses it too.

- Change the system in `template/`, not in this repository's installed copy. Then run `go run . .` to upgrade this repository from it. Don't use an installed `docs-bootstrap` for this: it has the template from when it was installed. See `docs/template/change-the-template.md`.
- Run the tests after any change to the installer or `template/`:

  ```bash
  go test ./...
  ```

<!-- docs-bootstrap:begin -->
## Documentation

Documentation lives in `docs/`, grouped in directories by area, such as a component or a system, as in `docs/api/`. Each page declares its [Diátaxis](https://diataxis.fr/) type in frontmatter: `tutorial`, `how-to`, `reference`, or `explanation`. `docs/README.md` indexes every page.

The rules live in `docs/standards/`, starting at `docs/standards/README.md`. They're the single source of truth, so don't restate them elsewhere. Our style is the Google developer documentation style guide plus the changes in `docs/standards/style-guide.md`.

Use the skills:

- `docs-write` to write or update a page.
- `docs-review` to review a page.

Check any page you change:

```bash
scripts/docs-lint.sh docs/path/to/page.md
```

In Claude Code, hooks in `.claude/settings.json` run this after each edit to `docs/`, and on every page before you finish if any page in `docs/` changed. They return errors for you to fix.

When you change code or configuration, update the pages that describe it in the same change. When you change a system directly, such as on a host or in a web interface, update the pages that describe it as part of the same task.
<!-- docs-bootstrap:end -->
