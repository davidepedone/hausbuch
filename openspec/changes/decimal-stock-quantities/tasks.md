# Tasks

## 1. Model and store (`household-stock`)

- [ ] 1.1 Add `stock.Quantity` (tenths) with parsing (dot or comma, at most one decimal, above 0) and `String`, and convert `Values.Quantity` and every existing test literal to it; verify unit tests for the accepted and refused texts of both "Add an item" quantity scenarios and for "2", "1.5" and "2.0" printing
- [ ] 1.2 Add migration `00007` with `quantity_tenths` filled from `quantity`, read the current quantity with the rule in the design, and write both columns in add, merge, update, increase, decrease and open; verify store tests for every "Change the quantity" and "Shelf life after opening" scenario, including two concurrent decreases, and a test that rows changed or inserted through the integer column alone (as the previous release does) are read from it
- [ ] 1.3 Verify that a database migrated by `00007` and written with decimal quantities opens and lists its rows with the previous release's code (the additive rule)

## 2. Web interface

- [ ] 2.1 Parse the quantity with `stock.Quantity` in the item form, make the quantity inputs of the add and edit forms `type="text" inputmode="decimal"`, and update the refusal message; verify web tests for "Quantity with a decimal", "Whole quantity" and "Missing name or quantity", and that undoing the removal of a 0.5 row brings it back with 0.5

## 3. Integration

- [ ] 3.1 With `make check` green, add, increase, decrease, open and undo decimal quantities in the browser at 390px and 1280px, typing the quantity with a comma and with a dot; verify every step behaves as the spec says and the home page's "Expiring" box shows the decimal quantity

## Workflow follow-up

- Archive the change once merged, which closes issue #29.
