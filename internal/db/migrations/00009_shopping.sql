-- +goose Up
-- The household's shopping list: one row per entry, to buy or bought.
CREATE TABLE shopping_entry (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    -- Name trimmed and lower-cased in Go: SQLite's NOCASE only folds ASCII.
    name_key        TEXT    NOT NULL,
    -- The stock section it is put away into, '' for none.
    section         TEXT    NOT NULL DEFAULT '',
    -- In tenths, as in stock_item; NULL when not given.
    quantity_tenths INTEGER,
    unit            TEXT    NOT NULL DEFAULT '',
    shop            TEXT    NOT NULL DEFAULT '',
    bought          INTEGER NOT NULL DEFAULT 0,
    created_by      INTEGER NOT NULL REFERENCES app_user (id),
    -- Unix seconds.
    created_at      INTEGER NOT NULL
);

-- The shop each name was last listed with, kept when its entries are gone.
CREATE TABLE shopping_product (
    name_key TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    shop     TEXT NOT NULL
);

-- +goose Down
DROP TABLE shopping_product;
DROP TABLE shopping_entry;
