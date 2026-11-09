# Proposal

## Why

Page bodies are Markdown typed by hand, and the editor only shows a one-line reminder of the syntax. People who don't
know Markdown can't add a link to a website, or even make a word bold, without looking it up (issue #30).

## What Changes

- **A row of formatting buttons above the page text**: external link, link to a page, bold, heading, bulleted list
  and checklist. Each inserts the Markdown into the text at the cursor, around the selected text, or at the start of
  the selected lines.
- **External link** asks for the address and uses the selected text as the link's text, or the address itself when
  nothing is selected; an address typed without `http://` or `https://` gets `https://`.
- **Pasting an address over selected text** turns that text into a link to it.
- The Markdown stays visible and editable in the text: the buttons are shortcuts, not a different editor.

### Out of scope

- A WYSIWYG editor, or hiding the Markdown.
- Buttons for the other formats Markdown supports (italic, numbered lists, tables, code): they can be added later
  the same way.
- Removing formatting with the same button (toggling).
- Choosing a page from a list for a page link: the button only writes `[[` and `]]`.
- Keyboard shortcuts for the buttons.
- Formatting buttons anywhere else, such as the stock sections' fields.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `wiki-pages`: a new requirement for the editor's formatting buttons and for pasting an address over selected text.

## Impact

- The page editor template, and a new script in `internal/web/static`, loaded only by the editor.
- A test-only Go dependency to run the script's text functions in tests (see design).
