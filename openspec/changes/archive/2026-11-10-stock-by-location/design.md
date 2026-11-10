# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. A row's location is free text in
`stock_item.location`; `stock_product` remembers, per section and name, the location a product was last kept in, and
`Store.Locations(section)` reads that table for the add form, the edit page, the move page and the put-away page. The
category field already shows its suggestions as chip buttons through the `categoryField` part and `suggestions.js`.
The home page has one tile per section plus the shopping list.

## Goals / Non-Goals

**Goals:**

- One spelling per place, encouraged by offering the places already used everywhere a location is entered.
- No new table: places are what the rows say.

**Non-Goals:**

- Acting on rows from a location's page.

## Decisions

### Locations from the rows, keyed ignoring case

`Store.Places(ctx)` returns each location in use with its row count, grouping the rows of every section by
`nameKey(location)` and labelling each group with its smallest spelling, the first in alphabetical order. The grouping
is in Go rather than SQL because SQLite's `lower` folds ASCII only, while names elsewhere compare with Go's
`strings.ToLower`. `Store.InLocation(ctx, location)` returns the rows whose location has the same key, ordered by
section and then as each section orders its rows. `Store.Locations` loses its section and reads `stock_product` for
every section, one per location ignoring case in the same way, for the suggestions.

*Alternative considered*: counting from `stock_product`. It remembers where products were last kept, not what is there
now.

### Pages at `/locations` and `/locations/{name}`

The name travels path-escaped (`url.PathEscape`), so a location holding "/" stays one path segment; `r.PathValue`
returns it unescaped. A location with no rows answers 200 with a message rather than 404: it is reached from a link
that was right when the page holding it was shown.

### Rows on a location's page link to their section

Each row shows name, quantity, unit and expiry state and links to `/{section}#item-{id}`, as the home page's Expiring
box does. The row actions of a section answer with that section's list part; reusing them here would need a second
answer shape for every action, for a page meant to be looked at.

### One part for a field with choices

`categoryField` becomes `choiceField`, filled by a template function `field(name, label, value, list, choices)` that
returns the struct the part needs; the category and location fields both use it, and `suggestions.js` is unchanged.

### Home tile

A fifth tile, "Locations", with the "where it is" page type's icon. On a phone the tiles are two per row, so the fifth
spans the row; from `sm` up they are five in a row.

## Risks / Trade-offs

- [Two spellings of one place still show as two locations, "Cellar" and "Cellar shelf"] → accepted: the page makes
  them visible, and the choices under the field make the next row use the existing one.
- [A long list of locations on the home tile's page] → it is one line per place; a household has a few dozen at most.
