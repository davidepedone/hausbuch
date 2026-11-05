-- +goose Up
-- Full-text index of the pages that aren't deleted, rowid being the page id. It holds the stems of the current title
-- and body, computed in Go in the page's language: FTS5 itself only stems English.
CREATE VIRTUAL TABLE page_fts USING fts5 (title_stem, body_stem, tokenize = 'unicode61 remove_diacritics 2');

-- +goose Down
DROP TABLE page_fts;
