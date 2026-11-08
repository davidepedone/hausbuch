# Proposal

## Why

Quantities in the stock sections are whole numbers of at least 1, but much of what a household keeps is measured by
weight or volume: half a kilo of flour or a litre and a half of oil can't be recorded today (issue #29).

## What Changes

- **Quantities with one decimal**: any quantity above 0 with at most one decimal is accepted, typed with either a dot
  or a comma ("0.5", "1,5"). Whole numbers are shown without decimals ("2", not "2.0").
- **− and + still change a quantity by 1**: − on a row with a quantity of 1 or less removes it, with the usual undo.
- **Opening**: marking a row with a quantity above 1 as opened splits 1 off into a row of its own, as today; a row with
  a quantity of 1 or less is marked opened as a whole.
- Adding to an unopened row with the same name and expiry date keeps summing quantities, decimals included.

### Out of scope

- A step for − and + other than 1, or a step per item or per unit.
- More than one decimal.
- Conversions between units ("500 g" and "0.5 kg" stay unrelated rows and values).
- Showing quantities with a decimal comma: the interface is in English, so decimals are shown with a dot.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `household-stock`: "Add an item" accepts quantities above 0 with one decimal; "Change the quantity" removes a row
  with 1 or less; "Shelf life after opening" marks a row of 1 or less as opened as a whole.

## Impact

- A migration adding a column for the quantity in tenths, filled from the current one; the current column stays
  written, so the previous release keeps working on a migrated database.
- `internal/stock` (validation, add and merge, increase, decrease, open, restore), the quantity parsing and messages in
  `internal/web/stock.go`, and the quantity fields and displays in the section, item and home templates.
