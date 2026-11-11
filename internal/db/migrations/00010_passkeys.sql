-- +goose Up
-- The WebAuthn user handle: random, so that authenticators don't learn the order of accounts. NULL until the account's
-- first passkey.
ALTER TABLE app_user ADD COLUMN webauthn_id BLOB;
CREATE UNIQUE INDEX app_user_webauthn_id ON app_user (webauthn_id);

CREATE TABLE passkey (
    credential_id BLOB    PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    -- The WebAuthn library's credential record, as JSON: public key, flags, sign counter.
    credential    TEXT    NOT NULL,
    -- Unix seconds.
    created_at    INTEGER NOT NULL,
    last_used_at  INTEGER NOT NULL
);
CREATE INDEX passkey_user_id ON passkey (user_id);

CREATE TABLE sign_in_link (
    -- SHA-256 of the token in the link, as for sessions.
    token_hash BLOB    PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    -- Unix seconds.
    expires_at INTEGER NOT NULL
);
CREATE INDEX sign_in_link_user_id ON sign_in_link (user_id);

-- +goose Down
DROP TABLE sign_in_link;
DROP TABLE passkey;
DROP INDEX app_user_webauthn_id;
ALTER TABLE app_user DROP COLUMN webauthn_id;
