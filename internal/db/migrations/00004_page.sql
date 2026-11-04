-- +goose Up
CREATE TABLE page (
    -- AUTOINCREMENT so that the id of a removed page, still in someone's bookmarks, never names another page.
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT    NOT NULL,
    -- Title trimmed and lower-cased in Go: SQLite's NOCASE only folds ASCII.
    title_key  TEXT    NOT NULL UNIQUE,
    type       TEXT    NOT NULL,
    language   TEXT    NOT NULL,
    -- Number of the current revision.
    revision   INTEGER NOT NULL,
    -- Unix seconds; both NULL unless the page is deleted.
    deleted_at INTEGER,
    deleted_by INTEGER REFERENCES app_user (id)
);

-- Append-only: a page's content is its newest revision.
CREATE TABLE page_revision (
    page_id    INTEGER NOT NULL REFERENCES page (id) ON DELETE CASCADE,
    number     INTEGER NOT NULL,
    title      TEXT    NOT NULL,
    type       TEXT    NOT NULL,
    language   TEXT    NOT NULL,
    body       TEXT    NOT NULL,
    author_id  INTEGER NOT NULL REFERENCES app_user (id),
    -- Unix seconds.
    created_at INTEGER NOT NULL,
    PRIMARY KEY (page_id, number)
);

-- Former titles of a page, so that links using them keep leading to it.
CREATE TABLE page_alias (
    title_key TEXT    PRIMARY KEY,
    page_id   INTEGER NOT NULL REFERENCES page (id) ON DELETE CASCADE
);
CREATE INDEX page_alias_page_id ON page_alias (page_id);

-- The titles a page's body links to, for "pages linking here".
CREATE TABLE page_link (
    page_id   INTEGER NOT NULL REFERENCES page (id) ON DELETE CASCADE,
    title_key TEXT    NOT NULL,
    PRIMARY KEY (page_id, title_key)
);
CREATE INDEX page_link_title_key ON page_link (title_key);

-- +goose Down
DROP TABLE page_link;
DROP TABLE page_alias;
DROP TABLE page_revision;
DROP TABLE page;
