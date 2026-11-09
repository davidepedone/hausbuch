# Tasks

## 1. Store

- [x] 1.1 Make `mergeInto` match the location ignoring case, as the design says; verify store tests for "Same product, different location", "Same location, different case" and "Different location moves apart", with the existing merge tests still green
- [x] 1.2 Add `stock.Store.Move` with `ErrSameLocation`; verify store tests for "Move some", "Move onto the same product", "Move all of it", "Move all of it onto the same product", "Move a decimal quantity", "Opened part moves apart", "Remembered where it went", "Too much, or nothing", "Same location" and an unknown row

## 2. Web interface

- [x] 2.1 Add the "Move" link to each row and the move page with its routes; verify web tests for "Default quantity", "Locations suggested", a move landing on `/<section>#item-<id>` of the receiving row, and both refusals answered with the page, the message and the entered values

## 3. Integration

- [x] 3.1 With `make check` green, in the browser at 390px and 1280px, add the same product in two locations, then move part of one row to the other and all of a row onto a matching one; verify every step behaves as the spec says and nothing overflows sideways

## Workflow follow-up

- Archive the change once merged.
