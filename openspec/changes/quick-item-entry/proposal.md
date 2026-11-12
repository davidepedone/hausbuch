# Proposal

## Why

Stock stays useful only if adding to it is quick, and today it isn't, especially when loading many items at once after
shopping or while tidying the cellar. Most packages print only a month and a year, but the expiry field asks for a
full date: the user has to pick the month's last day. A known product brings back its category, location and shelf
life, but not its unit. And after each "Add", the "Add item" form starts from scratch, so the user has to open it again
and type the same location again for each item.

## What Changes

- **Expiry as month and year**: wherever an item's expiry date is entered (adding an item, editing it, putting away a
  shopping list row), the field asks for a month and a year, and the row expires on that month's last day. An "Exact
  day" choice switches the field to a full date, for fresh food. A row whose expiry date is not a month's last day
  opens with the full date when edited.
- **The unit of a known product**: choosing a product name already used in the section also fills in the unit it last
  had, like its category, location and shelf life after opening already are.
- **Adding many items in a row**: after an item is added, the "Add item" form stays open and empty, with the cursor on
  the name, and says which item was added. The location of the item just added stays filled in, and choosing a known
  product name doesn't replace it.

### Out of scope

- Scanning barcodes.
- The monthly check of what is kept in a location.
- Remembering a unit in the shopping list's own suggestions.
- Other quick-entry ideas from the plan, such as remembering the quantity or the expiry date of a product.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `household-stock`: the expiry date is entered as a month and a year unless an exact day is chosen; a known product's
  unit is remembered and filled in; the add form stays open after adding, keeping the location.

## Impact

- `internal/web/templates/parts.html` (`expiryField`, `productFields`), `section.html`, `item.html`, `putaway.html`.
- `internal/web/stock.go` and `shopping.go`: reading the month and year, the add form after adding, the product
  answer with the unit.
- `internal/stock`: the unit in `stock_product`, a new migration.
- `internal/web/static/productfields.js`: the unit field, and the kept location not replaced.
- A small script for the "Exact day" switch.
