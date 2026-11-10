# Design

## Context

See `proposal.md` for the motivation and the delta specs for the required behaviour. The current state that shapes the
approach:

- `stock.Store.add` checks the values, then in one transaction either merges them into a matching row (`mergeInto`)
  or inserts a row, and remembers the product (`remember`). Quantities are stored in tenths (`quantity_tenths`).
- `stock_product` remembers, per section, each name used with the category, location and shelf life it last had;
  `GET /<section>/product?name=` answers the add form with those values as a fragment, and `productfields.js` copies
  the fragment's `#product-fields [name]` values into the fields the user hasn't typed in.
- A row's buttons post to `/<section>/items/<id>/<action>` and answer, with HTMX, the section's `items` part; the
  undo offer is a form in that part carrying the removed row's values, posting to `/<section>/items/restore`.

## Goals / Non-Goals

**Goals:**

- Putting away is one transaction: the item is never added to the stock while its entry stays on the list, or the
  other way round.
- The list works without JavaScript, like the sections: every action is a form post with a redirect, enhanced by HTMX.

**Non-Goals:**

- No change to how the stock adds, merges or remembers products.

## Decisions

### The list lives in package `stock`

The shopping list is a new file, `internal/stock/shopping.go`, on the same `Store`, rather than a package of its own.
Putting away has to add to `stock_item` and delete from the list in one transaction, and adding from a row needs the
row's section and values: in the same package, `PutAway` calls the body of `add` with its own transaction.

*Alternative considered*: an `internal/shopping` package calling `stock.Store.Add` and then deleting the entry. It would
keep the files apart, but either loses the single transaction or makes `stock` export a transaction-taking variant of
`add` for one caller.

### One table

```sql
CREATE TABLE shopping_entry (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    name_key        TEXT    NOT NULL,
    section         TEXT    NOT NULL DEFAULT '',   -- '' for no section
    quantity_tenths INTEGER,                       -- NULL when not given
    unit            TEXT    NOT NULL DEFAULT '',
    shop            TEXT    NOT NULL DEFAULT '',   -- '' for none
    bought          INTEGER NOT NULL DEFAULT 0,
    created_by      INTEGER NOT NULL REFERENCES app_user (id),
    created_at      INTEGER NOT NULL
);

-- The shop each name was last listed with, kept when its entries are gone.
CREATE TABLE shopping_product (
    name_key TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    shop     TEXT NOT NULL
);
```

`name_key` is computed with `nameKey`, as in the stock. One entry per product is kept in Go, not by a unique index:
adding, ticking and unticking go through one `merge(tx, entry, bought)` that looks for an entry with the same
`name_key`, `section` and `bought`, and either combines the quantities into it (deleting the moving entry, for a tick)
or writes the entry as it is. The list is ordered in Go: by `bought`, then section in `Sections` order with '' last,
then `name_key`.

`shopping_product` is keyed by name alone, not by section, because entries without a section (bread) need a shop too
and a product is usually bought at the same place whatever section it is kept in. Adding an entry with a shop upserts
its row; adding one without a shop takes the row's shop. Shop chips are built in Go from the entries' shops, merged
ignoring case with `nameKey` and labelled with the first spelling met.

*Alternative considered*: a `shop` column on `stock_product`. It would reuse the stock's product memory, but misses
entries without a section and products kept in two sections.

### Filtering and refreshing

The chips reuse `listfilter.js`: each entry carries `data-chip` with its shop's key. An entry without a shop gets a new
`data-chip-any` attribute that the script treats as matching every chip, a two-line change that leaves the sections'
filtering as it is.

The list's `entries` part refreshes itself with `hx-get="/shopping/entries" hx-trigger="every 10s"` on its own
container, outside the add form, so the form and what was typed in it are untouched. `listfilter.js` already
re-applies the chosen chip after every HTMX swap, and the chips are inside the refreshed part so that a new shop
appears. Polling stops by itself when the page is closed; a hidden tab keeps polling, which at one small request every
ten seconds is cheaper than adding visibility handling.

*Alternative considered*: server-sent events pushing each change. Instant, but it needs a long-lived connection per
open list, a fan-out of changes inside the server and reconnect handling, for a gain of a few seconds.

### Store operations

`AddEntry`, `Entries`, `Tick(id, bought bool)`, `RemoveEntry`, `ClearBought` (bought entries with section ''), and
`PutAway(id, userID, Values)`. `PutAway` checks the entry is bought and has a section, adds the values to that section
as `add` does, and deletes the entry, all in one transaction; a refused value (`ErrEmptyName`, `ErrQuantity`, …)
leaves both untouched. `AddRowToList(section, id)` reads the row and adds an entry with its name, unit and section, no
quantity and the remembered shop. `ListNames` (the names of every section and of `shopping_product`), `Shops` and a
`ListProduct(name)` returning the one section where the name is known, if exactly one, and its remembered shop serve
the suggestions.

### Routes and pages

- `GET /shopping`: the list page, with the add form at the top (Name with every section's and the list's names as
  suggestions, Quantity, Unit, Section as a select with "None", Shop with the shops used as suggestions), then the
  `entries` part: the shop chips, the groups to buy, then Bought with the clearing action when some bought entry has no
  section. `GET /shopping/entries` answers the `entries` part alone, for the refresh.
- `POST /shopping/entries`, `POST /shopping/entries/{id}/{action}` with `tick`, `untick` and `remove`, and
  `POST /shopping/clear`: answer the list's `entries` part with HTMX, or redirect to `/shopping`.
- `GET /shopping/product?name=`: the section the name is known in, when exactly one, and its remembered shop, as a
  `#product-fields` fragment holding those fields, so that `productfields.js` fills them in unchanged; 204 when there
  is neither.
- `GET` and `POST /shopping/entries/{id}/put-away`: a page with the section's add fields, filled in from the entry and
  from `Product`; a refusal renders it again with what was entered; success redirects to
  `/<section>#item-<id>` of the row that got the item. The add form's field templates (`productFields`,
  `expiryField`) move to `parts.html` if the new page can't reach them where they are.
- On a section: the row action "Add to shopping list" posts to `/<section>/items/<id>/list`, a new case of
  `changeItem`; the undo offer gets a second button posting the same hidden values to `/<section>/items/to-list`.
  Both answer the `items` part with a short status "Added <name> to the shopping list.", shown like the undo offer.
- The home page gets a fourth tile, "Shopping list", in the sections' grid, which becomes two columns on narrow
  screens if four don't fit.

*Alternative considered*: putting away inline in the list, with an expiry field on each bought entry. It would save a
page, but a bought entry may need any field changed (location, quantity), and the stock's add form already handles
all of them and their errors.

## Risks / Trade-offs

- [Two people tick within the same ten seconds] → each tick is applied on the server as it arrives; a tick on an
  entry someone else already removed or merged answers 404 for a plain post and, with HTMX, the current list.
- [A refresh lands while a user taps a checkbox] → the tap posts and its answer swaps the list again; at worst the tap
  hits the old list a moment before the swap and the post still names the right entry id.
- [The undo offer disappears after a few seconds, so its "Add to shopping list" can be missed] → the row action stays
  available while rows of the product remain, and the product can always be typed into the list with suggestions.
- [Merging on tick changes which entry id survives] → the list is re-rendered after every action, so no page keeps a
  stale id for long; a post on a missing id answers 404 like the stock's rows.

## Migration Plan

A new goose migration creating `shopping_entry` and `shopping_product`; its Down drops them. No existing data changes.
