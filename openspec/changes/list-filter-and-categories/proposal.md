# Proposal

## Why

The stock sections and the wiki's page list only grow longer, and the only way to find something is to scroll; the
stock sections have no way to divide their items, which each household would divide its own way, for example fresh
and long-life food (issue #28).

## What Changes

- **A common list filter**: at the top of the wiki's page list and of each stock section, a row of chips that narrow
  the list down, one chip at a time, with an "All" chip to clear it. A list the chosen chip leaves empty says so.
- **Categories in the stock sections**: each item may have a category, typed freely, with suggestions from the
  categories already used in that section. Categories differing only in case are the same.
- **Grouped stock lists**: a section's list is grouped by category, in alphabetical order, with items without one in
  an "Other" group at the end; within a group, the current order by name. A section where no item has a category
  shows its list as today, with no groups and no chips.
- **The chips**: in a stock section, its categories, plus "Other" when some items have none; in the wiki, the page
  types, replacing the page type tiles on the home page.
- **The category is remembered** with the product's location and shelf life after opening, and filled in when a known
  name is chosen.

### Out of scope

- Searching across sections and pages from one place: that is issue #35.
- A field filtering a list by name: next to the header's search box it reads as a second search, so finding an item
  by name is left to the header's search, which #35 extends to the stock sections.
- Choosing several chips at once.
- Categories for wiki pages, or categories shared between sections.
- Renaming or merging categories in one go: a category changes by editing its items.
- Keeping the filter when leaving the list and coming back.

## Capabilities

### New Capabilities

- `list-filter`: the chips shared by the lists, and how they narrow a list.

### Modified Capabilities

- `household-stock`: a new "Group by category" requirement; "Add an item" gains the optional category; "Undo a
  removal" brings the category back; "Suggestions from products already used" remembers and suggests it.
- `wiki-pages`: "List pages" filters by type with the list filter's chips instead of the page type tiles.

## Impact

- A migration adding a `category` column to the stock items and remembered products, with an empty default: the
  previous release keeps working on a migrated database.
- `internal/stock` (values, add, update, restore, remembering, suggestions, grouping), the item form in
  `internal/web/stock.go`, and the section, item and home templates.
- A new script for the filter, loaded by the pages that have a list; the page list's `?type=` filter on the server is
  replaced by the chips.
