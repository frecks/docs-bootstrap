# Documentation structure

We use [Diátaxis](https://diataxis.fr/) to decide what each page is for. Every page serves one of four reader needs, and a page that tries to serve two serves neither well.

Each page declares its Diátaxis type in frontmatter. Directories put tutorials and how-to guides together by type, and reference and explanation pages by the system they describe. See [Organize the pages](#organize-the-pages).

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

## Organize the pages

A reader comes to the docs either to do something or to look something up, and the tree splits the same way: tasks by type, and knowledge by system. These rules live outside the tree they govern, in `doc-standards/`.

```text
docs/
├── README.md              Index of every page, following the tree
├── tutorials/             Every tutorial
├── how-to/                Every how-to guide
├── overview/              The project as a whole
├── infrastructure/        Group: what the project runs on
│   ├── README.md          Lists the group's systems
│   └── network/
│       ├── network.md     type: reference
│       └── why-the-firewall-is-separate.md  type: explanation
├── services/              Group: what the project provides to its users
│   └── api/
│       └── endpoints.md   type: reference
└── incidents/
    └── 2026-10-20-short-description/
```

- **Tasks by type:** put every tutorial in `docs/tutorials/`, and every how-to guide in `docs/how-to/`. A reader with a task looks for the task, not for the system that does it. Keep each directory flat until it's too long to scan, about 20 pages, then divide it by who does the tasks or by system.
- **Knowledge by system:** put reference and explanation pages in a directory for the system they describe, named in lowercase with hyphens, such as `network/`. Put pages about the project as a whole in `overview/`.
- **Groups only when they help:** put system directories in a group, such as `infrastructure/` or `services/`, when there are more than about eight, or when they split along a line readers use. Keep every page within three directories of `docs/`.
- **Link across the split:** a how-to guide links to the reference and explanation pages it depends on, and those pages link back to it under **Related**.
- **Index each level:** `docs/README.md` lists every page with its type, with how-to guides under headings for who does them. Each group, `how-to/`, `tutorials/`, and each system with several pages has a `README.md` that lists its contents. Index pages don't need a type.
- Don't put pages directly in `docs/`, and don't create directories before they have pages.

`scripts/docs-lint.sh` fails a tutorial or how-to guide outside its directory, and any other type inside one.

### Incidents

Record each outage or data loss in its own directory under `docs/incidents/`, named `YYYY-MM-DD-short-description` for the day it started, such as `incidents/2026-03-14-database-disk-full/`. Each incident's pages stay together, whatever their type. If an incident leads to a new procedure, put it in `docs/how-to/`, and link to it.

- Write the postmortem as an explanation page titled "Why" plus what happened, with a summary, a timeline, the causes, the impact, what changed, the lessons, and the open actions.
- Add any other pages the incident needs in the same directory, such as a reference page listing the files kept from the recovery.
- List each incident in `docs/incidents/README.md` and in `docs/README.md`, newest first.

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
- Add each new page to `docs/README.md` under its heading, with its type, and to its directory's `README.md` if it has one.
- Update a page in the same change as the code or configuration it describes. When you change a system directly instead, such as on a host, in a web interface, or in a cloud console, update the pages that describe it as part of the same task, before you call the task done.
