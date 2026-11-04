# Proposal

> **Status: draft, to be discussed.** This records the ideas; the scope is not agreed yet. Design and tasks are
> written once it is, and the work starts after `add-wiki-foundation`, before the first release meant for everyday
> use.

## Why

The interface works and is responsive, but it looks generic: a neutral palette and default typography, unrelated to the
application's icon. People open hausbuch every day on their phones; it should feel like its own app and make pages,
recipes above all, pleasant to read.

## What Changes

Ideas to discuss:

- Colours taken from the icon (sage green on light green) for buttons, links and accents, instead of the current grey
  and black.
- The icon in the header, next to the name.
- A typeface with more character for headings, and more care in how a page reads (spacing, headings, lists), recipes
  above all.
- A richer home page: an icon per page type, or pages grouped by type instead of one flat list.
- A dark mode, following the device's setting.

### Out of scope

- Any change of behaviour: screens, fields and flows stay as the specs describe them.
- A second interface language.
- Choosing a theme per user.

## Open Questions

- References: apps or sites whose look should guide this.
- Which of the ideas above to keep, and whether dark mode is worth its cost (every screen checked twice).

## Capabilities

### New Capabilities

None: this change is about how the interface looks, not about what the system does.

### Modified Capabilities

None.

## Impact

- Templates and `internal/web/styles.css`; possibly a font embedded in the image, so nothing is loaded from the
  internet.
- Browser checks with screenshots at phone, tablet and desktop widths for every screen.
