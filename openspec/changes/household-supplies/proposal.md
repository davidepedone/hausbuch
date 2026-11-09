# Proposal

## Why

Cleaning products and other household supplies have no section of their own, so they end up in the Pantry, which is
meant for food: they make its list longer and their expiry dates, which matter little, get mixed with the food's
warnings (issue #27).

## What Changes

- **A third stock section, Supplies**, next to Medicines and Pantry, at `/supplies`, with everything the other two
  sections offer: adding, editing, changing quantities, removing with undo, opening, suggestions from products already
  used, expiry warnings.
- **Its own tile on the home page**, next to the Medicines and Pantry tiles, with its own icon.
- **Its own expiry lead time**, 7 days unless the installation configures another with
  `HAUSBUCH_SUPPLIES_LEAD_DAYS`.
- The home page's "Expiring" box lists the rows of every section, Supplies included.

### Out of scope

- Moving items already recorded in the Pantry to Supplies: users move them by hand, removing and adding them again.
- Sections defined by the installation or by users: the sections stay fixed in the code.
- Hiding Supplies rows from the home page's "Expiring" box, or any per-section choice of what the box shows.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `household-stock`: "Medicines and Pantry sections" becomes three sections, each reached from the home page.
- `expiry-warnings`: "Lead time per section" gains the Supplies lead time; "Expiring box on the home page" lists every
  section instead of both.

## Impact

- `internal/stock` (the list of sections, their labels and default lead times), `cmd/hausbuch` (the new environment
  variable), the section icon in `internal/web/templates/parts.html` and the section tiles in `home.html`.
- README: the new section in the introduction and the new environment variable.
- No migration: rows already carry their section as text.
