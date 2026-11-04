package user

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
)

func failSignIns(t *testing.T, s *Store, n int) {
	t.Helper()
	for range n {
		if _, err := s.Authenticate(context.Background(), "anna", "wrong"); !errors.Is(err, ErrBadCredentials) {
			t.Fatalf("wrong password: %v", err)
		}
	}
}

func TestLockAfterFailures(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	signedIn(t, s)

	failSignIns(t, s, 3)
	c.advance(14 * time.Minute)
	if _, err := s.Authenticate(context.Background(), "anna", "pw"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("right password 14 minutes after 3 failures: %v, want ErrBadCredentials", err)
	}
}

func TestTwoFailuresDoNotLock(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 2)
	if _, err := s.Authenticate(context.Background(), "anna", "pw"); err != nil {
		t.Fatalf("right password after 2 failures: %v", err)
	}
}

func TestOnlyFailuresInARowCount(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 2)
	if _, err := s.Authenticate(context.Background(), "anna", "pw"); err != nil {
		t.Fatal(err)
	}
	failSignIns(t, s, 2)
	if _, err := s.Authenticate(context.Background(), "anna", "pw"); err != nil {
		t.Fatalf("2 failures, a success, 2 failures locked the account: %v", err)
	}
}

func TestLockExpires(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	signedIn(t, s)

	failSignIns(t, s, 3)
	c.advance(10 * time.Minute)
	failSignIns(t, s, 1) // during the lock: doesn't extend it
	c.advance(5 * time.Minute)
	if _, err := s.Authenticate(context.Background(), "anna", "pw"); err != nil {
		t.Fatalf("right password 15 minutes after the last of the 3 failures: %v", err)
	}
}

func TestUnlock(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 3)
	if err := s.Unlock(context.Background(), "Anna"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(context.Background(), "anna", "pw"); err != nil {
		t.Fatalf("right password right after unlock: %v", err)
	}
	if err := s.Unlock(context.Background(), "nobody"); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("Unlock(unknown) = %v, want ErrNoSuchUser", err)
	}
}

func TestLockSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hausbuch.db")
	first, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	s := NewStore(first)
	signedIn(t, s)
	failSignIns(t, s, 3)
	_ = first.Close()

	second, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, err := NewStore(second).Authenticate(context.Background(), "anna", "pw"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("right password after a restart during the lock: %v, want ErrBadCredentials", err)
	}
}
