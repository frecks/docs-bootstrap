# Docs bootstrap

Set up any repository with a documentation system that agents and people can write to the same standard:

- **Structure:** [Diátaxis](https://diataxis.fr/) page types, with pages grouped by topic.
- **Style:** the [Google developer documentation style guide](https://developers.google.com/style), with the project's changes recorded in one file.
- **Checks:** [Vale](https://vale.sh/) for prose, markdownlint for syntax, and checks for page type, links, and leftover template text, in one script.
- **Agents:** `docs-write` and `docs-review` skills, and Claude Code hooks that run the checks after every edit in `docs/` and before the agent finishes.

## Quick start

Install the `docs-bootstrap` command from this repository:

```bash
go install .
```

Then set up a repository:

```bash
docs-bootstrap ../my-repo
```

The same rules work for code, infrastructure, or both. To upgrade later, update `docs-bootstrap` and run the same command again. The installer updates files you haven't edited, and keeps the ones you have.

See [Install docs-bootstrap](docs/installer/install-docs-bootstrap.md), [Set up a repository](docs/installer/set-up-a-repository.md), and the rest of the [documentation](docs/README.md).

## Requirements

- Go 1.22 or later, to install `docs-bootstrap`, and Git, for `--diff`.
- Vale, Node.js 22 or later with `npm`, and `jq`, in the repositories you set up.
