# Tasks

## 1. Store

- [x] 1.1 Add the migration for `shopping_entry` and `shopping_product`; verify the migration test (or a store test opening a fresh database) creates the tables and its Down drops them
- [x] 1.2 Add `AddEntry`, `Entries`, `RemoveEntry` and `ClearBought` in `internal/stock/shopping.go`, with the shared `merge` and the remembered shop; verify store tests for "Grouped list", "Bought entries apart", "Entry for the stock", "Entry without a section", "Refused entry", "Added twice", "Added again without a quantity", "Already bought", "Different section", "No longer needed", "Clear", "Nothing to clear", "Bought there again" and "Shop changed"
- [x] 1.3 Add `Tick`; verify store tests for "Tick", "Untick", "Ticked twice" and an unknown entry
- [x] 1.4 Add `PutAway`, sharing `add`'s body inside one transaction; verify store tests for "Saved", "Added to an existing row", "Refused form" (entry still on the list, no row added), an entry not bought or without a section refused, and an unknown entry
- [x] 1.5 Add `AddRowToList`, `ListNames`, `Shops` and `ListProduct`; verify store tests for "Running low", "Already on the list", "Known product", "Name used in two sections", "Shop suggestions" and a row added with its remembered shop

## 2. Shopping list pages

- [x] 2.1 Add the list page, its add form and the entry actions (`tick`, `untick`, `remove`, clear) with HTMX and redirect answers; verify web tests for each action with and without `HX-Request`, "Empty list", "Shared by the household", the refusal answered with the message and the entered values, and a 404 for an unknown entry
- [x] 2.2 Add `GET /shopping/product` and wire the add form's section and shop to `productfields.js`; verify web tests for a name known in one section (the fragment holds it), one with a remembered shop, and one with neither (204)
- [x] 2.3 Add the shop chips with `data-chip-any` support in `listfilter.js`, and the `GET /shopping/entries` refresh every 10 seconds on the `entries` part; verify web tests for "Same shop, different case", "No shops" and the part's refresh attributes, the sections' existing filter tests still green, and in the browser "At the pharmacy", "Bought by someone else" (two sessions) and "Typing meanwhile"
- [x] 2.4 Add the put-away page and its post, moving the shared field templates to `parts.html` if needed; verify web tests for "Only the expiry left", "Entry without a quantity", "No section" (no put-away link), the redirect to the receiving row and the refusal answered with the page
- [x] 2.5 Add the "Shopping list" tile to the home page; verify the web test for "Tile on the home page" and the existing home page tests still green

## 3. From the stock

- [x] 3.1 Add the row action "Add to shopping list" and the undo offer's second button with their routes and the "Added … to the shopping list." status; verify web tests for "Running low", "Confirmed on the row", "Used the last one" (row stays removed, entry added) and the existing undo tests still green

## 4. Integration

- [x] 4.1 With `make check` green, in the browser at 390px and 1280px: use the last bottle of a product and add it to the list from the undo offer, add a row running low, add an entry with no section and entries with two shops, filter by one shop, tick an entry from a second session and see it move within ten seconds, tick everything, put away an entry filling only the expiry, clear the rest; verify every step behaves as the specs say, the four home tiles fit, and nothing overflows sideways

## Workflow follow-up

- Archive the change once merged.
