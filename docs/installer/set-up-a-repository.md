---
type: how-to
---

# Set up a repository

Install the documentation system into a new or existing repository. The repository gets the documentation standards, Vale and markdownlint configuration, a lint script, a Claude Code hook, and the `docs-write` and `docs-review` agent skills.

## Before you begin

- The `docs-bootstrap` command. See [Install docs-bootstrap](install-docs-bootstrap.md).
- A clean working tree in the target repository, if it already exists, so you can review the changes with `git diff`.
- The tools the checks need, which are listed in [Check documentation](../documentation/check-documentation.md#before-you-begin).

## Choose a profile

Pick the profile that matches what the repository describes. See [Options](options.md#profiles) for the differences.

- `software` is the default, for code that people build and change.
- `infrastructure` is for servers, networks, and other systems that people operate. It adds rules for real hostnames and for naming the host before each command.

## Install

1. Preview the changes:

   ```bash
   docs-bootstrap ../my-repo --profile software --dry-run
   ```

   Replace `../my-repo` with the path to your repository. The output lists each file the installer would create, merge, or leave alone.

1. Run the same command without `--dry-run`:

   ```bash
   docs-bootstrap ../my-repo --profile software
   ```

   If the directory doesn't exist, the installer creates it.

1. If the output lists **Conflicts**, your repository already has a file the installer manages, such as `.vale.ini`. The installer keeps your file. Compare the two versions:

   ```bash
   docs-bootstrap ../my-repo --diff
   ```

   Then either merge the changes you want into your file by hand, or take the new version:

   ```bash
   docs-bootstrap ../my-repo --replace .vale.ini
   ```

1. If your `AGENTS.md` already described your documentation, delete that description. The installer added its own, between the `docs-bootstrap:begin` and `docs-bootstrap:end` comments, and keeps it up to date.

1. In your repository, add product names that your docs use to `.vale/styles/config/vocabularies/STYLE/accept.txt`, where `STYLE` is the style name the installer printed.

## Verify

In your repository, run:

```bash
scripts/docs-lint.sh
```

The output ends with this section:

```text
== markdownlint
OK
```

A fresh install has no errors. In an existing repository with pages in `docs/`, expect errors for pages without a `type` in frontmatter, or not in a directory for their topic. Fix them as described in the repository's `docs/standards/structure.md`. An agent can do this with the `docs-write` skill.

## Related

- [Install docs-bootstrap](install-docs-bootstrap.md)
- [Upgrade a repository](upgrade-a-repository.md)
- [Options](options.md)
- [Why the installer manages, seeds, or merges each file](why-files-are-managed-seeded-or-merged.md)
