# Design

## Context

See `proposal.md` for the motivation and the scope. The current state that shapes the approach:

- Styling is Tailwind v4: `internal/web/styles.css` is compiled by `make css` (the standalone binary, pinned in the
  `Makefile`, also run in the `Dockerfile`) into `static/app.css`, which is embedded with the rest of `static/` and the
  templates. Static assets are served without login.
- The templates carry utility classes directly; every colour is a Tailwind `stone` shade, with red, amber and green
  kept for errors, warnings and confirmations. Page bodies use `prose prose-stone` from `@tailwindcss/typography`.
- The icon (`static/favicon.svg`) uses two colours: `#4E6B52` for its rounded 3px strokes and `#DDE8D8` for its
  background.
- Page types are `page.Types` (Blank, Recipe, How-to, Where is it), available to templates through the `types`
  function.

## Goals / Non-Goals

**Goals:**

- Every colour, font and icon comes from the image: nothing is loaded from the internet.
- Text keeps WCAG AA contrast (4.5:1) wherever the sage colours are used.

**Non-Goals:**

- Turning colours into semantic tokens: without a dark mode a direct replacement is enough.
- Restyling the red, amber and green messages.

## Decisions

### A `sage` palette in `@theme`, used in place of the dark greys

`styles.css` defines a `sage` colour in its `@theme` block, built around the icon's two colours: `sage-100` is
`#DDE8D8` (the icon's background), `sage-700` is `#4E6B52` (its strokes), with `sage-50` for light tints and
`sage-800` for hover. Templates then use `bg-sage-700 hover:bg-sage-800` where they use `bg-stone-800
hover:bg-stone-700` today (primary buttons, active filter chips), and `text-sage-700` for links. Neutral greys stay
`stone`.

Measured contrast: white on `#4E6B52` is 5.9:1, `#4E6B52` on the `stone-50` page background 5.7:1 and on `#DDE8D8`
4.7:1, all above AA; dark text on `#DDE8D8` is 13.8:1.

Alternatives considered:
- *Redefining `stone` itself*: no template edits, but it would also tint borders and secondary text, which the
  proposal keeps grey.

### Header

The header's background becomes `sage-100` with a `sage-200` bottom border; the icon is inlined before the name as
`favicon.svg`'s content at 28px, so it needs no extra request and stays crisp. The search field keeps a white
background so it reads as a field on the green band.

### Fraunces, latin subset, weight and optical size axes

The font is the `fraunces-latin-opsz-normal.woff2` file of `@fontsource-variable/fraunces` 5.3.0 (about 67 KB),
stored as `static/fonts/fraunces.woff2` with the font's OFL licence next to it as `static/fonts/OFL.txt`, as the
licence requires. It has the weight and optical size axes; the latin subset covers English and Italian. `styles.css`
declares it with `@font-face` (`font-display: swap`, so text shows at once in the fallback) and adds `--font-display:
"Fraunces", Georgia, serif` to `@theme`; templates use `font-display` on the app name and page titles, and the
typography settings use it for headings inside a page.

Alternatives considered:
- *The full file with the SOFT and WONK axes* (about 121 KB): axes the design doesn't use.
- *The weight axis only* (smaller): optical size is what makes Fraunces look right both at the size of a page title
  and at the size of a heading inside a page.
- *Downloading it in `make css`*: one more network step in every build for a file that never changes; committing it
  keeps the build as it is.

### Page typography: one `prose-sage` theme for every page type

`@tailwindcss/typography` reads its colours from `--tw-prose-*` variables, so `styles.css` defines a `prose-sage`
class setting them (headings and links in `sage-700`, list markers and counters in sage, body text staying `stone`)
plus the heading font, a slightly larger body size and line height below the `sm` breakpoint, and more space above
`h2`, so the sections of a skeleton (Ingredients, Steps…) stand apart. The three templates using `prose prose-stone`
(page, revision, the editor's preview) switch to `prose prose-sage`, so the preview keeps looking like the saved page.

### Page type icons as inline SVG in a template

`parts.html` gets a `typeIcon` template taking a `page.Type` and emitting a 24×24 SVG drawn like the application's
icon: a single colour (`currentColor`), rounded 2px strokes, no fill. Inline SVG with `currentColor` lets the icon
follow its context: sage on a light tile, white on the active one. Motifs: a blank sheet (Blank), a pot with steam
(Recipe), a wrench (How-to), a box with a location pin (Where is it).

A test checks that `typeIcon` renders an SVG for every type in `page.Types`, so a type added later without an icon
fails the build instead of showing an empty space.

Alternatives considered:
- *One SVG file per type under `static/`*: four more requests on the home page, and `<img>` can't take the tile's
  text colour.

### Home page: type tiles above the list

The filter stays what it is today, a link to `/?type=<type>`, and the list keeps its order and content; only how they
are presented changes:

- The chips become a grid of tiles, one per type, each with its icon (32px) above the type's label, on a `sage-50`
  background with a `sage-200` border. Two columns on phones, four from the `sm` breakpoint. The tile of the active
  filter is `sage-700` with white text and keeps `aria-current="page"`.
- The "All" chip goes. The list's heading is "Recently changed" without a filter and the type's label with one; with
  a filter, a "Show all" link next to the heading goes back to `/`.
- Each row of the list starts with the type's icon (20px) in a `sage-50` rounded square; the row's second line keeps
  the type's label, so the type is still written, not only drawn.
- Tile labels are the existing `Label()` values ("Recipe", "Where is it"), so no new interface strings are needed.

Alternatives considered:
- *Clearing the filter by tapping the active tile again*: one control fewer, but a gesture nobody guesses.
- *A grid of page cards instead of a list*: more scrolling as pages grow, and the list already reads well on phones.

### Checking the result

The change has no behaviour to test beyond the icon test above; the existing tests must stay green, since class
changes must not touch content. Every screen is checked in a browser with screenshots at phone (390px), tablet
(820px) and desktop (1280px) widths: login, home (with and without a type filter, and with no pages), a recipe, a how-to, editing with
the preview, the editing warning, history, comparing revisions, a revision, deleted pages, search, change password.

## Risks / Trade-offs

- [Templates edited by hand miss a `stone-800`] → after the change, a search of the templates for `stone-800` and
  `stone-700` backgrounds finds none.
- [Fraunces loads after the first paint and the titles shift] → `font-display: swap` with a serif fallback of similar
  width; the file is small and cached, so it happens on the first visit only.
- [Italian characters outside the latin subset] → the subset covers Latin-1, which includes every accented letter
  Italian uses; checked on an Italian recipe during the browser checks.
