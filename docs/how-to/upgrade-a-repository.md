---
type: how-to
---

# Upgrade a repository

Bring a repository up to date with the current version of the documentation system. Upgrade after a new version of `docs-bootstrap`, such as one with a new rule or a new pinned version of Vale's Google style.

While the version is 0.x, the system isn't stable, and the installer doesn't migrate a repository between versions. A new version can drop or rename files, and the installer leaves the old ones in place. After you upgrade, compare the repository with the new template, and delete or move what's left over by hand.

## Before you begin

- The repository was set up with the installer, so it has a `.docs-bootstrap.json` file. To set one up, see [Set up a repository](set-up-a-repository.md).
- A clean working tree in the repository, so you can review the changes with `git diff`.

## Upgrade

1. Update `docs-bootstrap`, so it has the current documentation system. See [Install docs-bootstrap](install-docs-bootstrap.md#update).

1. Run the installer on the repository again. You don't need to repeat its options, because it reads them from `.docs-bootstrap.json`:

   ```bash
   docs-bootstrap ../my-repo
   ```

1. Read the output:

   - **Updated:** files you hadn't edited, now at the new version.
   - **Conflicts, left unchanged:** files you edited, which the installer kept.
   - **Not restored:** files you deleted, which the installer leaves deleted.
   - **Removed:** files that an earlier version installed and the new template doesn't have, such as the `Secrets.yml` Vale rule that 0.6.0 dropped. The installer deletes each one you hadn't edited.
   - **No longer in the template, but kept:** the same kind of file, which you edited. The installer leaves it, and stops managing it. Delete it if nothing uses it.
   - **Not updated:** seed files, such as `docs/how-to/check-documentation.md`, that differ from the new version because you edited them, or because an installer before 0.4.0 created them without recording a hash. The installer lists each one once for each new version. If you haven't edited one, take the new version with `--replace` and its path, as for a conflict.

1. For each conflict, compare your version with the new one:

   ```bash
   docs-bootstrap ../my-repo --diff
   ```

   Then do one of these:

   - To keep your version, do nothing. The installer reports the file again on each upgrade.
   - To take the new version and lose your edits, run the installer with `--replace` and the file's path:

     ```bash
     docs-bootstrap ../my-repo --replace doc-standards/style-guide.md
     ```

   - To keep your edits and add the new changes, edit your file by hand to include them.

1. If you upgraded from a version before 0.5.0, move the pages, as described in [Move pages to the 0.5.0 layout](#move-pages-to-the-050-layout).

1. If you upgraded from a version before 0.6.0, read [Changes in 0.6.0](#changes-in-060).

1. Review the changes with `git diff`, and commit them.

## Changes in 0.6.0

Version 0.6.0 replaces seven of the Google Vale rules with copies that fit technical docs, and adds a git pre-commit hook. The [style guide](../../doc-standards/style-guide.md) and [rationale](../../doc-standards/rationale.md) say why.

- **Vale rules:** the `Headings`, `Colons`, `Acronyms`, `DateFormat`, `Contractions`, `Semicolons`, and `Parens` rules in your style replace Google's, which `.vale.ini` turns off. `Google.Passive` and `Google.OxfordComma` are off too. If you kept your `.vale.ini` as a conflict, the new rules run alongside Google's until you take the new version or turn Google's off by hand. Run `scripts/docs-lint.sh` afterward, and fix what the new rules flag.
- **Secrets:** the `Secrets` Vale rule is gone. The pre-commit hook runs gitleaks on every commit instead, with the allowlists in `.gitleaks.toml`.
- **Pre-commit hook:** the installer turns on the hook in `.githooks/` by setting `core.hooksPath`, unless that's already set or you ran the installer with `--skip-tools`. If so, see [Check before each commit](check-documentation.md#check-before-each-commit).
- **Links after an edit:** the Claude Code edit hook reports a link to a file or heading that doesn't exist as a warning. The Stop hook and the pre-commit hook still stop on it.

If you edited `docs/how-to/check-documentation.md`, the installer lists it as not updated. Add the gitleaks install steps and the *Check before each commit* section from the new version to your copy.

## Move pages to the 0.5.0 layout

Version 0.5.0 moved the standards out of `docs/` and put how-to guides and tutorials in their own directories. The installer adds the new files, and leaves the old ones for you to move. `scripts/docs-lint.sh` reports each page that's still in the wrong place.

1. If you edited any file in `docs/standards/`, make the same edits to its copy in `doc-standards/`. Then delete `docs/standards/`.

1. If you edited `docs/documentation/why-the-documentation-system-works-this-way.md`, make the same edits to `doc-standards/rationale.md`. If you edited `docs/documentation/check-documentation.md`, make them to `docs/how-to/check-documentation.md`. Then delete `docs/documentation/`.

1. Move every page with `type: how-to` to `docs/how-to/`, and every page with `type: tutorial` to `docs/tutorials/`. Leave reference and explanation pages in the directory for their system.

1. Update `docs/README.md` and the `README.md` in each directory, so they list the pages where they are now. Add a `README.md` to `docs/tutorials/` if you have tutorials.

1. Run the checks, and fix each link and placement they report:

   ```bash
   scripts/docs-lint.sh
   ```

## Verify

Run the installer again:

```bash
docs-bootstrap ../my-repo
```

If you took every new version, the output includes this message:

```text
Already up to date.
```

## Related

- [Options](../installer/options.md)
- [Why the installer manages, seeds, or merges each file](../installer/why-files-are-managed-seeded-or-merged.md)
