-- +goose Up
-- The installation's VAPID key pair, created at the first start: subscriptions are bound to its public key, so it must
-- not change. One row.
CREATE TABLE push_key (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    -- The P-256 private scalar, 32 bytes.
    private_key BLOB    NOT NULL
);

-- A browser's Web Push subscription, belonging to the session it was made in: ending the session ends it.
CREATE TABLE push_subscription (
    endpoint     TEXT    PRIMARY KEY,
    session_hash BLOB    NOT NULL REFERENCES session (token_hash) ON DELETE CASCADE,
    user_id      INTEGER NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    -- The browser's keys, base64url as the browser gives them.
    p256dh       TEXT    NOT NULL,
    auth         TEXT    NOT NULL,
    -- Unix seconds.
    created_at   INTEGER NOT NULL
);
CREATE INDEX push_subscription_session_hash ON push_subscription (session_hash);
CREATE INDEX push_subscription_user_id ON push_subscription (user_id);

-- A user's choices for their daily summary; no row means the defaults.
CREATE TABLE push_choice (
    user_id   INTEGER PRIMARY KEY REFERENCES app_user (id) ON DELETE CASCADE,
    -- Minutes after midnight, in the installation's time zone.
    minutes   INTEGER NOT NULL,
    medicines INTEGER NOT NULL,
    pantry    INTEGER NOT NULL,
    supplies  INTEGER NOT NULL,
    -- YYYY-MM-DD of the last day a summary was due; NULL before the first.
    last_day  TEXT
);

-- The state, expiring (1) or expired (2), each stock row had in the user's last summary.
CREATE TABLE push_told (
    user_id INTEGER NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    item_id INTEGER NOT NULL REFERENCES stock_item (id) ON DELETE CASCADE,
    state   INTEGER NOT NULL,
    PRIMARY KEY (user_id, item_id)
);
CREATE INDEX push_told_item_id ON push_told (item_id);

-- +goose Down
DROP TABLE push_told;
DROP TABLE push_choice;
DROP TABLE push_subscription;
DROP TABLE push_key;
