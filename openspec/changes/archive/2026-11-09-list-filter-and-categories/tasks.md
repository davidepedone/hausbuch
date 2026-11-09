# Tasks

## 1. Store (`household-stock`)

- [x] 1.1 Add migration `00008` with `category` on `stock_item` and `stock_product`, and carry `Values.Category` through add, update, restore, `remember` and `Product`, plus a `Categories` suggestion query; verify a migration test that existing rows get an empty category, and store tests for "Item with a category", "Undo", "Buy something again", "Category suggestions" and "Suggestions stay in their section"
- [x] 1.2 Add `stock.GroupItems`; verify unit tests for "Grouped list", "Same category, different case", "Chips of a section" and "No categories"

## 2. Web interface

- [x] 2.1 Write `static/listfilter.js` (chips, groups, "Nothing matches", re-applying the chosen chip after HTMX swaps); its behaviour is checked in the browser in 3.1
- [x] 2.2 In the section page, add the chips and the grouped list, and the category field (with suggestions) to the add form, the edit form and the undo strip, filled in from a known product; verify web tests for "Item with a category", "Chips of a section", "No categories", undoing with a category and the product fragment filling the category
- [x] 2.3 On the home page, replace the type tiles with the type chips, and drop `?type=` and `page.Store.List`'s type parameter; verify web tests for "Home page" and "Chips of the page list", replacing `TestHomeFiltersByType`

## 3. Integration

- [x] 3.1 With `make check` green, in the browser at 390px and 1280px, on the home page and in a section with categories: choose a chip, "All", a type chip with no pages, use, open, remove and undo a row while a chip is chosen, and remove the only item of a chosen chip; verify every step behaves as the specs say and nothing overflows sideways

## Workflow follow-up

- Archive the change once merged, which closes issue #28.
