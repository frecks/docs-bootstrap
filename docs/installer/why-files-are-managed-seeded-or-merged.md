---
type: explanation
---

# Why the installer manages, seeds, or merges each file

This page explains how the installer decides what to do with each file, and why. Read it before you change how the installer treats a file. For the list of files and what happens to each, see [Options](options.md#files).

## Context

The installer has to work under these conditions:

- **Shared standards:** many repositories use the same documentation system, and each should get improvements when the system changes.
- **Local changes:** each repository adapts the standards, such as adding a style rule or changing its audience. An upgrade must never lose that work.
- **Existing repositories:** a repository may already have a `.gitignore`, Claude Code settings, agent instructions, or its own Vale configuration.
- **Owned content:** some files, such as the docs index, start from the template but then belong to the repository.

## Decision

We made these decisions on 2026-10-19, when we built the installer.

### Managed files upgrade only when unedited

For each managed file, the installer stores a hash of the version it installed in `.docs-bootstrap.json`. On the next run, it compares the file in the repository with that hash:

- If they match, no one has edited the file, so the installer replaces it with the new version.
- If they differ, someone has edited it, so the installer keeps it and reports a conflict.

Local edits always win. To take the new version anyway, you name the file with `--replace`, so losing an edit is always a deliberate choice.

When a new version drops a managed file from the template, the installer deletes it if it's unedited, so a repository doesn't keep files that nothing uses, such as an old script. If someone has edited it, the installer keeps it, and the repository owns it from then on.

When someone deletes a managed file, the installer leaves it deleted. A repository may not want a file, such as the tutorial template, and restoring it on every upgrade would undo that choice.

### The installer creates seed files once

Files such as `docs/README.md` and the vocabulary's `accept.txt` start from the template, but the repository fills them with its own content right away. Upgrading them would mean merging that content, so the installer creates them once and never changes them again.

### The installer merges shared files

Files such as `.gitignore`, `.claude/settings.json`, and `AGENTS.md` hold settings from other tools and people. Replacing them would break those, and reporting them as conflicts would block most installs in existing repositories. So the installer adds only what the documentation system needs, such as a hook or a section, and leaves the rest of the file alone.

### The installer saves options

The installer saves its options, the project name and the date of the first install, in `.docs-bootstrap.json`. An upgrade then renders the template the same way as the first install, including the Vale style name, which comes from the project name. Without this, forgetting an option on upgrade would turn every managed file into a change.

## Alternatives considered

- **Always overwrite:** simple, but it loses local changes, so repositories would stop upgrading.
- **Never overwrite:** safe, but repositories would drift from the shared standards and miss fixes.
- **A three-way merge:** it could combine local and new changes automatically, but it needs the old template version for every file, and its merge conflicts are harder to resolve than a whole-file choice. We may revisit this if conflicts become common.
- **A Git submodule or package for the standards:** it keeps one copy, but it stops each repository from adapting the standards, which is a goal of the style guide.

## Consequences

- A repository can adapt any file and still take updates to the rest.
- Every local edit shows up as a conflict on each upgrade, until someone takes the new version or the template matches the edit. A repository that changes many files gets a long conflict list.
- Seed files never get template improvements. Someone must make changes to them by hand in each repository.

## Related

- [Upgrade a repository](upgrade-a-repository.md)
- [Options](options.md)
