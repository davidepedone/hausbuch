# Design

## Context

See `proposal.md` for the motivation and the delta specs for the required behaviour. The current state that shapes
the approach:

- The home page filters pages by type on the server, through `/?type=<type>` links on the type tiles and
  `page.Store.List(ctx, type)`.
- A stock section's list (`#items`) is re-rendered by the server and swapped in by HTMX after every row action
  (decrease, increase, open, remove, undo).
- The location and the shelf life after opening are remembered per product in `stock_product` and filled in by the
  `productFields` fragment when a known name is chosen.
- `editor.js` keeps its logic in pure functions tested in goja from Go, with the DOM wiring around them.
- Lists are household-sized: tens of entries, rarely a few hundred.

## Goals / Non-Goals

**Goals:**

- One filter script and one markup pattern for both lists.
- No request to the server while filtering.

**Non-Goals:**

- Working without JavaScript: without it the lists show everything, as today, and the chips do nothing.

## Decisions

### Filtering in the browser

A new `static/listfilter.js` filters the entries already on the page. The markup marks the filter's chips, each entry
with `data-chip`, and each group, so that a group with no entry left is hidden; a "Nothing matches" line is shown when
no entry is. Chips are buttons with `aria-pressed`.

In a stock section the chips are part of `#items`, since they follow its categories, so an HTMX swap replaces them:
the filter's box, outside `#items`, remembers the chosen chip's key, and the script re-applies it on
`htmx:afterSwap`, choosing "All" when the new list no longer has that chip.

*Alternative considered*: chips as links filtering on the server (`?category=`). Every row action would then have to
carry the filter along to render the same filtered list.

### No name field

A field filtering the list by name was considered and dropped: right under the header's search box it reads as a
second search. Finding an item by name is left to the header's search, which #35 extends to the stock sections.

### The page list's type filter moves to the chips

The type tiles become chips, each with its type's icon, and `?type=` is no longer read: `page.Store.List` loses its
type parameter, and the heading stays "Recently changed". A saved `/?type=…` link opens the full list.

### Categories stored with the item and the product

Migration `00008` adds `category TEXT NOT NULL DEFAULT ''` to `stock_item` and `stock_product`; only adding columns,
it leaves a database the previous release can still use. `stock.Values` gains `Category`, trimmed like the other
texts, and it flows through add, update, restore, `remember`, `Product` and a new `Categories(section)` suggestion
query, distinct ignoring case, like `Locations`. Adding to an existing row (same name and expiry date) leaves that
row's category as it is, as it does with its location; the product remembers the new one.

The category field joins `productFields`, so that choosing a known name fills it in with the location and shelf life.

### Grouping in Go

`stock.GroupItems(items)` returns the groups of a listed section: key is the category lower-cased, label the spelling
of the group's first item, groups by key, the "Other" group (empty key) last, items keeping the list's order. With no
category at all it returns one group with an empty key, and the template then shows neither headings nor chips. The
chips are the groups.

## Risks / Trade-offs

- [A category typed as "Other" shows as a second group labelled "Other"] → Accepted: it is a user's own choice and
  visible at once; the empty key keeps the two apart for filtering.
- [Large lists filter slower in the browser] → At household sizes, hiding a few hundred elements per keystroke is
  instantaneous.
