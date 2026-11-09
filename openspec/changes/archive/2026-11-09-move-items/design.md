# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. The current state that shapes the
approach:

- `stock.Store.Update(section, id, user, values)` rewrites a row of a section in one statement and remembers the
  product in that section; `add` merges into an unopened row with the same `name_key` and expiry date, and also
  remembers the product.
- Rows carry their section as text; nothing else refers to a row's section.
- The edit form posts to `/<section>/items/<id>` and the save redirects to `/<section>#item-<id>`.

## Goals / Non-Goals

**Goals:**

- Moving is one save, in one transaction: a row is never in both sections, or in neither.

## Decisions

### Update takes the target section

`Update` gains the section to save the row in. When it is the row's own, nothing changes. Otherwise, inside the same
transaction:

- if the row isn't opened and the target has an unopened row with the same `name_key` and expiry date, that row's
  quantity grows by the edited quantity (with `changed_by` and `changed_at` set) and the moved row is deleted;
- otherwise the row is updated as today with `section` set to the target.

The product is remembered in the target section. The row's name before the edit is read first; once the row is
moved, the old section's product with that name is deleted unless a row with it remains there. `Update` returns the id the row ends up with, so that the save can
redirect to `/<target>#item-<id>`; a target that isn't a section is refused like any other invalid field.

*Alternative considered*: a separate `Move`, called after `Update`. Two transactions, and a failure between them
would leave the edits saved but the row in the old section.

### The field

The edit page gets a Section `<select>` listing `stock.Sections` by label, the row's section selected; the form posts
it as `section`. A post without the field, such as an edit page loaded by an older release, keeps the row in its
section.

## Risks / Trade-offs

- [A merge keeps the target row's location and category, not the moved row's] → As when adding the same product
  twice; the moved row's values are still remembered as the product's latest.
