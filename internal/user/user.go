// Package user holds residents' accounts, their passwords and their sessions.
package user

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	ErrUsernameTaken  = errors.New("username is taken")
	ErrInvalid        = errors.New("username and display name must not be empty")
	ErrBadCredentials = errors.New("invalid username or password")
	ErrNoSuchUser     = errors.New("no account with that username")
)

// User is a resident's account.
type User struct {
	ID          int64
	Username    string
	DisplayName string
	Disabled    bool
	LockedUntil time.Time // zero when not locked
}

// DefaultSessionTimeout is how long a session lasts without being used.
const DefaultSessionTimeout = 30 * 24 * time.Hour

// Store reads and writes accounts and sessions. Now is the clock, replaceable in tests.
type Store struct {
	DB             *sql.DB
	Now            func() time.Time
	SessionTimeout time.Duration
}

// NewStore returns a Store on db using the real clock and the default session timeout.
func NewStore(db *sql.DB) *Store {
	return &Store{DB: db, Now: time.Now, SessionTimeout: DefaultSessionTimeout}
}

func usernameKey(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// Add creates an account. Usernames are unique ignoring case and surrounding whitespace.
func (s *Store) Add(ctx context.Context, username, displayName, password string) (User, error) {
	username, displayName = strings.TrimSpace(username), strings.TrimSpace(displayName)
	if username == "" || displayName == "" {
		return User{}, ErrInvalid
	}
	if err := checkNewPassword(password, username, displayName); err != nil {
		return User{}, err
	}
	hash, err := hashPassword(ctx, password)
	if err != nil {
		return User{}, err
	}
	res, err := s.DB.ExecContext(ctx,
		"INSERT INTO app_user (username, username_key, display_name, password_hash) VALUES (?, ?, ?, ?)",
		username, usernameKey(username), displayName, hash)
	if isUniqueViolation(err) {
		return User{}, ErrUsernameTaken
	}
	if err != nil {
		return User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Username: username, DisplayName: displayName}, nil
}

// Sign-in lock: after maxFailedSignIns wrong passwords in a row for a username from one address, that username is
// refused from that address, even with the right password, for lockDuration. Failures older than lockDuration don't
// count.
const (
	maxFailedSignIns = 3
	lockDuration     = 15 * time.Minute
)

// lockedUntilColumn is the end of an account's latest lock from any address, as a column of a query on app_user u.
const lockedUntilColumn = "(SELECT MAX(f.locked_until) FROM sign_in_failure f WHERE f.username_key = u.username_key)"

// Authenticate returns the enabled account with that username and password, unless the username is locked from
// address, counting wrong passwords towards that lock. Every failure is ErrBadCredentials, so callers can't tell an
// unknown username, a wrong password, a disabled account or a lock apart.
func (s *Store) Authenticate(ctx context.Context, username, password, address string) (User, error) {
	key := usernameKey(username)
	row := s.DB.QueryRowContext(ctx,
		"SELECT id, username, display_name, disabled, password_hash FROM app_user WHERE username_key = ?", key)
	var u User
	var hash string
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Disabled, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		// Hash anyway, so an unknown username takes as long to refuse as a wrong password. It leaves no row behind,
		// so the failures table can't be filled with made-up names.
		if _, err := checkPassword(ctx, dummyHash, password); err != nil {
			return User{}, err
		}
		// Not what was typed: a username field often gets a password typed in the wrong place.
		failed("unknown user", "unknown", address)
		return User{}, ErrBadCredentials
	}
	if err != nil {
		return User{}, err
	}
	ok, err := checkPassword(ctx, hash, password)
	if err != nil {
		return User{}, err
	}
	// One transaction for reading and writing the count, so that concurrent attempts can't share it.
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback() }() // a no-op after Commit
	var failures int
	var lockedUntil sql.NullInt64
	var lastFailure int64
	err = tx.QueryRowContext(ctx,
		"SELECT failures, locked_until, last_failure_at FROM sign_in_failure WHERE username_key = ? AND address = ?",
		key, address).Scan(&failures, &lockedUntil, &lastFailure)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}
	now := s.Now()
	switch {
	case lockedUntil.Valid && lockedUntil.Int64 > now.Unix():
		// Attempts during a lock don't extend it: it ends lockDuration after the failure that started it.
		failed("locked", u.Username, address)
		return User{}, ErrBadCredentials
	case !ok:
		if lockedUntil.Valid || now.Unix()-lastFailure >= int64(lockDuration.Seconds()) {
			failures = 0 // after a lock, or after a pause, the count starts again
		}
		failures++
		var lock any
		if failures >= maxFailedSignIns {
			lock = now.Add(lockDuration).Unix()
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sign_in_failure (username_key, address, failures, locked_until, last_failure_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (username_key, address) DO UPDATE SET
				failures = excluded.failures, locked_until = excluded.locked_until,
				last_failure_at = excluded.last_failure_at`, key, address, failures, lock, now.Unix()); err != nil {
			return User{}, err
		}
		if err := tx.Commit(); err != nil {
			return User{}, err
		}
		failed("wrong password", u.Username, address)
		if lock != nil {
			slog.Warn("sign-in lock started", "user", u.Username, "address", address)
		}
		return User{}, ErrBadCredentials
	case u.Disabled:
		failed("disabled", u.Username, address)
		return User{}, ErrBadCredentials
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM sign_in_failure WHERE username_key = ? AND address = ?", key, address); err != nil {
		return User{}, err
	}
	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	slog.Info("signed in", "method", "password", "user", u.Username, "address", address)
	return u, nil
}

// failed logs a password sign-in refused for reason.
func failed(reason, username, address string) {
	slog.Warn("sign-in failed", "method", "password", "reason", reason, "user", username, "address", address)
}

// Unlock lifts the sign-in locks of an account from every address and forgets its failed attempts.
func (s *Store) Unlock(ctx context.Context, username string) error {
	var exists bool
	if err := s.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM app_user WHERE username_key = ?)",
		usernameKey(username)).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNoSuchUser
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM sign_in_failure WHERE username_key = ?", usernameKey(username))
	return err
}

// DeleteStaleFailures removes the failed sign-ins that no longer count: not locking, and older than lockDuration.
func (s *Store) DeleteStaleFailures(ctx context.Context) error {
	now := s.Now()
	_, err := s.DB.ExecContext(ctx,
		"DELETE FROM sign_in_failure WHERE coalesce(locked_until, 0) <= ? AND last_failure_at <= ?",
		now.Unix(), now.Add(-lockDuration).Unix())
	return err
}

// SetPassword replaces an account's password.
func (s *Store) SetPassword(ctx context.Context, username, password string) error {
	var name, displayName string
	err := s.DB.QueryRowContext(ctx, "SELECT username, display_name FROM app_user WHERE username_key = ?",
		usernameKey(username)).Scan(&name, &displayName)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoSuchUser
	}
	if err != nil {
		return err
	}
	if err := checkNewPassword(password, name, displayName); err != nil {
		return err
	}
	hash, err := hashPassword(ctx, password)
	if err != nil {
		return err
	}
	return s.update(ctx, username, "password_hash = ?", hash)
}

// ChangePassword replaces the password of the account with that id, given its current password. A wrong current
// password is ErrBadCredentials.
func (s *Store) ChangePassword(ctx context.Context, id int64, current, password string) error {
	var username, displayName, hash string
	if err := s.DB.QueryRowContext(ctx, "SELECT username, display_name, password_hash FROM app_user WHERE id = ?", id).
		Scan(&username, &displayName, &hash); err != nil {
		return err
	}
	if err := checkNewPassword(password, username, displayName); err != nil {
		return err
	}
	ok, err := checkPassword(ctx, hash, current)
	if err != nil {
		return err
	}
	if !ok {
		return ErrBadCredentials
	}
	newHash, err := hashPassword(ctx, password)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, "UPDATE app_user SET password_hash = ? WHERE id = ?", newHash, id)
	return err
}

// Disable stops an account from signing in and ends its sessions. Its past edits keep its name.
func (s *Store) Disable(ctx context.Context, username string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // a no-op after Commit
	res, err := tx.ExecContext(ctx, "UPDATE app_user SET disabled = 1 WHERE username_key = ?", usernameKey(username))
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNoSuchUser
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM session WHERE user_id = (SELECT id FROM app_user WHERE username_key = ?)",
		usernameKey(username)); err != nil {
		return err
	}
	return tx.Commit()
}

// Enable lets a disabled account sign in again.
func (s *Store) Enable(ctx context.Context, username string) error {
	return s.update(ctx, username, "disabled = 0")
}

// update applies set to the account with that username, or returns ErrNoSuchUser.
func (s *Store) update(ctx context.Context, username, set string, args ...any) error {
	res, err := s.DB.ExecContext(ctx, "UPDATE app_user SET "+set+" WHERE username_key = ?",
		append(args, usernameKey(username))...)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNoSuchUser
	}
	return nil
}

// List returns every account, ordered by username.
func (s *Store) List(ctx context.Context) ([]User, error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT u.id, u.username, u.display_name, u.disabled, "+lockedUntilColumn+" FROM app_user u ORDER BY u.username_key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func scanUser(row interface{ Scan(...any) error }) (User, error) {
	var u User
	var lockedUntil sql.NullInt64
	if err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Disabled, &lockedUntil); err != nil {
		return User{}, err
	}
	if lockedUntil.Valid {
		u.LockedUntil = time.Unix(lockedUntil.Int64, 0)
	}
	return u, nil
}

func isUniqueViolation(err error) bool {
	var e *sqlite.Error
	return errors.As(err, &e) && e.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}
