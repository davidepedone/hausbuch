# Design

## Context

See `proposal.md` for the motivation and `specs/household-stock/spec.md` and `specs/expiry-warnings/spec.md` for the
required behaviour. The current state that shapes the approach:

- Everything is one SQLite database; migrations are embedded goose SQL files and may only add (README, development
  section).
- Each domain is a package with a `Store` over `*sql.DB` whose `Now` is the clock, replaceable in tests
  (`internal/page`, `internal/user`). `cmd/hausbuch` reads and validates environment variables in `serve` and passes
  stores to `web.NewHandler`.
- The web layer renders `layout.html` + `parts.html` + one template per screen, posts forms and redirects, and uses
  htmx for the editor's partial updates (`renderPart`). Days are shown in the process's local time, which `TZ` sets.
- The home page (`home.html`) has the page type tiles and the list of pages from `add-visual-design`.

## Goals / Non-Goals

**Goals:**

- One model and one set of handlers for both sections; a section is a value, not a copy of the code.
- Quantity changes, opening and removal without reloading the page, and an undo that needs no server-side state.

**Non-Goals:**

- Paging or searching within a section: a household's medicines and pantry are tens of rows, not thousands.
- Including items in the wiki's full-text search.

## Decisions

### A `stock` package with two tables

`internal/stock` holds the model and its `Store` (with `Now`, and the lead time of each section). A `Section` is
`medicines` or `pantry`, with a label, like `page.Type`. Migration `00006_stock.sql` adds:

- `stock_item`: `id` (AUTOINCREMENT), `section`, `name`, `name_key` (trimmed and lower-cased in Go, as `title_key`),
  `quantity` (`CHECK (quantity >= 1)`), `unit` and `location` (text, empty when not given), `expires_on` and
  `opened_on` (nullable `YYYY-MM-DD` text), `opened_days` (nullable integer), `created_by`, `created_at`,
  `changed_by`, `changed_at`; indexed on `(section, name_key)`.
- `stock_product`: the catalogue, primary key `(section, name_key)`, with `name`, `location`, `opened_days`. Every
  add, edit and undo upserts the row's product, so it always holds the last values used; removing a row leaves it.

Dates are calendar dates, stored as text: an expiry date has no time of day, and storing it as an instant would make
it depend on the time zone it was written in. "Today" is `Now()` in the local time zone, as `TZ` sets it.

Alternatives considered:
- *One table per section*: twice the queries for identical behaviour.
- *Making items wiki pages of a new type*: pages are free text; quantities, dates and warnings need fields.

### Effective expiry and warnings computed in Go

The effective expiry date and the state (expired, expiring, none) are computed in Go from the row and today, by one
function used by the sections and the home page. Each list loads all the rows of its section, which is small; the home
page loads both sections and keeps the rows with a state. Doing it in SQL would duplicate the rule in two languages.

### Adding merges into an unopened row

`Add` looks for an unopened row with the same section, `name_key` and `expires_on` (both null counting as equal) and
adds to its quantity in the same transaction; otherwise it inserts a row. Undo goes through `Add` too, so restoring a
row whose twin was added meanwhile merges them instead of creating a duplicate.

### Quantity changes as single statements

Decreasing is `UPDATE … SET quantity = quantity - 1 WHERE id = ? AND quantity > 1`, and, when no row was updated
because the quantity was 1, `DELETE … WHERE id = ? AND quantity = 1`. Both are relative to the stored value, so two
users decreasing at once both count, and neither can take a row below zero.

"Opened today" on a row with quantity 1 sets `opened_on`; on a larger quantity it decreases the row by one and inserts
a copy with quantity 1 and `opened_on` set, in one transaction.

### Pages, forms and htmx

Each section has its own routes, registered in a loop over the sections: `GET /medicines` and `GET /pantry` list the
rows with the "Add item" form above them, folded in a `<details>` once the section holds items; `POST /{section}/items`
adds; `GET`/`POST /{section}/items/{id}` edit, like pages; `POST /{section}/items/{id}/decrease`, `…/increase`,
`…/open` and `…/remove` change a row; `POST /{section}/items/restore` undoes a removal. The routes check that the row
belongs to the section in the path and answer 404 otherwise.

Adding and editing post the form and redirect, like pages. The row buttons post with htmx (`hx-post`, target the
section's list) and the server answers with the list fragment, so the page doesn't reload and the scroll position
stays; opening, which can add a row, is handled the same way.

### Undo without server state

When a row is removed, the list fragment starts with a strip "Removed Passata. Undo": a form whose hidden fields carry
the removed row's values, including its creator and creation time, and posts them to `…/items/restore`. Nothing is
kept on the server, so a removal really deletes the row, as the proposal decided. The strip hides itself after 10
seconds with a CSS animation that ends in `visibility: hidden`, which also makes it unclickable; no script is needed.

Alternatives considered:
- *Keeping removed rows for a minute and purging them*: a timer, a cleanup and a hidden state for a few seconds of
  undo.

### Suggestions and prefill

The add form's name and location fields use `<datalist>` elements filled from the section's catalogue and from the
locations of its products: the browser suggests while typing, with no script. When the name field changes, htmx asks
`GET /{section}/product?name=…` and, for a known product, replaces the location and shelf-life fields with prefilled
ones; for an unknown name it answers 204 and nothing changes.

### Home page as a dashboard

`home.html` gets, above what it has today: the "Expiring" box when there are rows to show (expired in red, expiring in
amber, the colours the interface already uses for errors and warnings), then two tiles, Medicines and Pantry, styled
like the page type tiles with an icon each (a pill box, a jar) drawn like the others in `typeIcon`'s style. The wiki
part follows under its "Pages" heading as today. The "Wiki" tile sketched while exploring is left out: the wiki is
right there below.

Each row in the box shows the name, quantity, section, the date and "expired" or "expires in N days", and links to
`/{section}#item-{id}`, where the row is.

### Configuration

`HAUSBUCH_MEDICINES_LEAD_DAYS` (default 30) and `HAUSBUCH_PANTRY_LEAD_DAYS` (default 7) are read in `serve`, validated
as whole numbers of at least 0 like `HAUSBUCH_BACKUP_KEEP`, and set on the stock store. `web.NewHandler` takes the
stock store as a new argument.

## Risks / Trade-offs

- [Many medicines print only a month and a year] → the date field takes a full date; the form's hint says to pick the
  month's last day. A month-only field would complicate every date comparison for a convenience.
- [Undo fields can be edited before posting] → a signed-in user can already change any row; the restore endpoint
  applies the same validation as adding.
- [The strip hides by CSS only] → if the animation doesn't run, the strip stays until the next action; the row is
  removed either way.
- [Datalist suggestions look different across browsers, and some phones show them only after typing] → acceptable:
  the prefill after choosing a name works regardless.

## Migration Plan

One additive migration creating two tables; the backup before a schema change covers it, and the previous release
runs on the migrated database, ignoring the new tables.
