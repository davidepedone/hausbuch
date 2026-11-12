// Package push keeps the browsers' Web Push subscriptions and each user's daily summary of expiring stock, and sends
// the summaries.
package push

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// Store reads and writes subscriptions and choices. Now is the clock, replaceable in tests.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
}

// NewStore returns a Store on the real clock.
func NewStore(db *sql.DB) *Store { return &Store{DB: db, Now: time.Now} }

// Keys are the installation's VAPID key pair, base64url as Web Push expects them.
type Keys struct {
	Public, Private string
}

// Keys returns the installation's VAPID key pair, creating it the first time.
func (s *Store) Keys(ctx context.Context) (Keys, error) {
	var raw []byte
	err := s.DB.QueryRowContext(ctx, "SELECT private_key FROM push_key WHERE id = 1").Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		key, genErr := ecdh.P256().GenerateKey(rand.Reader)
		if genErr != nil {
			return Keys{}, genErr
		}
		// Two servers starting together would both generate one: the first insert wins and both read it back.
		if _, err := s.DB.ExecContext(ctx, "INSERT OR IGNORE INTO push_key (id, private_key) VALUES (1, ?)",
			key.Bytes()); err != nil {
			return Keys{}, err
		}
		err = s.DB.QueryRowContext(ctx, "SELECT private_key FROM push_key WHERE id = 1").Scan(&raw)
	}
	if err != nil {
		return Keys{}, err
	}
	key, err := ecdh.P256().NewPrivateKey(raw)
	if err != nil {
		return Keys{}, err
	}
	enc := base64.RawURLEncoding
	return Keys{Public: enc.EncodeToString(key.PublicKey().Bytes()), Private: enc.EncodeToString(raw)}, nil
}

// Subscription is a browser's Web Push subscription, as PushSubscription.toJSON gives it.
type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// ErrSubscription is a subscription without an HTTPS endpoint or keys.
var ErrSubscription = errors.New("not a push subscription")

// Subscribe stores sub for the session with that hash, replacing any subscription with the same endpoint.
func (s *Store) Subscribe(ctx context.Context, sessionHash []byte, userID int64, sub Subscription) error {
	if !strings.HasPrefix(sub.Endpoint, "https://") || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
		return ErrSubscription
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO push_subscription (endpoint, session_hash, user_id, p256dh, auth, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (endpoint) DO UPDATE SET session_hash = excluded.session_hash, user_id = excluded.user_id,
			p256dh = excluded.p256dh, auth = excluded.auth, created_at = excluded.created_at`,
		sub.Endpoint, sessionHash, userID, sub.Keys.P256dh, sub.Keys.Auth, s.Now().Unix())
	return err
}

// Unsubscribe removes the subscription with that endpoint, if it belongs to the session with that hash.
func (s *Store) Unsubscribe(ctx context.Context, sessionHash []byte, endpoint string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM push_subscription WHERE endpoint = ? AND session_hash = ?",
		endpoint, sessionHash)
	return err
}

// Endpoints returns the endpoints subscribed in the session with that hash.
func (s *Store) Endpoints(ctx context.Context, sessionHash []byte) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT endpoint FROM push_subscription WHERE session_hash = ?", sessionHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var endpoints []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, rows.Err()
}

func (s *Store) subscriptions(ctx context.Context, userID int64) ([]Subscription, error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT endpoint, p256dh, auth FROM push_subscription WHERE user_id = ? ORDER BY created_at", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.Endpoint, &sub.Keys.P256dh, &sub.Keys.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (s *Store) remove(ctx context.Context, endpoint string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM push_subscription WHERE endpoint = ?", endpoint)
	return err
}

// Choice is what a user's daily summary covers and when it is due.
type Choice struct {
	Minutes  int // after midnight, in the installation's time zone
	Sections map[stock.Section]bool
}

// DefaultChoice is a user's choice until they change it: 09:00, every section.
func DefaultChoice() Choice {
	return Choice{Minutes: 9 * 60, Sections: map[stock.Section]bool{stock.Medicines: true, stock.Pantry: true,
		stock.Supplies: true}}
}

// ErrTime is a time of day outside 00:00–23:59.
var ErrTime = errors.New("not a time of day")

// Choice returns the user's choice, or the default.
func (s *Store) Choice(ctx context.Context, userID int64) (Choice, error) {
	var c Choice
	var medicines, pantry, supplies bool
	err := s.DB.QueryRowContext(ctx,
		"SELECT minutes, medicines, pantry, supplies FROM push_choice WHERE user_id = ?", userID).
		Scan(&c.Minutes, &medicines, &pantry, &supplies)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultChoice(), nil
	}
	if err != nil {
		return Choice{}, err
	}
	c.Sections = map[stock.Section]bool{stock.Medicines: medicines, stock.Pantry: pantry, stock.Supplies: supplies}
	return c, nil
}

// SetChoice stores the user's choice.
func (s *Store) SetChoice(ctx context.Context, userID int64, c Choice) error {
	if c.Minutes < 0 || c.Minutes >= 24*60 {
		return ErrTime
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO push_choice (user_id, minutes, medicines, pantry, supplies) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET minutes = excluded.minutes, medicines = excluded.medicines,
			pantry = excluded.pantry, supplies = excluded.supplies`,
		userID, c.Minutes, c.Sections[stock.Medicines], c.Sections[stock.Pantry], c.Sections[stock.Supplies])
	return err
}
