# Tasks

## 1. Search and pages

- [x] 1.1 Make `search.Stems` and `search.MakeExcerpt` work in both languages; verify unit tests for a word with different English and Italian stems, and an excerpt found by an Italian form in any page
- [x] 1.2 Remove the language from `page.Content`, revisions, indexing and `page.Skeleton`, keeping one English skeleton per type and writing `"en"` into the unused columns; verify page tests for "English word forms", "Italian word forms" and "Searching without choosing a language", "Save without changes", and the skeleton headings test

## 2. Web interface and configuration

- [x] 2.1 Remove the Language field, the language on the page, revision and compare views, and the skeleton swap on a language change; stop reading `HAUSBUCH_DEFAULT_LANGUAGE` and remove it from the README; verify web tests for "Type changed before writing", "Type changed after writing", "Open an old revision", and no language on the editor, page and revision views

## 3. Integration

- [ ] 3.1 With `make check` green, in the browser at 390px and 1280px: create pages of each type, switch the type before and after typing, search Italian and English word forms, open an old revision; verify every step behaves as the specs say

## Workflow follow-up

- Archive the change once merged, which closes issue #48.
- The release notes of the release containing it say that `HAUSBUCH_DEFAULT_LANGUAGE` can be removed.
