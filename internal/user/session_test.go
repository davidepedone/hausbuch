package user

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
)

// clock is a settable time source for Store.Now.
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newClock() *clock                   { return &clock{t: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)} }

const day = 24 * time.Hour

func signedIn(t *testing.T, s *Store) Session {
	t.Helper()
	u, err := s.Add(context.Background(), "anna", "Anna", "spoon kettle garden")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := s.CreateSession(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	return sess
}

func TestSessionValidWithinTimeout(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	sess := signedIn(t, s)

	c.advance(29 * day)
	got, err := s.LookupSession(context.Background(), sess.Token)
	if err != nil {
		t.Fatalf("session 29 days old with a 30-day timeout: %v", err)
	}
	if got.User.Username != "anna" {
		t.Errorf("session user = %q", got.User.Username)
	}
}

func TestRegularUseKeepsSession(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	sess := signedIn(t, s)

	for i := range 5 {
		c.advance(20 * day)
		if _, err := s.LookupSession(context.Background(), sess.Token); err != nil {
			t.Fatalf("request %d, %d days after signing in: %v", i+1, (i+1)*20, err)
		}
	}
}

func TestSessionExpires(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	sess := signedIn(t, s)

	c.advance(30*day + time.Second)
	if _, err := s.LookupSession(context.Background(), sess.Token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("session unused for 30 days: %v, want ErrNoSession", err)
	}
}

func TestExpiryIsRewrittenAtMostHourly(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	sess := signedIn(t, s)

	c.advance(30 * time.Minute)
	got, err := s.LookupSession(context.Background(), sess.Token)
	if err != nil || !got.Expires.Equal(sess.Expires) {
		t.Errorf("after 30 minutes: expires %v (%v), want unchanged %v", got.Expires, err, sess.Expires)
	}
	c.advance(time.Hour)
	got, err = s.LookupSession(context.Background(), sess.Token)
	if err != nil || !got.Expires.Equal(c.t.Add(s.SessionTimeout)) {
		t.Errorf("after 90 minutes: expires %v (%v), want %v", got.Expires, err, c.t.Add(s.SessionTimeout))
	}
}

func TestSessionOfDisabledUser(t *testing.T) {
	s := newStore(t)
	sess := signedIn(t, s)
	if _, err := s.DB.Exec("UPDATE app_user SET disabled = 1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LookupSession(context.Background(), sess.Token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("session of a disabled user: %v, want ErrNoSession", err)
	}
}

func TestDeleteSession(t *testing.T) {
	s := newStore(t)
	sess := signedIn(t, s)
	if err := s.DeleteSession(context.Background(), sess.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LookupSession(context.Background(), sess.Token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("deleted session: %v, want ErrNoSession", err)
	}
}

func TestSessionSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hausbuch.db")
	first, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sess := signedIn(t, NewStore(first))
	_ = first.Close()

	second, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err := NewStore(second).LookupSession(context.Background(), sess.Token); err != nil {
		t.Fatalf("session after reopening the database: %v", err)
	}
}

func TestOnlyTheTokenHashIsStored(t *testing.T) {
	s := newStore(t)
	sess := signedIn(t, s)
	var stored []byte
	if err := s.DB.QueryRow("SELECT token_hash FROM session").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stored, []byte(sess.Token)) || len(stored) != 32 {
		t.Errorf("stored %x for token %q, want its 32-byte SHA-256", stored, sess.Token)
	}
}

func TestExpiredSessionsAreRemoved(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	signedIn(t, s)
	c.advance(31 * day)
	if err := s.DeleteExpiredSessions(context.Background()); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.DB.QueryRow("SELECT count(*) FROM session").Scan(&n); err != nil || n != 0 {
		t.Errorf("%d sessions left (%v), want 0", n, err)
	}
}
