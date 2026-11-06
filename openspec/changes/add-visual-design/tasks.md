# Tasks

## 1. Colours and header

- [ ] 1.1 Add the `sage` palette to `styles.css`'s `@theme` and replace the `stone-800`/`stone-700` button and active chip backgrounds and the link colours in every template; verify `make check` passes and a search of the templates finds no `bg-stone-800` or `bg-stone-700`
- [ ] 1.2 Give the header the `sage-100` background with the icon inlined before the name, the search field staying white; verify in a browser at phone, tablet and desktop widths that the header wraps as before

## 2. Typeface

- [ ] 2.1 Add `fraunces-latin-opsz-normal.woff2` from `@fontsource-variable/fraunces` 5.3.0 as `static/fonts/fraunces.woff2` with its licence as `static/fonts/OFL.txt`, declare it with `@font-face` and `--font-display` in `styles.css`, and use `font-display` on the app name and page titles; verify in a browser that the font is served from `/static/fonts/` with no request leaving the server, and that an Italian title with accented letters renders in Fraunces

## 3. Reading a page

- [ ] 3.1 Define the `prose-sage` typography theme in `styles.css` (sage headings, links and list markers, Fraunces headings, larger text and line height on phones, more space above `h2`) and switch the page, revision and preview templates from `prose-stone` to it; verify `make check` passes and, in a browser, a recipe, a how-to and the editor's preview at the three widths

## 4. Page type icons and home page

- [ ] 4.1 Add the `typeIcon` template to `parts.html` with an icon for every page type, and a test that it renders an SVG for every type in `page.Types`; verify the test fails with a type missing its icon and passes with all of them
- [ ] 4.2 Replace the home page's chips with the type tiles, head the list "Recently changed" or with the filtered type's label plus a "Show all" link, and start each row with its type icon; verify the existing list and filter tests still pass, and in a browser at the three widths that the active tile is highlighted and "Show all" clears the filter

## 5. Integration

- [ ] 5.1 Build the image and check every screen listed in the design's "Checking the result" with screenshots at 390px, 820px and 1280px; verify nothing overflows horizontally, contrast is as the design states, and the red, amber and green messages look as before

## Workflow follow-up

- Archive the change once merged.
