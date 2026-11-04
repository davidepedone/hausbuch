-- +goose Up
CREATE TABLE app_user (
    id                 INTEGER PRIMARY KEY,
    username           TEXT    NOT NULL,
    -- Lower-cased username: SQLite's NOCASE only folds ASCII letters.
    username_key       TEXT    NOT NULL UNIQUE,
    display_name       TEXT    NOT NULL,
    password_hash      TEXT    NOT NULL,
    disabled           INTEGER NOT NULL DEFAULT 0,
    failed_sign_ins    INTEGER NOT NULL DEFAULT 0,
    -- Unix seconds; NULL when not locked.
    locked_until       INTEGER
);

-- +goose Down
DROP TABLE app_user;
