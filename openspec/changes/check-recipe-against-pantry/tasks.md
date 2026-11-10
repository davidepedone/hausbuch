# Tasks

## 1. Reading and matching

- [x] 1.1 Add `markdown.ListUnder`; verify tests for "Only the Ingredients list", "No ingredients", any heading level and
  case, nested and numbered lists, and `[[Title]]` links in an item
- [x] 1.2 Add the word matching and `stock.Check`; verify tests for "In the pantry", "Missing", "Already on the list",
  "Shorter ingredient", a bought entry not counting as on the list, and an ingredient with no words matching nothing

## 2. Web interface

- [x] 2.1 Add the check page at `/pages/{id}/pantry` and a "Check the pantry" link on recipe pages; verify web tests for
  the three states with the rows' names and locations, "No ingredients", "Not a recipe" (no link, and 404 on the URL)
- [x] 2.2 Add the form posting the ticked ingredients; verify web tests for "Add the missing ones", "Rename before
  adding", "Untick a missing one" and "Nothing ticked", and an entry merging with one already on the list

## 3. Integration

- [x] 3.1 With `make check` green, in the browser at 390px and at desktop width, check a recipe with found, listed and
  missing ingredients, rename one and add; verify the shopping list shows them and nothing overflows

## Workflow follow-up

- Archive the change once merged.
