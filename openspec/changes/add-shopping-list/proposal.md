# Proposal

## Why

The household keeps track of what it has at home, but not of what it needs to buy: when the last pack of something is
used, nothing remembers it, and after the shopping every product has to be typed into its section again from scratch.
A shopping list tied to the stock closes that loop and takes most of the typing out of keeping the stock up to date.

## What Changes

- **A shopping list for the whole household**, reached from a new tile on the home page. Each entry has a name, an
  optional quantity and unit, and optionally one of the stock sections; an entry without a section is something not
  kept in stock, such as bread.
- **Adding to the list** by hand, with the product names already used suggested; a name already on the list, not yet
  bought, in the same section, stays one entry.
- **Shops**: an entry can say where it is bought (a supermarket, the pharmacy, a hardware store). The list stays one,
  with a chip per shop to show only what to buy there; a product remembers the shop it was last bought at, so it is
  filled in the next time.
- **Kept up to date while open**: the list refreshes itself every few seconds, so what another user ticks or adds
  while out shopping shows up without reloading.
- **From the stock**: each row gets an "Add to shopping list" action, and the offer to undo a removal, shown when the
  last one is used or a row is removed, also offers to add the product to the list.
- **Shopping in two steps**: ticking an entry while shopping marks it as bought, and unticking it brings it back. At
  home, "Put away" on a bought entry with a section opens that section's add form filled in with the entry's name,
  quantity and unit and the category, location and shelf life after opening the product last had, so only the expiry
  date is left to enter; saving adds the row to the stock and takes the entry off the list.
- **Removing entries**: any entry can be removed, and one action clears the bought entries that have no section.

### Out of scope

- Adding a product to the list on its own when it runs out, or a minimum quantity to keep at home.
- Separate lists to create and name, a list of shops to manage, or ordering the list by the shop's aisles.
- Instant updates: another user's changes show up at the next refresh, a few seconds later.
- Using the list without a connection.
- Shopping list entries in the unified search.
- Editing an entry: it can be removed and added again.
- Putting away a bought entry into a section other than its own: it can be removed and the product added to the right
  section.
- Barcode scanning while putting things away.

## Capabilities

### New Capabilities

- `shopping-list`: the household's shopping list, its entries and their shops, ticking them as bought, and putting
  bought entries away into the stock.

### Modified Capabilities

- `household-stock`: "Undo a removal" also offers to add the removed product to the shopping list; new "Add a row to
  the shopping list" requirement.

## Impact

- A new migration for the list's table and the shop each product was last bought at.
- The list's operations in `internal/stock`, with a "put away" operation that adds to the stock and removes the entry
  in one transaction.
- New routes, a list page and a put-away page in `internal/web`; the section template gets the row action and the
  extra button in the undo offer; the home page gets a fourth tile.
