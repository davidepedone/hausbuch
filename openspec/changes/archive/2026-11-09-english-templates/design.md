# Design

## Context

See `proposal.md` for the motivation and the delta specs for the required behaviour. The current state that shapes
the approach:

- `page.Skeleton(type, lang)` reads `pagetypes/<type>.<lang>.md` from an embedded directory. The new page form starts
  from the default language's Blank skeleton, and both the type and the language `<select>` post to
  `/pages/skeleton`, which swaps the body while it is still the prefilled skeleton.
- `page` and `page_revision` store a `language`; a save that changes only the language creates a revision.
- The full-text index (`page_fts`) holds `search.Stems(text, lang)`, the stems in the page's language; `search.Query`
  already matches each searched word by its English or its Italian stem; `search.MakeExcerpt` stems the body in the
  page's language to find the match.
- At start-up `page.Store.Reindex` rebuilds the index from scratch.
- `cmd/hausbuch` reads `HAUSBUCH_DEFAULT_LANGUAGE` and refuses to start with an unknown value.
- A previous release must keep working on the database of a newer one ("Running a previous release" in
  `database-backups`): it reads and writes `language`, `NOT NULL` with no default, in both tables.

## Decisions

### One skeleton file per type

`pagetypes/<type>.it.md` are removed and `<type>.en.md` renamed `<type>.md`; `page.Skeleton(type)` loses its language
parameter. Only the type `<select>` swaps the skeleton.

### Both languages' stems in the index

`search.Stems(text)` returns, for each word, its English stem and, when it differs, its Italian one. The index then
answers `search.Query`, which already asks for either, whatever language a page is written in; `MakeExcerpt` marks the
first word whose English or Italian stem was searched for. The stored stems change, but `Reindex` rebuilds the index
at every start, so nothing else is needed.

*Alternative considered*: keeping the language, hidden, set from the default. It would still have to be configured
and would be wrong for any page written in the other language.

### The column stays

`language` stays in `page` and `page_revision`, and is no longer read. New pages and revisions write `"en"` into it,
only so that a previous release, which needs a value, can still run on the database; existing rows keep theirs.

*Alternative considered*: a migration dropping the column. The previous release would then fail on every page query,
and going back would need the copy of the database saved before the migration.

### The setting

`HAUSBUCH_DEFAULT_LANGUAGE` is no longer read: an installation that still sets it starts as usual. `web.NewHandler`
loses its default language.

## Risks / Trade-offs

- [A word's stem in one language matches an unrelated word's stem in the other] → An extra result now and then, in a
  household wiki; the excerpt shows why it matched.
- [After going back, pages created by this release read as English] → The previous release's search stems them in
  English until each is re-saved in Italian; acceptable for a rollback.
- [A column nobody reads] → A comment says why it is written; it can go with a later release that rebuilds the
  tables for another reason.
