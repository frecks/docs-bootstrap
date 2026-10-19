---
type: reference
---

# Template syntax

This page lists the placeholders and profile sections you can use in files under `template/`. The source of truth is the `render` and `variables_for` functions in `bootstrap.py`.

## Placeholders

The installer replaces each placeholder, in both file contents and file paths:

| Placeholder | Value | Example |
| --- | --- | --- |
| `{{PROJECT}}` | The `--name` option | `Acme api` |
| `{{STYLE}}` | The `--style` option | `AcmeApi` |
| `{{NOUN}}` | The `--noun` option | `project` |
| `{{AUDIENCE}}` | The `--audience` option, or the profile's default | `a developer who knows…` |
| `{{AREA}}` | What directories group pages by: `component` or `system`, from the profile | `component` |
| `{{EXAMPLE_AREA}}` | An example directory name: `api` or `network`, from the profile | `api` |
| `{{DATE}}` | The date of the first install, in ISO 8601 format | `2026-10-19` |

An unknown placeholder stops the install with an error.

## Profile sections

Text between a profile's opening and closing markers appears only for that profile:

```text
{{#infrastructure}}Name the host before commands.{{/infrastructure}}
{{#software}}Name the tools you need.{{/software}}
```

A marker alone on its own line is removed together with that line, so a section can span whole lines without leaving blank ones:

```text
{{#infrastructure}}
## Say where a command runs
{{/infrastructure}}
```

Sections don't nest.

## Related

- [Change the template](change-the-template.md)
- [Options](../installer/options.md)
