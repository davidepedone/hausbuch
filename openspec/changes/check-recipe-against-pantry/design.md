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

A name's words are its runs of letters and digits, lower-cased, leaving out those holding a digit ("200", "00") and a
short list of English and Italian filler words (articles, "of", "with", "di", "con" and the like). An ingredient
matches a name when the two share a word. `stock.Check(ingredients, rows, entries)` returns one result per ingredient:
the matching pantry rows, else whether a not-bought entry matches. The web handler reads the rows and entries and calls
it.

*Alternative considered*: all the words of the shorter name in the other. It misses the common case of an ingredient
written with a unit, "1 tbsp oil" against an "Olive oil" row, since neither name holds all the other's words.

*Alternative considered*: SQL `LIKE` per ingredient. It costs a query per ingredient for what a loop over a few hundred
rows does in memory.

### A page of its own, with a plain form

`GET /pages/{id}/pantry` shows the check; `POST /pages/{id}/pantry` takes the ticked ingredients as pairs of fields
(`add` holding the indexes ticked, `name-<i>` the names) and adds each as a Pantry entry, then redirects to
`/shopping`. Both answer 404 for a page that is not a recipe. With nothing ticked, the POST shows the check again with
a message and a 400, keeping the names typed. A ticked ingredient whose name was emptied is skipped.

*Alternative considered*: an inline panel on the recipe page with htmx. The check is a short detour before shopping, and
a page of its own works the same on a phone without any script.

## Risks / Trade-offs

- [One shared word matches too much: "Tomato sauce" finds a "Soy sauce" row, "1 tbsp oil" both "Olive oil" and
  "Sesame oil"] → accepted: the rows matched are shown by name, and the user decides what to add; a wrong "in the
  pantry" is seen at once, a wrong "missing" is not.
- [Filler words outside the list, or in other languages, still match] → the list covers English and Italian, the
  languages households are expected to write in; more can be added when seen.
- [An ingredient as written makes a poor entry name ("200 g flour")] → the name field is editable before adding; the
  entry's name is what the user leaves there.
