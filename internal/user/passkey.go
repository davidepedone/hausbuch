package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"time"
)

// ErrNoPasskey means no passkey of the account has that credential id.
var ErrNoPasskey = errors.New("no such passkey")

// Passkey is one of an account's passkeys. Credential is the WebAuthn library's record of it, as JSON.
type Passkey struct {
	ID         []byte // the credential id
	Credential []byte
	CreatedAt  time.Time
	LastUsedAt time.Time
}

// WebAuthnID returns the account's WebAuthn user handle, creating it on first use.
func (s *Store) WebAuthnID(ctx context.Context, userID int64) ([]byte, error) {
	handle := make([]byte, 32)
	if _, err := rand.Read(handle); err != nil {
		return nil, err
	}
	if _, err := s.DB.ExecContext(ctx, "UPDATE app_user SET webauthn_id = ? WHERE id = ? AND webauthn_id IS NULL",
		handle, userID); err != nil {
		return nil, err
	}
	err := s.DB.QueryRowContext(ctx, "SELECT webauthn_id FROM app_user WHERE id = ?", userID).Scan(&handle)
	return handle, err
}

// UserByHandle returns the account with that WebAuthn user handle, disabled or not, or ErrNoSuchUser.
func (s *Store) UserByHandle(ctx context.Context, handle []byte) (User, error) {
	u, err := scanUser(s.DB.QueryRowContext(ctx,
		"SELECT id, username, display_name, disabled, locked_until FROM app_user WHERE webauthn_id = ?", handle))
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNoSuchUser
	}
	return u, err
}

// Passkeys returns the account's passkeys, oldest first.
func (s *Store) Passkeys(ctx context.Context, userID int64) ([]Passkey, error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT credential_id, credential, created_at, last_used_at FROM passkey WHERE user_id = ? ORDER BY created_at, rowid",
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []Passkey
	for rows.Next() {
		var k Passkey
		var created, used int64
		if err := rows.Scan(&k.ID, &k.Credential, &created, &used); err != nil {
			return nil, err
		}
		k.CreatedAt, k.LastUsedAt = time.Unix(created, 0), time.Unix(used, 0)
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// AddPasskey stores a new passkey of the account.
func (s *Store) AddPasskey(ctx context.Context, userID int64, id, credential []byte) error {
	now := s.Now().Unix()
	_, err := s.DB.ExecContext(ctx,
		"INSERT INTO passkey (credential_id, user_id, credential, created_at, last_used_at) VALUES (?, ?, ?, ?, ?)",
		id, userID, credential, now, now)
	return err
}

// UsePasskey records a sign-in with the passkey: its updated record, holding the new sign counter, and the time.
func (s *Store) UsePasskey(ctx context.Context, id, credential []byte) error {
	return s.changePasskey(ctx, "UPDATE passkey SET credential = ?, last_used_at = ? WHERE credential_id = ?",
		credential, s.Now().Unix(), id)
}

// RemovePasskey removes one of the account's passkeys; another account's is ErrNoPasskey.
func (s *Store) RemovePasskey(ctx context.Context, userID int64, id []byte) error {
	return s.changePasskey(ctx, "DELETE FROM passkey WHERE user_id = ? AND credential_id = ?", userID, id)
}

func (s *Store) changePasskey(ctx context.Context, query string, args ...any) error {
	res, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNoPasskey
	}
	return nil
}
