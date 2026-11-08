-- +goose Up
-- Quantities in tenths, so that they can have one decimal. SQLite can't change a column's constraint, so the table is
-- rebuilt: unlike the others, this migration leaves a database the previous release can't use.
CREATE TABLE stock_item_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    section         TEXT    NOT NULL,
    name            TEXT    NOT NULL,
    -- Name trimmed and lower-cased in Go: SQLite's NOCASE only folds ASCII.
    name_key        TEXT    NOT NULL,
    quantity_tenths INTEGER NOT NULL CHECK (quantity_tenths > 0),
    unit            TEXT    NOT NULL DEFAULT '',
    location        TEXT    NOT NULL DEFAULT '',
    -- Calendar dates as YYYY-MM-DD, with no time zone; NULL when not given.
    expires_on      TEXT,
    opened_on       TEXT,
    -- Days the item lasts once opened; NULL when not given.
    opened_days     INTEGER,
    created_by      INTEGER NOT NULL REFERENCES app_user (id),
    -- Unix seconds.
    created_at      INTEGER NOT NULL,
    changed_by      INTEGER NOT NULL REFERENCES app_user (id),
    changed_at      INTEGER NOT NULL
);
INSERT INTO stock_item_new
SELECT id, section, name, name_key, quantity * 10, unit, location, expires_on, opened_on, opened_days, created_by,
       created_at, changed_by, changed_at
FROM stock_item;
DROP TABLE stock_item;
ALTER TABLE stock_item_new RENAME TO stock_item;
CREATE INDEX stock_item_section_name ON stock_item (section, name_key);

-- +goose Down
CREATE TABLE stock_item_old (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    section     TEXT    NOT NULL,
    name        TEXT    NOT NULL,
    name_key    TEXT    NOT NULL,
    quantity    INTEGER NOT NULL CHECK (quantity >= 1),
    unit        TEXT    NOT NULL DEFAULT '',
    location    TEXT    NOT NULL DEFAULT '',
    expires_on  TEXT,
    opened_on   TEXT,
    opened_days INTEGER,
    created_by  INTEGER NOT NULL REFERENCES app_user (id),
    created_at  INTEGER NOT NULL,
    changed_by  INTEGER NOT NULL REFERENCES app_user (id),
    changed_at  INTEGER NOT NULL
);
-- Rounded up to whole units.
INSERT INTO stock_item_old
SELECT id, section, name, name_key, (quantity_tenths + 9) / 10, unit, location, expires_on, opened_on, opened_days,
       created_by, created_at, changed_by, changed_at
FROM stock_item;
DROP TABLE stock_item;
ALTER TABLE stock_item_old RENAME TO stock_item;
CREATE INDEX stock_item_section_name ON stock_item (section, name_key);
