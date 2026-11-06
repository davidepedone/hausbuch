# Proposal

## Why

Households keep medicines and food that expire, and nobody remembers what is in the cabinet or at the back of the
pantry until it is too late. hausbuch already holds what the household needs to know; it should also know what the
household has, and point out what is about to expire.

## What Changes

- **Two new sections, Medicines and Pantry**, sharing one model: a list of items, each row with a name, a quantity, an
  optional unit written freely ("packs", "bottles"), where it is kept, and an optional expiry date. In the pantry many
  items have no expiry date.
- **One row per expiry date**: the same product with two expiry dates is two rows, shown next to each other.
- **Quantity with − and +**: a row reaching zero, or deleted, is removed for good; an "Undo" is offered for a few
  seconds afterwards.
- **Shelf life after opening**: a row can say how many days the item lasts once opened. "Opened today" splits one unit
  off into a row of its own, opened that day; its effective expiry is the earlier of the printed date and the opening
  date plus that many days, or the latter alone when there is no printed date.
- **A catalogue of products**: one entry per name ever used, remembering its last location and shelf life after
  opening. It suggests names while adding an item and prefills the rest; locations are free text, suggested from those
  already used.
- **Expiry warnings**: an item is *expired* after its effective expiry date and *expiring* within a lead time of it:
  30 days for medicines and 7 for the pantry by default, each set by an environment variable.
- **The home page becomes a dashboard**: an "Expiring" box at the top, listing expired and expiring items from both
  sections and shown only when there are any; below it, tiles for the wiki, Medicines and Pantry; then the wiki's pages
  as today.
- Every signed-in user can see and change everything, as in the wiki; each row records who added and last changed it.

### Out of scope

- Notifications outside the application: a daily summary to the household's Telegram group is the next change.
- Asking a Telegram bot what is at home: a later change, after notifications.
- Whether medicines go through Telegram at all, given that their names would pass through Telegram's servers: decided
  in the notifications change, possibly sending only the pantry.
- Setting the lead times from a settings page in the application: planned for a later phase; until then they come from
  the environment.
- A lead time per item.
- A shopping list, and a "finished" list to buy items again.
- A history of what was used or thrown away, and consumption statistics.
- A history of each row's changes beyond who added and last changed it.

## Capabilities

### New Capabilities

- `household-stock`: the Medicines and Pantry sections — items, quantities, opening, removal with undo, and the
  catalogue of products used for suggestions.
- `expiry-warnings`: when an item counts as expired or expiring, the lead time of each section, and where the
  application shows them.

### Modified Capabilities

None: the wiki's page list stays on the home page, below the new parts.

## Impact

- New tables and a migration (additive, as the project's rule requires), a new package for stock items, new handlers
  and templates, and the home page template.
- Two new environment variables for the lead times, documented in the README.
- Backups and the deploy are unchanged: the items live in the same database.
