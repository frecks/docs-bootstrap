# Review checklist

Use this checklist to decide whether a page is ready. It covers what linters can't check. Run `scripts/docs-lint.sh` first, and don't repeat its findings here.

The items come from [Diátaxis](https://diataxis.fr/), the [Write the Docs guide](https://www.writethedocs.org/guide/), and our [style guide](style-guide.md).

## Purpose and type

- [ ] The page serves one reader need, and its content matches the `type` in its frontmatter. See [structure.md](structure.md).
- [ ] The page is where its type belongs: a tutorial or how-to guide in `docs/tutorials/` or `docs/how-to/`, and any other page in the directory of the system a reader would look in first.
- [ ] The title tells the reader what the page is for. A how-to title starts with a verb.
- [ ] The first paragraph says what the page covers and who it's for.
- [ ] Nothing in the page belongs in a different type, such as background in a how-to guide or steps in a reference page.

## Accuracy

- [ ] Every command, path, name, option, value, hostname, address, and port matches the current code or system. Check the source, such as the code or a config file, when one is in the repository.
- [ ] Each procedure that runs somewhere other than the reader's own machine says which host and user it runs as.
- [ ] Commands are complete and can be copied as written.
- [ ] Each how-to guide or tutorial has a way to confirm it worked.
- [ ] Nothing is guessed. If something is unknown, the page says so.

## Reasons

- [ ] Any unusual setting or choice says why it's there, or links to the explanation page that does. Without a reason, the next person may remove it.
- [ ] Warnings explain the consequence, such as "This drops all open connections", not only "Be careful".

## Readers

- [ ] It assumes the audience in the [style guide](style-guide.md#audience): a technical reader who knows the common tools, but not this project.
- [ ] Terms specific to this project are explained or linked the first time they appear.
- [ ] The most important information comes first. The reader can find the step or fact they need by scanning headings.

## Single source of truth

- [ ] Facts aren't copied from another page. The page links to it instead.
- [ ] Links between pages are relative. `scripts/docs-lint.sh` checks that each linked file and anchor exists.
- [ ] The page is listed in `docs/README.md` with its type, and in its directory's `README.md` if there is one.

## Style judgment

- [ ] Bold is used only for UI elements and definition labels, not for emphasis.
- [ ] Link text says where the link goes.
- [ ] Items in a list are parallel in grammar.
- [ ] A series of three or more items in a sentence has a serial comma, as in "names, paths, and values".
- [ ] Sentences are short and direct. Any sentence that has to be read twice is rewritten.

## Severity

Rank each finding:

- **Must fix:** a reader could act on wrong or missing information, such as an incorrect command, address, or claim, a missing prerequisite or host, or a secret.
- **Should fix:** the page works, but a reader may struggle, such as a mixed type, a missing reason, or an unclear sentence.
- **Optional:** polish.

## Ready

A page is ready when all of these are true:

- `scripts/docs-lint.sh` ran with no checks skipped, and reports no errors.
- Each remaining warning is deliberate, with the reason given in the review. Suggestions don't block.
- No **Must fix** finding remains.
