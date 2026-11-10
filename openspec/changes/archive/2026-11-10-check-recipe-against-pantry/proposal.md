# Proposal

## Why

Recipes and the pantry live in the same app, but nothing connects them: before cooking, someone has to read the
ingredients and go and look in the cupboards, then type what is missing into the shopping list. The app already knows
both, so it can do the looking and leave only the choosing.

## What Changes

- **"Check the pantry" on recipe pages**: a page listing the recipe's ingredients, the items under its "Ingredients"
  heading, each shown as in the pantry (with where it is kept, how much there is and, when the row hasn't changed for three
  months, how long ago it last did), already on the shopping list, or missing.
- **Matching by name, ignoring case and quantities**: an ingredient is in the pantry when a pantry row's name and the
  ingredient share a word, leaving out numbers and filler words like "of" or "di", so "200 g flour" finds the "Flour"
  row and "1 tbsp oil" the "Olive oil" one. Only presence is checked, never whether there is enough.
- **Adding to the shopping list, never on its own**: missing ingredients come ticked, the others unticked; each has its
  name in an editable field, filled in with the ingredient as written, so "200 g flour" can become "Flour". "Add to
  shopping list" adds the ticked ones as Pantry entries, with the usual merging, and opens the shopping list.

### Out of scope

- Comparing quantities, or scaling a recipe by its servings.
- Plurals, synonyms and typos ("eggs" against an "Egg" row): left to the smarter search planned separately.
- Looking in sections other than Pantry, or treating expired or opened rows differently from the others.
- Taking the ingredients out of the stock when the recipe is cooked.
- Ingredients written anywhere but as list items under an "Ingredients" heading, or on pages of another type.
- Remembering which ingredients a user chose not to add.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `shopping-list`: adds checking a recipe against the pantry and adding its missing ingredients to the list.

## Impact

- `internal/markdown`: the items listed under a heading of a page's body.
- `internal/stock`: matching ingredients against pantry rows and list entries.
- `internal/web`: the check page and its form, a "Check the pantry" link on recipe pages.
- No database change.
