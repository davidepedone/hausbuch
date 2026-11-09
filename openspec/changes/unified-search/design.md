# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. The current state that shapes the
approach:

- Pages are searched through the SQLite full-text index (`page_fts`), which holds stems and strips diacritics.
- Stock rows have `name_key`, the name trimmed and lower-cased in Go, since SQLite's `lower()` only folds ASCII; there
  is no accent-free form of the name and no index on parts of it.
- `search.Words` splits text into lower-cased words on anything that isn't a letter, digit or mark.
- Stock sections are household-sized: tens of rows each.
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

`searchData` gains `Items []stock.Item`. The template shows a "Pages" group when there are page results and one group
per section with rows, headed by the section's label; each row reuses the expiry note and the link of the "Expiring"
box. "Nothing found" and the offer to create a page appear only when both are empty.

The header's field becomes "Search" in its placeholder and `aria-label`.

## Risks / Trade-offs

- [A short query such as "a" lists most of the stock] → Accepted: results are grouped and the user refines the query;
  pages already behave alike for common words.
