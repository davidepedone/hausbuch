# Proposal

## Why

The search box in the header looks only into wiki pages, so "do we still have X?" can't be answered without opening
each stock section and scrolling it, and the stock sections have no search of their own (issue #35).

## What Changes

- **The header's search finds stock items too**: an item matches when every searched word is part of its name,
  ignoring case and accents, so "pas" finds "Passata" and "Pasta". Pages are searched as today.
- **Results grouped by where they come from**: pages first, then one group per stock section with matches, in the
  sections' order; a group without matches is not shown.
- **Each stock result** shows its name, quantity, unit, category, location and effective expiry date, marked when
  expired or expiring, and leads to its row in its section.
- **The box is labelled "Search"** instead of "Search pages".
- With no result at all, the search still offers to create a page titled with the searched text.

### Out of scope

- Searching stock items by category, location or any field other than the name.
- Word forms for stock items ("passate" doesn't find "Passata"): names are matched by their text.
- A filter by name inside the lists: it was dropped from #28 because it read as a second search box, and this change
  makes it unnecessary.
- Acting on stock items from the results (using one, opening it): results only lead to the row.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `page-search`: a new "Search stock items by name" requirement; "Search results" groups pages and stock items and
  says what a stock result shows; "Search is always at hand" names the box "Search".

## Impact

- `internal/stock` (a search over every section's rows), `internal/web/search.go`, the search and layout templates.
- `golang.org/x/text`, already an indirect dependency, becomes a direct one to strip accents.
