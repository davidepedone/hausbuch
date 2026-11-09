# Tasks

## 1. Store

- [x] 1.1 Add `search.Fold` and `stock.Store.Search`; verify unit tests for `Fold` on accented and upper-case text, and store tests for "Part of a name", "Case and accents", "Every word must be in the name", "Other fields are not searched", the sections' order and an empty query

## 2. Web interface

- [x] 2.1 Show the stock results in the search page, grouped by section after the pages, with name, quantity, unit, category, location, expiry note and a link to the row, and "Nothing found" only when nothing matches; verify web tests for "Grouped results", "Stock result", "Expiring stock result" and "No results"
- [x] 2.2 Label the header's field "Search"; verify a web test for "Label"

## 3. Integration

- [ ] 3.1 With `make check` green, in the browser at 390px and 1280px, search a part of a stock name with different case and accents, a word found in pages and in stock, and a word found nowhere; follow a stock result to its row; verify every step behaves as the spec says and nothing overflows sideways

## Workflow follow-up

- Archive the change once merged, which closes issue #35.
