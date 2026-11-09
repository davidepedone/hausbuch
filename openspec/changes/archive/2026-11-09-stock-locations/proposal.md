# Proposal

## Why

The same product is often kept in more than one place, such as one pack of pasta in the kitchen and two in the cellar.
Today, adding a product merges it into any unopened row with the same name and expiry date wherever that row is kept,
so the second location is silently lost; and taking some packs from one place to another can only be recorded by
editing the whole row or by using some and adding them again elsewhere.

## What Changes

- **The location joins the merge rule**: adding a product, or moving it to another section, adds to an existing row
  only when the name, the expiry date and the location all match, name and location ignoring case. Otherwise it is a
  row of its own, listed next to the other rows of the same product.
- **A "Move" action on each row** asks how much to move, 1 by default, and where to, suggesting the locations already
  used. That quantity is split off into the new location, added to a matching row there if one exists; moving the
  whole quantity just changes the row's location. Three packs in the cellar, two moved to the kitchen: one row of 1 in
  the cellar, one of 2 in the kitchen.

### Out of scope

- Moving part of a row to another section: the edit page still moves the whole row.
- Merging when the location is changed from the edit page: the edit page keeps saving the row as it is.
- Rows already merged before this change: their lost locations can't be recovered.
- A list of locations to manage, or a view of the stock by location.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `household-stock`: "One row per expiry date" and "Move an item to another section" take the location into account;
  new "Move part of an item to another location" requirement.

## Impact

- `internal/stock` (the merge rule and a new move operation), a move route and page in `internal/web/stock.go`, the
  section and new move templates. No schema migration.
