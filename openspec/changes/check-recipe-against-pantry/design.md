# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. A page stores its type; the recipe
template starts its body with an `## Ingredients` heading, but the body is free Markdown the household may have
changed. `internal/markdown` already parses bodies with goldmark (`Links` walks the AST for `[[Title]]` links).
`stock.Store.List(ctx, Pantry)` returns the pantry rows, `Entries` the shopping list, and `AddEntry` adds an entry with
the list's merging rule.

## Goals / Non-Goals

**Goals:**

- Read the ingredients from the body as it is, with no new field on pages.
- Keep the matching a pure function, testable without a database.

**Non-Goals:**

- Any change to how recipes are written or stored.

## Decisions

### Ingredients from the Markdown AST

`markdown.ListUnder(src, heading)` parses the body and returns the plain text of every list item between the first
heading whose text equals `heading` ignoring case and the next heading of the same or a higher level. A nested list's
items count as ingredients too; an item's text is its own text without its sub-list, with `[[Title]]` links reduced to
their title.

*Alternative considered*: a line-based scan for `- ` after `## Ingredients`. It would miss `*` and numbered lists and
headings written with other levels, which goldmark already handles.

### Matching by words, in package `stock`

A name's words are its runs of letters and digits, lower-cased. An ingredient matches a name when the shorter of the two
word sets, by count, is contained in the other; an empty word set never matches. `stock.Check(ingredients, rows,
entries)` returns one result per ingredient: the matching pantry rows, else whether a not-bought entry matches. The web
handler reads the rows and entries and calls it.

*Alternative considered*: SQL `LIKE` per ingredient. It can't express "all the words of the shorter one" and costs a
query per ingredient.

### A page of its own, with a plain form

`GET /pages/{id}/pantry` shows the check; `POST /pages/{id}/pantry` takes the ticked ingredients as pairs of fields
(`add` holding the indexes ticked, `name-<i>` the names) and adds each as a Pantry entry, then redirects to
`/shopping`. Both answer 404 for a page that is not a recipe. With nothing ticked, the POST shows the check again with
a message and a 400, keeping the names typed. A ticked ingredient whose name was emptied is skipped.

*Alternative considered*: an inline panel on the recipe page with htmx. The check is a short detour before shopping, and
a page of its own works the same on a phone without any script.

## Risks / Trade-offs

- [Short words match too much: an ingredient "salt" matches a "Salted butter" row only if the word is identical, but
  "Oil" matches "Olive oil" and "Sesame oil" alike] → accepted: the rows matched are shown by name, and the user decides
  what to add.
- [An ingredient as written makes a poor entry name ("200 g flour")] → the name field is editable before adding; the
  entry's name is what the user leaves there.
