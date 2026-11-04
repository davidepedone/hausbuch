-- +goose Up
CREATE TABLE session (
    -- SHA-256 of the token in the cookie: a copy of the database doesn't let anyone sign in.
    token_hash BLOB    PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    -- Unix seconds.
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX session_user_id ON session (user_id);

-- +goose Down
DROP TABLE session;
