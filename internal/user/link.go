package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"
)

var (
	// ErrNoLink means the link signs no one in: unknown, used, expired, replaced, or of a disabled account.
	ErrNoLink   = errors.New("the sign-in link is not valid")
	ErrDisabled = errors.New("the account is disabled")
)

// LinkLifetime is how long a sign-in link is valid.
const LinkLifetime = 15 * time.Minute

// CreateLink returns the token of a new one-time sign-in link for the enabled account with that username, and makes
// its earlier links invalid.
func (s *Store) CreateLink(ctx context.Context, username string) (string, error) {
	var id int64
	var disabled bool
	err := s.DB.QueryRowContext(ctx, "SELECT id, disabled FROM app_user WHERE username_key = ?", usernameKey(username)).
		Scan(&id, &disabled)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", ErrNoSuchUser
	case err != nil:
		return "", err
	case disabled:
		return "", ErrDisabled
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }() // a no-op after Commit
	if _, err := tx.ExecContext(ctx, "DELETE FROM sign_in_link WHERE user_id = ? OR expires_at <= ?", id, s.Now().Unix()); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO sign_in_link (token_hash, user_id, expires_at) VALUES (?, ?, ?)",
		tokenHash(token), id, s.Now().Add(LinkLifetime).Unix()); err != nil {
		return "", err
	}
	return token, tx.Commit()
}

// LinkUser returns the account a valid link signs into, without using the link up.
func (s *Store) LinkUser(ctx context.Context, token string) (User, error) {
	return linkUser(s.DB.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.display_name, u.disabled, u.locked_until
		FROM sign_in_link l JOIN app_user u ON u.id = l.user_id
		WHERE l.token_hash = ? AND l.expires_at > ?`, tokenHash(token), s.Now().Unix()))
}

// UseLink uses a valid link up and returns the account it signs into.
func (s *Store) UseLink(ctx context.Context, token string) (User, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback() }() // a no-op after Commit
	var id int64
	err = tx.QueryRowContext(ctx, "DELETE FROM sign_in_link WHERE token_hash = ? AND expires_at > ? RETURNING user_id",
		tokenHash(token), s.Now().Unix()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNoLink
	}
	if err != nil {
		return User{}, err
	}
	u, err := linkUser(tx.QueryRowContext(ctx,
		"SELECT id, username, display_name, disabled, locked_until FROM app_user WHERE id = ?", id))
	if err != nil {
		return User{}, err
	}
	return u, tx.Commit()
}

// linkUser is the account a link's row names, or ErrNoLink when there is none or it is disabled.
func linkUser(row *sql.Row) (User, error) {
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) || err == nil && u.Disabled {
		return User{}, ErrNoLink
	}
	return u, err
}
