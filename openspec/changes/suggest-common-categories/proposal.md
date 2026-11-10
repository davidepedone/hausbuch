# Proposal

## Why

A section's category field suggests only the categories already used in it, so a household that starts from an empty
section gets no suggestions and no idea of what a category is for (issue #59).

## What Changes

- **Common categories suggested per section**, next to the ones already used, wherever a category is entered: the add
  form, the edit page and the put-away page of the shopping list.
  - Pantry: Fresh, Long-life, Frozen, Drinks
  - Medicines: Pain & fever, Cold & flu, Stomach, First aid
  - Supplies: Cleaning, Laundry, Paper & bags, Personal care
- They are only suggestions: nothing is stored, a suggested category that no row has never appears as a group or a
  chip, and categories stay optional and free text.
- A used category with the same name as a suggested one, ignoring case, is suggested once, with the household's own
  spelling.

### Out of scope

- A page to create, rename or delete categories; renaming a category on all its rows at once.
- Hiding the common suggestions, or changing them per installation.
- Suggestions in another language than the interface's.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `household-stock`: "Suggestions from products already used" also suggests each section's common categories.

## Impact

- `internal/stock`: the common categories per section, merged into `Categories`.
- No change to the web layer, the templates or the database: every form already takes its suggestions from
  `Categories`.
