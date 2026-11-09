# Tasks

## 1. Text functions (`wiki-pages`)

- [x] 1.1 Add `internal/web/static/editor.js` with the pure functions for bold, page link, external link (with `withScheme`), line prefixes for heading, list and checklist, and pasted-address detection, and a Go test running them in goja (test-only dependency); verify a test for every text outcome in the "Formatting buttons in the editor", "External links from the editor" and "Paste an address over selected text" scenarios, including cancel and empty address leaving the body unchanged

## 2. Editor

- [x] 2.1 Add the toolbar above `#body` in `edit.html`, load `editor.js` there only, and wire clicks, the address prompt and paste through `insertText` with the `setRangeText` fallback; verify a web test that the editor has the six buttons and the script, and `make check` green
- [x] 2.2 In a browser, at 390px and 1280px: each button on a selection, on nothing selected and on several lines; the link prompt with and without scheme and cancelled; pasting an address and ordinary text over a selection; Ctrl+Z after a button; buttons still working after changing type on a new page; verify every step behaves as the spec says and nothing overflows sideways

## Workflow follow-up

- Archive the change once merged, which closes issue #30.
