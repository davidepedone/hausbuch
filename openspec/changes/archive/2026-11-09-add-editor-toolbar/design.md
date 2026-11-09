# Design

## Context

See `proposal.md` for the motivation and `specs/wiki-pages/spec.md` for the required behaviour. The current state that
shapes the approach:

- The editor (`edit.html`) is a form with the title, type, language and a `textarea name="body"` inside `#body`. On a
  new page, changing type or language swaps `#body` through htmx with a new skeleton, so the textarea element is
  replaced.
- The only script is `htmx.min.js`; there is no JavaScript of the project's own, no build step for scripts and no
  Node.js in the build or CI. `make check` runs `golangci-lint`, `go test ./...` and the build.
- Static files are embedded and served from `/static/`.

## Goals / Non-Goals

**Goals:**

- Every scenario tested in `go test`, without adding Node.js to the build.
- The browser's undo (Ctrl+Z, shake to undo) still works after a button or a paste.

**Non-Goals:**

- Working without JavaScript: the editor stays usable by typing Markdown, the buttons just do nothing.

## Decisions

### One small script, with the text logic as pure functions

`internal/web/static/editor.js`, loaded with `defer` by the editor template only. It has two parts:

- **Pure functions** taking the body text and the selection (start, end) and returning the edit to make: the range to
  replace, the text to put there and the selection afterwards. One per action (`bold`, `pageLink`, `link(url)`,
  `linePrefix(prefix)` for heading, list and checklist), plus `pastedLink(text)` returning the address when the pasted
  text is a single `http(s)://` address, and `withScheme(url)`.
- **Wiring**: one `click` listener on the toolbar and one `paste` listener on the form, delegated, so they keep working
  after htmx replaces the textarea. The external link button uses `window.prompt` for the address: it works by touch
  on every phone without a dialog of our own.

The toolbar is a row of buttons above `#body`, outside it so the skeleton swap leaves it alone, with `type="button"`
so they never submit the form. Each has a visible short label or symbol and an `aria-label`. The row wraps on narrow
screens rather than scrolling.

### Applying an edit through `insertText`

The wiring selects the range and calls `document.execCommand("insertText", false, text)`, then sets the selection.
Unlike assigning `value` or `setRangeText`, this goes through the browser's undo history, in every current browser,
for a textarea. `execCommand` is deprecated but has no replacement for this; if it returns `false` the wiring falls
back to `setRangeText`, losing undo for that edit only.

### Testing the pure functions with goja

The pure functions are tested from Go by loading `editor.js` into [goja](https://github.com/dop251/goja), a JavaScript
interpreter written in Go, as a test-only dependency: `go test ./...` covers every text scenario without Node.js.
The script guards its wiring with `typeof document !== "undefined"`, so it loads in goja without a DOM.

The Go web tests check that the editor page has the toolbar with its six buttons and loads the script. The wiring
(click, prompt, paste, undo) and the phone layout are checked in a browser, as `tasks.md` says.

*Alternatives considered*: Node's test runner (adds Node.js to CI and to every machine running `make check`); a
headless browser in Go tests (a browser in CI, slow and heavy for six buttons); testing only by hand (leaves the
scenarios without tests, against the project's rule).

## Risks / Trade-offs

- [`execCommand` removed from a future browser] → The fallback keeps the buttons working, without undo.
- [`window.prompt` looks plain] → Accepted: it is native, accessible and needs no code; a dialog of our own can come
  later if it is a problem.
- [goja runs a subset of modern JavaScript] → The script uses plain ES2015+ that goja supports (no modules); a test
  failing to load the script would show it at once.
