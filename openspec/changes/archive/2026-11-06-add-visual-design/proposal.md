# Proposal

## Why

The interface works and is responsive, but it looks generic: a neutral palette and default typography, unrelated to the
application's icon. People open hausbuch every day on their phones; it should feel like its own app and make pages,
recipes above all, pleasant to read.

## What Changes

- **Accent colour from the icon**: the icon's sage green replaces the current dark grey on buttons, active filter
  chips, links and the headings inside a page. The warm grey neutrals (backgrounds, borders, secondary text) stay.
- **Header**: a light green band, the colour of the icon's background, with the icon next to the name.
- **Typeface for headings**: Fraunces for the name in the header, page titles and the headings inside a page. Body
  text keeps the system font. The font file is embedded in the image, so nothing is loaded from the internet.
- **Reading a page**: the page body's typography is tuned for every page type alike — headings in Fraunces and sage,
  list markers and step numbers in sage, more generous line height and spacing, slightly larger text on phones.
- **An icon per page type**, drawn in the same style as the application's icon.
- **A richer home page**: the type filter chips become a tile per type, with its icon, filtering the list as the chips
  do; the list below shows each page's type icon and is headed "Recently changed", or the type's name when filtered,
  with a "Show all" link to clear the filter.

### Out of scope

- A dark mode: it would need every colour turned into a semantic token with light and dark values; if wanted, it is a
  change of its own.
- Grouping the home page by type: the type tiles already do that job.
- The number of pages of each type on its tile: it is new data, not a new look.
- Styling specific to recipes (ingredients as a checklist, a box for servings and time): it would depend on recognising
  sections by their headings, which people can rename or remove, and a checklist is behaviour.
- Tinting the neutral greys green.
- Any change of behaviour: screens, fields and flows stay as the specs describe them.
- A second interface language.
- Choosing a theme per user.

## Capabilities

### New Capabilities

None: this change is about how the interface looks, not about what the system does.

### Modified Capabilities

None.

## Impact

- `internal/web/styles.css` (accent colour, font, typography settings), the templates, and new static files: the font
  and the page type icons.
- Browser checks with screenshots at phone, tablet and desktop widths for every screen.
