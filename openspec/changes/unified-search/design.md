# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. The current state that shapes the
approach:

- Pages are searched through the SQLite full-text index (`page_fts`), which holds stems and strips diacritics.
- Stock rows have `name_key`, the name trimmed and lower-cased in Go, since SQLite's `lower()` only folds ASCII; there
  is no accent-free form of the name and no index on parts of it.
- `search.Words` splits text into lower-cased words on anything that isn't a letter, digit or mark.
- Stock sections are household-sized: tens of rows each; the wiki has tens to a few hundred pages.
- The full-text index holds stems, so a word still being typed ("irri") matches nothing until it is complete.
- The search page renders `[]page.Result`; the home page's "Expiring" box already shows a stock row with its section,
  expiry note and a link to `/<section>#item-<id>`.

## Goals / Non-Goals

**Goals:**

- Stock matching that ignores accents for any letter, not just ASCII.

**Non-Goals:**

- An index for stock names: at these sizes a scan is instantaneous.

## Decisions

### Matching in Go over every row

`stock.Store.Search(ctx, text)` reads every row, as `Warnings` already does, and keeps those whose folded name
contains each folded word of `search.Words(text)`; no words means no results. Rows come back in the sections' order
(`stock.Sections`), and within a section as `List` orders them, so the template only has to group consecutive rows.

`search.Fold` folds a text for this comparison: Unicode decomposition (`golang.org/x/text/unicode/norm`, NFD), marks
removed, lower-cased. It sits in `internal/search` next to `Words`, where the other text handling for search lives.

*Alternative considered*: a stored accent-free `name_fold` column with `LIKE '%…%'`. It needs a migration and keeping
the column in step on every write, to save a scan of a few hundred rows.

### One results page, groups in the template

`searchData` gains the stock rows split by section, one entry per section with matches. The template shows a "Pages"
group when there are page results and one group per section, headed by the section's label; each row reuses the expiry note and the link of the "Expiring"
box. "Nothing found" and the offer to create a page appear only when both are empty.

The header's field becomes "Search" in its placeholder and `aria-label`.

### Page titles matched on parts of words

`page.Store.Search` keeps its full-text query and adds the pages whose folded title contains each folded word, found
by scanning the titles of the pages that aren't deleted. Title matches come first, whether by stem or by part of a
word: the ids matching the full-text query on `title_stem` alone are read separately, and the merged list is sorted
stably on "matches in the title", keeping the full-text order within each half. A page found only by its title gets an
excerpt as today, which is empty when its body doesn't contain the words.

*Alternative considered*: a prefix query on the last word (`"irri"*`) in the full-text index. The index holds stems,
so a prefix of the typed word stops matching once it grows past the stem ("irrigaz" against "irrig"), and results
would come and go while typing.

### Results while typing through HTMX

The header's field gets `hx-get="/search"`, triggered on `input changed delay:200ms` and on `search` (the event a
search field fires when its ✕ clears it), targeting a panel under the field. `/search` answers an HTMX request with
only the results part, the same template the full page uses, and with nothing for an empty query, which empties the
panel. The form still submits normally on Enter, opening the full page.

A small `static/search.js`, loaded by the layout, empties the panel on Escape and on a click outside the search form.
The panel scrolls within the screen's height and sits above the page, so the screen underneath doesn't move.

*Alternative considered*: showing the panel only while the field has focus, through CSS. Safari doesn't focus a link
when it is tapped, so the panel would close before the tap reached the result.

## Risks / Trade-offs

- [A short query such as "a" lists most of the stock and many titles] → Accepted: results are grouped, the panel
  scrolls, and the user refines the query as they type.
- [A request per pause in typing] → One query over the index and two scans of household-sized tables, at most every
  200ms while typing.
