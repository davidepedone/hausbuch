# Tasks

## 1. Store

- [x] 1.1 Add `search.Fold` and `stock.Store.Search`; verify unit tests for `Fold` on accented and upper-case text, and store tests for "Part of a name", "Case and accents", "Every word must be in the name", "Other fields are not searched", the sections' order and an empty query

## 2. Web interface

- [x] 2.1 Show the stock results in the search page, grouped by section after the pages, with name, quantity, unit, category, location, expiry note and a link to the row, and "Nothing found" only when nothing matches; verify web tests for "Grouped results", "Stock result", "Expiring stock result" and "No results"
- [x] 2.2 Label the header's field "Search"; verify a web test for "Label"

- [x] 2.3 Match page titles on parts of words in `page.Store.Search`, title matches first; verify page tests for "Part of a title", title matches by part of a word and by stem both before body-only matches, and a deleted page's title not matching
- [x] 2.4 Answer an HTMX request to `/search` with only the results part (nothing for an empty query), wire the header's field to a panel under it, and add `static/search.js` closing the panel on Escape and on a click outside; verify web tests for the partial answer, the empty answer and the field's attributes on every signed-in screen

## 3. Integration

- [x] 3.1 With `make check` green, in the browser at 390px and 1280px: type a part of a stock name and of a page title with different case and accents and watch the panel; clear the field with its ✕, press Escape, tap outside; confirm a search with Enter; search a word found nowhere; follow a page and a stock result from the panel; verify every step behaves as the spec says and nothing overflows sideways

## Workflow follow-up

- Archive the change once merged, which closes issue #35.
