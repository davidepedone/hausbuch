# Proposal

## Why

The interface stays in English (issue #33, closed), and the page templates can too: whoever writes a page in Italian
renames its section headings if they want (issue #48). Without Italian templates, a page's language is left with one
job, telling the search how to recognise word forms, and the search can do that for both languages at once: the field
every editor has to set, and the default language every installation configures, are then worth nothing.

## What Changes

- **One skeleton per type, in English**: a new Recipe, How-to or Where is it page starts with English headings.
- **No page language**: the editor's Language field, the language shown on a page and in its history, and the
  `HAUSBUCH_DEFAULT_LANGUAGE` setting are removed.
- **Search matches English and Italian word forms in every page**, as it does today for a page in the right language:
  "irrigare" finds "l'irrigazione" and "descale" finds "descaling" wherever they are written.
- The Italian skeletons are gone, and `HAUSBUCH_DEFAULT_LANGUAGE` is ignored. Pages keep their body, and the database
  keeps the stored language, unused, so that the previous release still runs on it.

### Out of scope

- Rewriting the headings of existing pages created from an Italian skeleton.
- Languages other than English and Italian for search.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `page-templates`: "Skeletons in the page's language" is removed; "Skeleton follows choices until the user writes"
  is replaced by "Skeleton follows the type until the user writes".
- `wiki-pages`: "Page language" is removed.
- `page-search`: "Search pages by words" matches word forms in both languages, whatever the page.
- `page-history`: "Every save is a revision" and "View the history of a page" no longer mention a language.

## Impact

- No migration: the `language` columns stay, written with a fixed value; the full-text index is rebuilt at start-up,
  as today, with both languages' stems.
- `internal/search` (stems and excerpts in both languages), `internal/page` (content, skeletons, revisions,
  indexing), `internal/web` (editor, page, revision and compare views), `cmd/hausbuch` and the README (the setting).
