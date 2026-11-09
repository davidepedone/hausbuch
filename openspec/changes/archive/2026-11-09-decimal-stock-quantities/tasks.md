# Tasks

## 1. Model and store (`household-stock`)

- [x] 1.1 Add `stock.Quantity` (tenths) with parsing (dot or comma, at most one decimal, above 0) and `String`, and convert `Values.Quantity` and every existing test literal to it; verify unit tests for the accepted and refused texts of both "Add an item" quantity scenarios and for "2", "1.5" and "2.0" printing
- [x] 1.2 Add migration `00007` rebuilding `stock_item` with `quantity_tenths`, and use it in add, merge, update, increase, decrease and open; verify a migration test that existing rows keep their values with quantities times ten and the index exists, and store tests for every "Change the quantity" and "Shelf life after opening" scenario, including two concurrent decreases

## 2. Web interface

- [x] 2.1 Parse the quantity with `stock.Quantity` in the item form, make the quantity inputs of the add and edit forms `type="text" inputmode="decimal"`, and update the refusal message; verify web tests for "Quantity with a decimal", "Whole quantity" and "Missing name or quantity", and that undoing the removal of a 0.5 row brings it back with 0.5

## 3. Integration

- [x] 3.1 With `make check` green, add, increase, decrease, open and undo decimal quantities in the browser at 390px and 1280px, typing the quantity with a comma and with a dot; verify every step behaves as the spec says and the home page's "Expiring" box shows the decimal quantity

## Workflow follow-up

- Archive the change once merged, which closes issue #29.
- When cutting the release that contains it, add to its GitHub release notes that rolling back needs the database copy saved before the migration.
