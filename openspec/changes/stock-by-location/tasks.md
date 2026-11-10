# Tasks

## 1. Store

- [x] 1.1 Add `Store.Places` and `Store.InLocation`; verify store tests for "Locations across sections", "No
  locations", "What is in the cellar" (rows of two sections, case ignored) and a location left empty
- [x] 1.2 Make `Store.Locations` read every section, one per location ignoring case; verify store tests for "Locations
  from the whole home", "One choice per location" and "Nothing used yet", with the existing suggestion tests still green

## 2. Web interface

- [x] 2.1 Add the Locations page, the page per location and the home tile, and link a row's location to its page, in a section and on a recipe's pantry check;
  verify web tests for "Locations across sections", "No locations", "What is in the cellar", "From a row to its
  location", "A location left empty", and a location holding "/"
- [x] 2.2 Turn `categoryField` into `choiceField` and use it for the location field of the add form, the edit page, the
  move page and the put-away page; verify web tests for the choices on those pages and a browser test for "Choose a
  location under the field", with the category tests still green

## 3. Integration

- [x] 3.1 With `make check` green, in the browser at 390px and at desktop width, open the home page, the Locations page,
  a location's page and an add form; verify the tiles, the lists and the choices hold together and nothing overflows

## Workflow follow-up

- Archive the change once merged.
