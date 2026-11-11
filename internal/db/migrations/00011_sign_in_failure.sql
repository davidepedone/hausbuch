-- +goose Up
-- Failed password sign-ins per username and address (an IPv4 address or an IPv6 /64), so that failures from one
-- address lock the username only from there. Locks running at the upgrade are dropped.
CREATE TABLE sign_in_failure (
    username_key    TEXT    NOT NULL,
    address         TEXT    NOT NULL,
    failures        INTEGER NOT NULL,
    -- Unix seconds; NULL when not locked.
    locked_until    INTEGER,
    -- Unix seconds.
    last_failure_at INTEGER NOT NULL,
    PRIMARY KEY (username_key, address)
);

ALTER TABLE app_user DROP COLUMN failed_sign_ins;
ALTER TABLE app_user DROP COLUMN locked_until;

-- +goose Down
ALTER TABLE app_user ADD COLUMN failed_sign_ins INTEGER NOT NULL DEFAULT 0;
-- Unix seconds; NULL when not locked.
ALTER TABLE app_user ADD COLUMN locked_until INTEGER;
DROP TABLE sign_in_failure;
