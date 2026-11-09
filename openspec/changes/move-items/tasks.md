# Tasks

## 1. Store

- [x] 1.1 Give `stock.Store.Update` the target section and return the row's resulting id, moving or merging as the design says; verify store tests for "Move to the right section", "Move onto the same product", "Opened row moves apart", "Suggested in the new section", "Edit and move together" and an unknown target refused

## 2. Web interface

- [x] 2.1 Add the Section field to the edit page and redirect a save to the row in its section; verify web tests for moving from the edit page, landing on `/<target>#item-<id>`, and an unknown section refused with the form kept

## 3. Integration

- [ ] 3.1 With `make check` green, in the browser at 390px and 1280px, move an item to another section, then onto an existing row of the same product; verify every step behaves as the spec says and nothing overflows sideways

## Workflow follow-up

- Archive the change once merged, which closes issue #47.
