# Proposal

## Why

An item added to the wrong section, such as a cleaning product in the Pantry instead of Supplies, can only be moved
today by removing it and adding it again, which loses who added it and when and means typing everything anew
(issue #47).

## What Changes

- **A Section field on an item's edit page**, set to its current section.
- **Saving with another section moves the row** there with all its values, the edits made in the same form included;
  the user lands on the row in its new section.
- **Moving onto the same product**: when the new section already has an unopened row with the same name and expiry
  date, and the moved row isn't opened either, the quantities add up as when adding, and the moved row disappears.
- **The new section remembers the product** for its suggestions, category included, as after adding it there.

### Out of scope

- Moving several items at once.
- Moving from the section's list without opening the item.
- Forgetting the product in the section it left: its name stays among that section's suggestions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `household-stock`: a new "Move an item to another section" requirement.

## Impact

- `internal/stock` (moving a row inside the update), the item form and redirect in `internal/web/stock.go`, and the
  item template.
