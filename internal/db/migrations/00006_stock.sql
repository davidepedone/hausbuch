-- +goose Up
-- What the household has at home: one row per product and expiry date, or per opened unit.
CREATE TABLE stock_item (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    section     TEXT    NOT NULL,
    name        TEXT    NOT NULL,
    -- Name trimmed and lower-cased in Go: SQLite's NOCASE only folds ASCII.
    name_key    TEXT    NOT NULL,
    quantity    INTEGER NOT NULL CHECK (quantity >= 1),
    unit        TEXT    NOT NULL DEFAULT '',
    location    TEXT    NOT NULL DEFAULT '',
    -- Calendar dates as YYYY-MM-DD, with no time zone; NULL when not given.
    expires_on  TEXT,
    opened_on   TEXT,
    -- Days the item lasts once opened; NULL when not given.
    opened_days INTEGER,
    created_by  INTEGER NOT NULL REFERENCES app_user (id),
    -- Unix seconds.
    created_at  INTEGER NOT NULL,
    changed_by  INTEGER NOT NULL REFERENCES app_user (id),
    changed_at  INTEGER NOT NULL
);
CREATE INDEX stock_item_section_name ON stock_item (section, name_key);

-- Every product a section has had, with the values it last had, for suggestions; kept when its rows are removed.
CREATE TABLE stock_product (
    section     TEXT    NOT NULL,
    name_key    TEXT    NOT NULL,
    name        TEXT    NOT NULL,
    location    TEXT    NOT NULL DEFAULT '',
    opened_days INTEGER,
    PRIMARY KEY (section, name_key)
);

-- +goose Down
DROP TABLE stock_product;
DROP TABLE stock_item;
