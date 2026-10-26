---
type: reference
---

# Template syntax

This page lists the placeholders you can use in files under `template/`. The source of truth is the `render` and `variablesFor` functions in `main.go`.

## Placeholders

The installer replaces each placeholder, in both file contents and file paths:

| Placeholder | Value | Example |
| --- | --- | --- |
| `{{PROJECT}}` | The `--name` option | `Acme api` |
| `{{STYLE}}` | The Vale style name, made from the project name | `AcmeApi` |
| `{{DATE}}` | The date of the first install, in ISO 8601 format | `2026-10-19` |

An unknown placeholder, or any other `{{` left after rendering, stops the install with an error.

## Related

- [Change the template](../how-to/change-the-template.md)
- [Options](../installer/options.md)
