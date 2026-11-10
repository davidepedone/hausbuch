# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. `Store.Categories(section)` returns
the categories the section's products last had, one per name ignoring case, sorted that way; the section's add form,
the edit page and the shopping list's put-away page all fill their `categories` datalist from it. Groups and chips are
built from the rows themselves (`GroupItems`), not from `Categories`.

## Goals / Non-Goals

**Goals:**

- One place holds the common categories, so that every form that suggests categories gets them.

**Non-Goals:**

- No storage: the common categories never reach the database.

## Decisions

### A map in package `stock`, merged in `Categories`

`commonCategories map[Section][]string` holds each section's list. `Categories` appends the section's common categories
after the used ones, then sorts and compacts ignoring case as it does today; the sort is stable, so of two spellings of
one name the used one, coming first, is kept.

*Alternative considered*: adding the common categories in the web layer or the templates. It would leave `Categories`
meaning "used", but three handlers fill the same datalist, and each would need the merge.

## Risks / Trade-offs

- [A household writing categories in another language sees the English suggestions next to its own] → accepted: they
  are only suggestions; a translated interface would translate them with the rest.
