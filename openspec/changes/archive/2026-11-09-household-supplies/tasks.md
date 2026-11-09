# Tasks

## 1. Model and configuration

- [x] 1.1 Add `stock.Supplies` after Pantry in `stock.Sections`, make `Label` a switch over the three sections, and give Supplies a default lead time of 7; verify store tests for "Supplies kept apart from food" and "Supplies default"
- [x] 1.2 Read `HAUSBUCH_SUPPLIES_LEAD_DAYS` in `leadTimes` and document it in the README, with Supplies in the introduction; verify `cmd/hausbuch` tests for "Configured supplies lead time" and that an invalid value names the setting

## 2. Web interface

- [x] 2.1 Draw the Supplies icon in `sectionIcon` and lay the home page's section tiles out in three columns with the icon above the label; verify `TestEverySectionHasAnIcon` covers Supplies, and web tests for "Sections on the home page" and "Supplies in the box"

## 3. Integration

- [x] 3.1 With `make check` green, open the home page and the Supplies section in the browser at 390px and 1280px, add a supplies item expiring within 7 days and check it appears in the "Expiring" box with its section; verify nothing overflows sideways and the three tiles sit on one row

## Workflow follow-up

- Archive the change once merged, which closes issue #27.
