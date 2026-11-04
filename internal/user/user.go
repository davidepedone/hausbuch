// Package user holds residents' accounts, their passwords and their sessions.
package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	ErrUsernameTaken  = errors.New("username is taken")
	ErrInvalid        = errors.New("username, display name and password must not be empty")
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
	if username == "" || displayName == "" || password == "" {
		return User{}, ErrInvalid
	}
	hash, err := hashPassword(password)
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

// Sign-in lock: after maxFailedSignIns wrong passwords in a row a username is refused, even with the right password,
// for lockDuration.
const (
	maxFailedSignIns = 3
	lockDuration     = 15 * time.Minute
)

// Authenticate returns the enabled, unlocked account with that username and password, counting wrong passwords
// towards the sign-in lock. Every failure is ErrBadCredentials, so callers can't tell an unknown username, a wrong
// password, a disabled account or a lock apart.
func (s *Store) Authenticate(ctx context.Context, username, password string) (User, error) {
	row := s.DB.QueryRowContext(ctx,
		"SELECT id, username, display_name, disabled, locked_until, password_hash FROM app_user WHERE username_key = ?",
		usernameKey(username))
	var u User
	var lockedUntil sql.NullInt64
	var hash string
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Disabled, &lockedUntil, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		// Hash anyway, so an unknown username takes as long to refuse as a wrong password.
		_, _ = checkPassword(dummyHash(), password)
		return User{}, ErrBadCredentials
	}
	if err != nil {
		return User{}, err
	}
	ok, err := checkPassword(hash, password)
	if err != nil {
		return User{}, err
	}
	now := s.Now()
	switch {
	case lockedUntil.Valid && lockedUntil.Int64 > now.Unix():
		// Attempts during a lock don't extend it: it ends lockDuration after the failure that started it.
		return User{}, ErrBadCredentials
	case !ok:
		_, err := s.DB.ExecContext(ctx, `
			UPDATE app_user SET
				locked_until    = CASE WHEN failed_sign_ins + 1 >= ? THEN ? ELSE locked_until END,
				failed_sign_ins = CASE WHEN failed_sign_ins + 1 >= ? THEN 0 ELSE failed_sign_ins + 1 END
			WHERE id = ?`, maxFailedSignIns, now.Add(lockDuration).Unix(), maxFailedSignIns, u.ID)
		if err != nil {
			return User{}, err
		}
		return User{}, ErrBadCredentials
	case u.Disabled:
		return User{}, ErrBadCredentials
	}
	if _, err := s.DB.ExecContext(ctx,
		"UPDATE app_user SET failed_sign_ins = 0, locked_until = NULL WHERE id = ?", u.ID); err != nil {
		return User{}, err
	}
	return u, nil
}

// Unlock lifts the sign-in lock of an account and forgets its failed attempts.
func (s *Store) Unlock(ctx context.Context, username string) error {
	return s.update(ctx, username, "failed_sign_ins = 0, locked_until = NULL")
}

// SetPassword replaces an account's password.
func (s *Store) SetPassword(ctx context.Context, username, password string) error {
	if password == "" {
		return ErrInvalid
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return s.update(ctx, username, "password_hash = ?", hash)
}

// ChangePassword replaces the password of the account with that id, given its current password. A wrong current
// password is ErrBadCredentials.
func (s *Store) ChangePassword(ctx context.Context, id int64, current, password string) error {
	if password == "" {
		return ErrInvalid
	}
	var hash string
	if err := s.DB.QueryRowContext(ctx, "SELECT password_hash FROM app_user WHERE id = ?", id).Scan(&hash); err != nil {
		return err
	}
	ok, err := checkPassword(hash, current)
	if err != nil {
		return err
	}
	if !ok {
		return ErrBadCredentials
	}
	newHash, err := hashPassword(password)
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

var dummyHash = sync.OnceValue(func() string {
	h, err := hashPassword("")
	if err != nil {
		panic(err)
	}
	return h
})

// List returns every account, ordered by username.
func (s *Store) List(ctx context.Context) ([]User, error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT id, username, display_name, disabled, locked_until FROM app_user ORDER BY username_key")
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
