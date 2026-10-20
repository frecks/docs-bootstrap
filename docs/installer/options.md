---
type: reference
---

# Options

This page lists the options and profiles of the installer, and how it treats each file it installs. The source of truth is `main.go` and the files in `template/`.

## Command

```bash
docs-bootstrap TARGET [OPTIONS]
```

`TARGET` is the repository to set up. The installer creates the directory if it doesn't exist. Options can come before or after `TARGET`. To install the command, see [Install docs-bootstrap](install-docs-bootstrap.md).

The command includes a copy of the template from when you installed it. To change the template, run the installer from the `docs-bootstrap` repository with `go run .` instead, as described in [Change the template](../template/change-the-template.md).

## Options

| Option | Default | Description |
| --- | --- | --- |
| `--profile` | `software` | The rule set: `software` or `infrastructure`. See [Profiles](#profiles). |
| `--name` | The directory name, with hyphens and underscores as spaces and the first letter capitalized | Project name, used in titles such as `NAME documentation`. Required if `TARGET` is a root directory. |
| `--style` | The words of the project name, each capitalized, without spaces or punctuation, or `House` if that doesn't start with a letter | Name of the local Vale style and vocabulary, such as `AcmeApi`. Letters and digits only. |
| `--noun` | `project` or `environment`, from the profile | What the docs describe, as in `this NOUN`. |
| `--audience` | A sentence from the profile | Who the docs are for, completing `Write for …`. |
| `--dry-run` | Off | Prints what would change, without writing. |
| `--diff` | Off | Prints a diff for each file in conflict, using `git diff`, with your version as `yours/` and the new version as `docs-bootstrap/`. Needs Git. |
| `--replace PATH` | None | Takes the new version of a managed file, replacing local edits. Repeat it for more files. |
| `--version` | Off | Prints the version and exits. Doesn't need a `TARGET`. |

The installer saves `--profile`, `--name`, and `--style` in `.docs-bootstrap.json`, and `--noun` and `--audience` only if you give them. A later run uses the saved values, unless you give the option again. Without a saved value, `--noun` and `--audience` follow the profile, so they change if you change the profile. To go back to the default, give an empty value, such as `--audience ""`.

The installer exits with status 2 when the command line is invalid, such as an unknown option or profile, a missing `TARGET`, a `--style` with other characters, or a `--replace` path that isn't a managed file. It exits with status 1 when any file is in conflict or the install fails. Otherwise, it exits with status 0.

## Profiles

| | `software` | `infrastructure` |
| --- | --- | --- |
| Directories group pages by | Component, such as `docs/api/` | System, such as `docs/network/` |
| Default audience | A developer who knows the languages and tools, but not this codebase | An operator who knows Linux and networking, but not this environment |
| Real hostnames and addresses rule | No | Yes |
| Name the host before each command | No | Yes |
| Facts come from | The code, config files, and a running build | Config files and the running system |

## Files

The installer treats each file one of three ways. For the reasons, see [Why the installer manages, seeds, or merges each file](why-files-are-managed-seeded-or-merged.md).

| Kind | Files | On first run | On later runs |
| --- | --- | --- | --- |
| Managed | Every file in `template/` that isn't listed below | Created, or a conflict if a different file exists | Updated if unedited since the last run, otherwise a conflict |
| Seed | `docs/README.md`, the pages in `docs/documentation/`, and the vocabulary's `accept.txt` | Created if missing | Never changed |
| Merged | `.gitignore`, `.claude/settings.json`, `AGENTS.md`, `CLAUDE.md`, and the links in `.claude/skills/` | Created, or the needed lines added to the existing file | The needed lines added again if missing |

How each merged file changes:

- **`.gitignore`:** the installer adds the lines from the template that aren't already in the file, each with any comment directly before it in the template.
- **`.claude/settings.json`:** the installer adds the documentation hook to `hooks.PostToolUse`, and keeps every other setting and hook.
- **`AGENTS.md`:** the installer adds a documentation section between `<!-- docs-bootstrap:begin -->` and `<!-- docs-bootstrap:end -->`, and replaces that section on later runs. Text outside it is yours. If either comment is missing, the installer reports a conflict and leaves the file unchanged.
- **`CLAUDE.md`:** if it's missing, the installer creates it as a link to `AGENTS.md`. If it's a file, the installer adds an `@AGENTS.md` line, which makes Claude Code read `AGENTS.md`.

The installer keeps the line endings of each file it changes, so a file with CRLF line endings keeps them.

## Manifest

`.docs-bootstrap.json` in the target repository records:

| Key | Contents |
| --- | --- |
| `version` | The installer version that last ran. |
| `options` | The saved options, and the date of the first install. |
| `files` | A SHA-256 hash of each managed file, as last installed. |

## Related

- [Install docs-bootstrap](install-docs-bootstrap.md)
- [Set up a repository](set-up-a-repository.md)
- [Template syntax](../template/template-syntax.md)
