---
type: how-to
---

# Change the template

Change the documentation system that the installer sets up, such as a style rule, a skill, or a lint check. Every repository gets the change the next time someone updates `docs-bootstrap` and upgrades the repository.

## Before you begin

- Go 1.22 or later.
- Vale, Node.js 22 or later with `npm`, and `jq`, to run the lint checks on this repository. See [Check documentation](../documentation/check-documentation.md#before-you-begin).

## Make the change

1. Edit the files in `template/`. Use [placeholders](template-syntax.md) wherever the text depends on the project. Write each rule so that it works for any repository. If a rule applies only in some cases, such as to commands that run on another host, say so in the rule. The installer makes a file executable if it starts with `#!`, and not otherwise.

1. To add a file that a repository should own after the first install, add its template path to `seed` in `main.go`. Other new files are managed, and upgrade automatically.

1. To change a markdownlint package version, edit `template/.markdownlint/package.json`, then update the lockfile:

   ```bash
   npm install --package-lock-only --ignore-scripts --prefix template/.markdownlint
   ```

   Check that no package in the new lockfile needs an install script, because `scripts/docs-lint.sh` installs with `--ignore-scripts`. This command prints `0` when none does:

   ```bash
   jq '[.packages[] | select(.hasInstallScript)] | length' template/.markdownlint/package-lock.json
   ```

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

1. If a repository's files would change on upgrade, increase `version` in `main.go`. While the version is 0.x, the system isn't stable: increase the minor version for a change that needs a manual step on upgrade, such as moving pages or deleting a file the template no longer has, and the patch version otherwise. The installer doesn't migrate repositories between versions.

## Verify

Install into a new directory, and check the result:

```bash
go run . /tmp/check-install
/tmp/check-install/scripts/docs-lint.sh
```

The check reports no errors.

## Related

- [Template syntax](template-syntax.md)
- [Upgrade a repository](../installer/upgrade-a-repository.md)
