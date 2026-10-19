# Docs bootstrap

Set up any repository with a documentation system that agents and people can write to the same standard:

- **Structure:** [Diátaxis](https://diataxis.fr/) page types, with pages grouped by topic.
- **Style:** the [Google developer documentation style guide](https://developers.google.com/style), with the project's changes recorded in one file.
- **Checks:** [Vale](https://vale.sh/) for prose, markdownlint for syntax, and checks for page type, links, and leftover template text, in one script.
- **Agents:** `docs-write` and `docs-review` skills, and a Claude Code hook that runs the checks after every edit in `docs/`.

## Quick start

```bash
python3 bootstrap.py ../my-repo --profile software
```

Use `--profile infrastructure` for repositories that describe servers and networks. Run the same command again later to upgrade: the installer updates files you haven't edited, and keeps the ones you have.

See [Set up a repository](docs/installer/set-up-a-repository.md), and the rest of the [documentation](docs/README.md).

## Requirements

- Python 3.8 or later, to run the installer and the link check.
- Vale, Node.js, and `jq`, in the repositories you set up.
