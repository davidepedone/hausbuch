# Tasks

## 1. Model and store (`household-stock`, `expiry-warnings`)

- [ ] 1.1 Add migration `00006_stock.sql` with `stock_item` and `stock_product`, and the `internal/stock` package with `Section` (medicines, pantry, with labels) and a `Store` with `Now`; verify the migration applies in `make check` and that a database migrated by it opens with the previous release's code (the additive rule)
- [ ] 1.2 Implement adding (validation, merge into an unopened row with the same name ignoring case and expiry date, catalogue upsert), editing, listing a section in the specified order, and getting a row; verify store tests for every "Medicines and Pantry sections", "Add an item", "One row per expiry date" and "Edit an item" scenario
- [ ] 1.3 Implement increase, decrease (removing at 1), remove, and "opened today" (marking or splitting one off); verify store tests for every "Change the quantity", "Remove an item" and "Shelf life after opening" scenario, including two concurrent decreases
- [ ] 1.4 Implement the effective expiry date and the state of a row given today and the section's lead time, and the list of expired and expiring rows of both sections in the home page's order; verify unit tests for every "Effective expiry date" and "Expired and expiring items" scenario, with dates around a time zone's midnight
- [ ] 1.5 Implement the catalogue lookups (names and locations of a section, a product by name); verify store tests for both "Suggestions from products already used" scenarios at store level

## 2. Configuration

- [ ] 2.1 Read and validate `HAUSBUCH_MEDICINES_LEAD_DAYS` and `HAUSBUCH_PANTRY_LEAD_DAYS` in `serve` (defaults 30 and 7), set them on the stock store and pass the store to `web.NewHandler`; verify tests for the "Lead time per section" scenarios, including the refusal to start
- [ ] 2.2 Document both variables in the README's configuration table and mention the two sections in its description; verify the table matches the defaults in the code

## 3. Sections in the web interface

- [ ] 3.1 Build `GET /medicines` and `GET /pantry` with the add form (name and location with suggestions, quantity, unit, expiry date with the month-only hint, shelf life after opening), adding and editing with redirects, rows marked expired or expiring with their effective date, and 404 for a row of another section; verify web tests for the section, add, edit and "Warnings in the sections" scenarios
- [ ] 3.2 Add the htmx row actions (decrease, increase, opened today, remove) answering with the list fragment, and the undo strip with its hidden fields, restore endpoint and 10-second CSS hiding; verify web tests for the "Change the quantity", "Remove an item" and "Undo a removal" scenarios, and in a browser that the strip disappears and can no longer be clicked
- [ ] 3.3 Add the product prefill endpoint and wire it to the name field; verify web tests for "Buy something again" (known name prefills, unknown name changes nothing) and in a browser that choosing a suggested name fills the fields

## 4. Home page

- [ ] 4.1 Add the "Expiring" box and the Medicines and Pantry tiles, with their icons, above the wiki part of the home page; verify web tests for both "Expiring box on the home page" scenarios and that the existing home page tests still pass

## 5. Integration

- [ ] 5.1 Build the image and, with a fresh volume and `TZ` set, add items in both sections, open one of several, use the last one and undo it, and check the home page box; screenshot the home page, both sections, the add form and the edit form at 390px, 820px and 1280px; verify nothing overflows horizontally and every step behaves as the specs say

## Workflow follow-up

- Archive the change once merged.
