---
type: how-to
---

# Change the template

Change the documentation system that `bootstrap.py` installs, such as a style rule, a skill, or a lint check. Every repository gets the change the next time someone upgrades it.

## Before you begin

- Python 3.8 or later.
- Vale, Node.js, and `jq`, to run the lint checks on this repository. See [Check documentation](../documentation/check-documentation.md#before-you-begin).

## Make the change

1. Edit the files in `template/`. Use [placeholders and profile sections](template-syntax.md) wherever the text depends on the project or profile.

1. To add a file that a repository should own after the first install, add its template path to `SEED` in `bootstrap.py`. Other new files are managed, and upgrade automatically.

1. Run the tests:

   ```bash
   python3 -m unittest discover -s tests
   ```

1. Upgrade this repository, which uses its own system, and run its checks:

   ```bash
   python3 bootstrap.py .
   scripts/docs-lint.sh
   ```

1. If a repository's files would change on upgrade, increase `VERSION` in `bootstrap.py`. Use a new major version when an upgrade needs a manual step, such as moving pages.

## Verify

Install into a new directory with each profile, and check the result:

```bash
python3 bootstrap.py /tmp/check-software --profile software
/tmp/check-software/scripts/docs-lint.sh
python3 bootstrap.py /tmp/check-infrastructure --profile infrastructure
/tmp/check-infrastructure/scripts/docs-lint.sh
```

Each run reports no errors.

## Related

- [Template syntax](template-syntax.md)
- [Upgrade a repository](../installer/upgrade-a-repository.md)
