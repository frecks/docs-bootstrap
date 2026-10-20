---
type: how-to
---

# Change the template

Change the documentation system that the installer sets up, such as a style rule, a skill, or a lint check. Every repository gets the change the next time someone updates `docs-bootstrap` and upgrades the repository.

## Before you begin

- Go 1.22 or later.
- Vale, Node.js, and `jq`, to run the lint checks on this repository. See [Check documentation](../documentation/check-documentation.md#before-you-begin).

## Make the change

1. Edit the files in `template/`. Use [placeholders and profile sections](template-syntax.md) wherever the text depends on the project or profile. The installer makes a file executable if it starts with `#!`, and not otherwise.

1. To add a file that a repository should own after the first install, add its template path to `seed` in `main.go`. Other new files are managed, and upgrade automatically.

1. Run the tests:

   ```bash
   go test ./...
   ```

1. Upgrade this repository, which uses its own system, and run its checks:

   ```bash
   go run . .
   scripts/docs-lint.sh
   ```

   Use `go run .` here and in the verify steps, not an installed `docs-bootstrap`. `go run .` builds from your working tree, while an installed command has the template from when you installed it.

1. If a repository's files would change on upgrade, increase `version` in `main.go`. Use a new major version when an upgrade needs a manual step, such as moving pages.

## Verify

Install into a new directory with each profile, and check the result:

```bash
go run . /tmp/check-software --profile software
/tmp/check-software/scripts/docs-lint.sh
go run . /tmp/check-infrastructure --profile infrastructure
/tmp/check-infrastructure/scripts/docs-lint.sh
```

Each run reports no errors.

## Related

- [Template syntax](template-syntax.md)
- [Upgrade a repository](../installer/upgrade-a-repository.md)
