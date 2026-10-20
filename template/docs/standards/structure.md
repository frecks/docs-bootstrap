# Documentation structure

We use [Diátaxis](https://diataxis.fr/) to decide what each page is for. Every page serves one of four reader needs, and a page that tries to serve two serves neither well.

Diátaxis describes pages, not folders. Pages are grouped by the area they describe, such as a component or a system, and each page declares its type in frontmatter.

## The four types

| Type | Reader's need | Answers | Template |
| --- | --- | --- | --- |
| `tutorial` | Learn by doing, with guidance | "Can you teach me to…?" | [tutorial.md](templates/tutorial.md) |
| `how-to` | Complete a specific task | "How do I…?" | [how-to.md](templates/how-to.md) |
| `reference` | Look up a fact | "What is the value of…?" | [reference.md](templates/reference.md) |
| `explanation` | Understand a design or decision | "Why is it like this?" | [explanation.md](templates/explanation.md) |

## Choose a type

Ask two questions about what the reader needs:

1. Is it about **doing** something, or **knowing** something?
1. Is the reader **learning**, or **working**?

| | Learning | Working |
| --- | --- | --- |
| Doing | Tutorial | How-to guide |
| Knowing | Explanation | Reference |

Most pages are how-to guides, such as running a migration, releasing a version, or rotating a certificate, and reference, such as configuration, commands, APIs, and hosts. Write an explanation page when a decision needs its reasons recorded. Write a tutorial when someone new needs to learn the project by doing.

## Declare the type

Start every page with frontmatter that names its type:

```markdown
---
type: how-to
---

# Rotate the TLS certificate
```

The value is one of `tutorial`, `how-to`, `reference`, or `explanation`. The templates include it. `scripts/docs-lint.sh` fails a page without a valid type, and a reviewer checks that the content matches it.

## Organize by area

Put each page in a directory named for the area it describes, such as a component of the code or a system it runs on, in lowercase with hyphens:

```text
docs/
├── README.md              Index of every page, grouped by area
├── standards/             These rules
├── api/
│   ├── README.md          Optional index for the area
│   ├── endpoints.md       type: reference
│   └── add-an-endpoint.md type: how-to
├── network/
│   └── rotate-the-tls-certificate.md   type: how-to
└── storage/
    └── why-the-database-is-postgresql.md   type: explanation
```

- Name the directory for what the reader is working on, such as `api/`, `storage/`, `deployment/`, or `network/`.
- Put a page that spans areas in the directory where a reader would start. Put pages about the project as a whole in `overview/`.
- Don't put pages directly in `docs/`, and don't create directories before they have pages.
- When an area has several pages, add a `README.md` to its directory that lists them by type. Index pages don't need a type.
- Group by subject below an area directory only when it holds enough pages that a reader would struggle to scan it.

Grouping by area keeps everything about one area together, which is how a reader looks for it. The type is a property of the page, so it lives on the page.

## What each type includes

### Tutorial

- A concrete result the reader reaches by the end, stated at the start.
- Steps that always work, in order, with each result shown so the reader knows they're on track.
- Only the explanation needed to keep going. Link to explanation pages for more.
- No choices or alternatives. Pick one path.

### How-to guide

- A title that names the task, starting with a verb, such as "Rotate the TLS certificate".
- Prerequisites, including the tools, access, and host needed.
- Numbered steps, each one an action.
- A way to confirm the task worked.
- No teaching or background. Link to it.

### Reference

- Facts in a predictable layout, usually a table or list, organized like the thing it describes.
- Complete and accurate, with no instructions or opinions.
- The source of each fact, such as the config file it comes from, so it can be checked.

### Explanation

- A title that names the topic, often starting with "Why" or "How", such as "Why the database is PostgreSQL".
- The context, the decision, the alternatives considered, and why they were rejected.
- The consequences and the limits of the decision.
- No steps. Link to the how-to guides that put it into practice.

Write one explanation page per topic. If the topic is a single decision, keep the sections in the explanation template, which follows the shape of an [architecture decision record](https://adr.github.io/). If it covers several related decisions, such as how a whole area fits together, keep the same sections and give each decision its own subsection under **Decision**.

Record the date of each decision. If no one knows it, say so rather than guessing.

## Mixed pages

When a page starts to serve a second need, split it, and link the pages together. Common signs:

- A how-to guide that explains why each step works. Move the reasons to an explanation page.
- A reference page with steps. Move them to a how-to guide.
- A tutorial that offers options. Choose one, and move the options to a how-to guide.

## Names and links

- Name files in lowercase with hyphens, matching the title, such as `rotate-the-tls-certificate.md`.
- Keep each fact in one page, and link to it from others instead of copying it. An explanation page may summarize a rule to explain it, but it links to the rule, and the rule's own page stays the authority.
- Add each new page to `docs/README.md` under its area, with its type, and to the area's `README.md` if it has one.
- Update a page in the same change as the code or configuration it describes.
