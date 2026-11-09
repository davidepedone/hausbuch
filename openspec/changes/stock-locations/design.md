# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. The current state that shapes the
approach:

- `mergeInto` adds a quantity to the first unopened row of a section with the same `name_key` and expiry date, in one
  `UPDATE … WHERE id = (SELECT …)`. `add` (adding and restoring) and `save` (moving to another section) both use it.
- `stock_item.location` holds the location as entered; there is no lower-cased key for it, unlike `name_key`, and
  SQLite's `lower` only folds ASCII.
- `Open` already splits part of a row into a new row inside one transaction; the row's buttons post to
  `/<section>/items/<id>/<action>` and answer with the section's list.

## Goals / Non-Goals

**Goals:**

- A move is one transaction: the quantity taken from a row is always found in the other location.

**Non-Goals:**

- No schema change.

## Decisions

### Matching the location in Go

`mergeInto` selects the candidate rows by section, `name_key`, expiry date and `opened_on IS NULL`, as today, with
their locations, and picks the first one, by id, whose location matches `nameKey(location)` (trimmed, lower-cased with
Go's Unicode folding); then it updates that row by id. Adding, restoring and moving to another section all get the new
rule through it.

*Alternative considered*: a `location_key` column, filled by a migration. It would keep the match in SQL, but adds a
column to keep in sync on every write for a lookup that reads a handful of rows.

### `Store.Move`

`Move(section, id, userID, quantity, location)` checks, inside one transaction, that the row exists, that the quantity
is not above the row's (`ErrQuantity` otherwise) and that the trimmed location differs from the row's, ignoring case
(new `ErrSameLocation`). Then, with the row's values and the new location and quantity:

- if `mergeInto` finds a matching row, the quantity is taken from the moved row, which is deleted when nothing is left;
- otherwise, moving the whole quantity sets the row's location; moving part of it takes the quantity from the row and
  inserts a copy with the new location and quantity, as `Open` does, recorded as added and changed by the user now.

The product is remembered with the new location, through `mergeInto` or `remember`.

### The move page

A row gets a "Move" link next to "Edit", leading to `GET /<section>/items/<id>/move`: a small page with the row's
name, quantity and location, a Quantity field (1, or the row's quantity when less) and a "Move to" field with the
section's locations as suggestions. It posts to `POST /<section>/items/<id>/move`, registered before the generic
`{action}` route so that it is not taken as one of the row's buttons. A refusal renders the page again with what was
entered and the message; success redirects to `/<section>#item-<id>` of the row that received the quantity.

*Alternative considered*: a form inside the row, opened in place with HTMX. It would save a page load, but the rows are
already dense on a phone and the edit page sets the precedent of a separate page.

## Risks / Trade-offs

- [Changing the location from the edit page can still leave two rows that would have merged] → They are listed next
  to each other and the "Move" action merges them; the proposal keeps the edit page out of scope.
- [A merge keeps the receiving row's category and other values, not the moved row's] → As when adding the same
  product twice.
