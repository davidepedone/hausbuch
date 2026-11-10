# Proposal

## Why

The stock is organised by section, but a household thinks of it by place too: "what is in the cellar?" has no answer
today short of opening every section and reading every row. And since a location is free text, suggested only within
one section and not at all on the iPhone, the same place easily ends up written two ways, which a view by place would
show as two places.

## What Changes

- **A "Locations" page**, reached from a tile on the home page, listing every location the stock's rows are kept in,
  across sections, with the number of rows in each. Locations differing only in case are one.
- **A page per location**, listing its rows grouped by section, each linking to the row in its section, where it can be
  changed. A row's location in a section becomes a link to its location's page.
- **Location suggestions from the whole home**: the location field suggests the locations used in any section, not only
  in its own, and shows them as choices under the field, as the category field does.

### Out of scope

- Changing quantities or editing rows on a location's page: its rows link to their section.
- Renaming a location on all its rows at once, merging two spellings, or a list of locations to manage.
- Common locations suggested before the household has any.
- Nested locations ("Cellar" containing "Cellar, shelf 2").
- The monthly check by location planned for later.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `household-stock`: adds the Locations page and a page per location; locations are suggested from the whole home and
  shown as choices under the field.

## Impact

- `internal/stock`: the locations in use with their row counts, the rows kept in one location, and the locations used
  in any section.
- `internal/web`: the two pages, a home tile, links on the rows' locations, and the location field's choices on the add
  form, the edit page, the move page and the put-away page.
- No database change.
