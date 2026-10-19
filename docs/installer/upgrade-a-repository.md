---
type: how-to
---

# Upgrade a repository

Bring a repository up to date with the current version of the documentation system. Upgrade after the template in this repository changes, such as a new rule or a new pinned version of Vale's Google style.

## Before you begin

- The repository was set up with the installer, so it has a `.docs-bootstrap.json` file. To set one up, see [Set up a repository](set-up-a-repository.md).
- A clean working tree in the repository, so you can review the changes with `git diff`.

## Upgrade

1. From the `docs-bootstrap` repository, run the installer on the repository again. You don't need to repeat its options, because it reads them from `.docs-bootstrap.json`:

   ```bash
   go run . ../my-repo
   ```

1. Read the output:

   - **Updated:** files you hadn't edited, now at the new version.
   - **Conflicts, left unchanged:** files you edited, which the installer kept.
   - **Not restored:** files you deleted, which the installer leaves deleted.

1. For each conflict, compare your version with the new one:

   ```bash
   go run . ../my-repo --diff
   ```

   Then do one of these:

   - To keep your version, do nothing. The installer reports the file again on each upgrade.
   - To take the new version and lose your edits, run the installer with `--replace` and the file's path:

     ```bash
     go run . ../my-repo --replace docs/standards/style-guide.md
     ```

   - To keep your edits and add the new changes, edit your file by hand to include them.

1. Review the changes with `git diff`, and commit them.

## Verify

Run the installer again:

```bash
go run . ../my-repo
```

If you took every new version, the output ends with this message:

```text
Already up to date.
```

## Related

- [Options](options.md)
- [Why the installer manages, seeds, or merges each file](why-files-are-managed-seeded-or-merged.md)
