package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"
)

// ErrNoSession means the token belongs to no valid session: unknown, expired, or of a disabled account.
var ErrNoSession = errors.New("no valid session")

// A session's expiry moves forward on use, but it is rewritten at most this often, so that reading pages doesn't
// always mean writing to the database.
const sessionRefreshInterval = time.Hour

// Session is a signed-in user's session.
type Session struct {
	Token   string
	User    User
	Expires time.Time
}

func tokenHash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// CreateSession starts a session for the user, valid for s.SessionTimeout, and removes expired ones.
func (s *Store) CreateSession(ctx context.Context, u User) (Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	now := s.Now()
	expires := now.Add(s.SessionTimeout)
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM session WHERE expires_at <= ?", now.Unix()); err != nil {
		return Session{}, err
	}
	_, err := s.DB.ExecContext(ctx,
		"INSERT INTO session (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
		tokenHash(token), u.ID, now.Unix(), expires.Unix())
	if err != nil {
		return Session{}, err
	}
	return Session{Token: token, User: u, Expires: expires}, nil
}

// LookupSession returns the valid session with that token, moving its expiry forward when it is due. The returned
// Expires is the one now stored, so the caller can tell whether to refresh the cookie.
func (s *Store) LookupSession(ctx context.Context, token string) (Session, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.display_name, u.disabled, u.locked_until, s.expires_at
		FROM session s JOIN app_user u ON u.id = s.user_id
		WHERE s.token_hash = ?`, tokenHash(token))
	var u User
	var lockedUntil sql.NullInt64
	var expiresAt int64
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Disabled, &lockedUntil, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNoSession
	}
	if err != nil {
		return Session{}, err
	}
	if lockedUntil.Valid {
		u.LockedUntil = time.Unix(lockedUntil.Int64, 0)
	}
	now := s.Now()
	expires := time.Unix(expiresAt, 0)
	if !expires.After(now) || u.Disabled {
		return Session{}, ErrNoSession
	}
	if fresh := now.Add(s.SessionTimeout); fresh.Sub(expires) >= sessionRefreshInterval {
		if _, err := s.DB.ExecContext(ctx, "UPDATE session SET expires_at = ? WHERE token_hash = ?",
			fresh.Unix(), tokenHash(token)); err != nil {
			return Session{}, err
		}
		expires = fresh
	}
	return Session{Token: token, User: u, Expires: expires}, nil
}

// DeleteSession ends the session with that token, if there is one.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM session WHERE token_hash = ?", tokenHash(token))
	return err
}

// DeleteExpiredSessions removes sessions past their expiry.
func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM session WHERE expires_at <= ?", s.Now().Unix())
	return err
}
