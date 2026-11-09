# Proposal

## Why

The search box in the header looks only into wiki pages, so "do we still have X?" can't be answered without opening
each stock section and scrolling it, and the stock sections have no search of their own (issue #35).

## What Changes

- **The header's search finds stock items too**: an item matches when every searched word is part of its name,
  ignoring case and accents, so "pas" finds "Passata" and "Pasta".
- **Results grouped by where they come from**: pages first, then one group per stock section with matches, in the
  sections' order; a group without matches is not shown.
- **Each stock result** shows its name, quantity, unit, category, location and effective expiry date, marked when
  expired or expiring, and leads to its row in its section.
- **Results while typing**: as the user types in the header's box, a panel under it shows the results, grouped the
  same way; clearing the box (its ✕) or pressing Escape closes the panel and leaves the screen as it was, and Enter
  opens the full results page as today.
- **Page titles match on part of a word**, like stock names, so that a page shows up while its title is still being
  typed ("irri" finds "Irrigation"); bodies still match whole words and their forms.
- **The box is labelled "Search"** instead of "Search pages".
- With no result at all, the search still offers to create a page titled with the searched text.

### Out of scope

- Searching stock items by category, location or any field other than the name.
- Word forms for stock items ("passate" doesn't find "Passata"): names are matched by their text.
- A filter by name inside the lists: it was dropped from #28 because it read as a second search box; results while
  typing in the header's box take its place, so that there is one text field for finding things.
- Acting on stock items from the results (using one, opening it): results only lead to the row.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `page-search`: new "Search stock items by name" and "Results while typing" requirements; "Search pages by words"
  also matches parts of titles; "Search results" groups pages and stock items and says what a stock result shows;
  "Search is always at hand" names the box "Search".

## Impact

- `internal/stock` (a search over every section's rows), `internal/page` (title matches), `internal/web/search.go`,
  the search and layout templates, and a small script closing the results panel.
- `golang.org/x/text`, already an indirect dependency, becomes a direct one to strip accents.
