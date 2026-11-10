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

### Choices under the field, next to the datalist

Safari on iOS doesn't open a datalist's options when the field is tapped: it shows a few of them in the keyboard's bar,
only once something is typed. The category field therefore also lists the suggestions as chip buttons under it, in a
`categoryField` part shared by the three forms; `suggestions.js` writes a tapped choice into the field and fires an
`input` event, so that `productfields.js` treats it as typed and a late answer for a known name doesn't overwrite it.
The datalist stays: on desktop browsers it narrows the suggestions while typing.

*Alternative considered*: replacing the datalist with a custom dropdown. It would need its own keyboard handling and
accessibility for what a row of buttons already does.

## Risks / Trade-offs

- [A household with many categories gets a long row of choices] → accepted: the choices wrap, and a section rarely has
  more than a dozen categories.

- [A household writing categories in another language sees the English suggestions next to its own] → accepted: they
  are only suggestions; a translated interface would translate them with the rest.
