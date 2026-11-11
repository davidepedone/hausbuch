package user

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
)

// Two client addresses, from the ranges reserved for documentation.
const (
	here  = "192.0.2.1"
	there = "198.51.100.7"
)

func failSignIns(t *testing.T, s *Store, n int, address string) {
	t.Helper()
	for range n {
		if _, err := s.Authenticate(context.Background(), "anna", "wrong", address); !errors.Is(err, ErrBadCredentials) {
			t.Fatalf("wrong password: %v", err)
		}
	}
}

func signInFrom(s *Store, address string) error {
	_, err := s.Authenticate(context.Background(), "anna", "spoon kettle garden", address)
	return err
}

func TestLockAfterFailures(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	signedIn(t, s)

	failSignIns(t, s, 3, here)
	c.advance(14 * time.Minute)
	if err := signInFrom(s, here); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("right password 14 minutes after 3 failures: %v, want ErrBadCredentials", err)
	}
}

func TestOtherAddressesUnaffected(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 3, here)
	if err := signInFrom(s, there); err != nil {
		t.Fatalf("right password from another address during a lock: %v", err)
	}
	if err := signInFrom(s, here); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("a sign-in from another address lifted the lock: %v", err)
	}
}

func TestFailuresFromDifferentAddressesDoNotAddUp(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 2, here)
	failSignIns(t, s, 2, there)
	if err := signInFrom(s, here); err != nil {
		t.Fatalf("right password after 2 failures from each of two addresses: %v", err)
	}
}

func TestTwoFailuresDoNotLock(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 2, here)
	if err := signInFrom(s, here); err != nil {
		t.Fatalf("right password after 2 failures: %v", err)
	}
}

func TestOnlyFailuresInARowCount(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 2, here)
	if err := signInFrom(s, here); err != nil {
		t.Fatal(err)
	}
	failSignIns(t, s, 2, here)
	if err := signInFrom(s, here); err != nil {
		t.Fatalf("2 failures, a success, 2 failures locked the account: %v", err)
	}
}

func TestOldFailuresDoNotCount(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	signedIn(t, s)
	failSignIns(t, s, 2, here)
	c.advance(15 * time.Minute)
	failSignIns(t, s, 1, here)
	if err := signInFrom(s, here); err != nil {
		t.Fatalf("2 failures, 15 minutes, 1 failure locked the account: %v", err)
	}
}

func TestLockExpires(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	signedIn(t, s)

	failSignIns(t, s, 3, here)
	c.advance(10 * time.Minute)
	failSignIns(t, s, 1, here) // during the lock: doesn't extend it
	c.advance(5 * time.Minute)
	if err := signInFrom(s, here); err != nil {
		t.Fatalf("right password 15 minutes after the last of the 3 failures: %v", err)
	}
}

func TestUnlock(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 3, here)
	failSignIns(t, s, 3, there)
	if err := s.Unlock(context.Background(), "Anna"); err != nil {
		t.Fatal(err)
	}
	for _, address := range []string{here, there} {
		if err := signInFrom(s, address); err != nil {
			t.Fatalf("right password from %s right after unlock: %v", address, err)
		}
	}
	if err := s.Unlock(context.Background(), "nobody"); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("Unlock(unknown) = %v, want ErrNoSuchUser", err)
	}
}

func TestLockSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hausbuch.db")
	first, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := NewStore(first)
	signedIn(t, s)
	failSignIns(t, s, 3, here)
	_ = first.Close()

	second, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if err := signInFrom(NewStore(second), here); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("right password after a restart during the lock: %v, want ErrBadCredentials", err)
	}
}

func failureRows(t *testing.T, s *Store) int {
	t.Helper()
	var n int
	if err := s.DB.QueryRow("SELECT count(*) FROM sign_in_failure").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestUnknownUsernamesLeaveNoRow(t *testing.T) {
	s := newStore(t)
	for range 3 {
		if _, err := s.Authenticate(context.Background(), "nobody", "spoon kettle garden", here); !errors.Is(err, ErrBadCredentials) {
			t.Fatal(err)
		}
	}
	if n := failureRows(t, s); n != 0 {
		t.Errorf("%d rows of failures for an unknown username, want 0", n)
	}
}

func TestListShowsALockFromAnyAddress(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	signedIn(t, s)
	failSignIns(t, s, 3, there)
	users, err := s.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := c.now().Add(lockDuration); !users[0].LockedUntil.Equal(want) {
		t.Errorf("LockedUntil = %v, want %v", users[0].LockedUntil, want)
	}
}

func TestDeleteStaleFailures(t *testing.T) {
	s, c := newStore(t), newClock()
	s.Now = c.now
	signedIn(t, s)
	failSignIns(t, s, 3, here)  // a lock until 15 minutes from now
	c.advance(10 * time.Minute) // a failure that still counts
	failSignIns(t, s, 1, there)
	if err := s.DeleteStaleFailures(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := failureRows(t, s); n != 2 {
		t.Fatalf("%d rows after removing stale failures with a lock and a recent failure, want 2", n)
	}
	c.advance(15 * time.Minute)
	if err := s.DeleteStaleFailures(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := failureRows(t, s); n != 0 {
		t.Errorf("%d rows once the lock ended and the failure is old, want 0", n)
	}
}

func TestTooManyPasswordChecksAtOnce(t *testing.T) {
	s := newStore(t)
	signedIn(t, s)
	failSignIns(t, s, 1, here)
	release := TakePasswordChecks(50 * time.Millisecond) // two password checks running
	_, err := s.Authenticate(context.Background(), "anna", "wrong", here)
	release()
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("sign-in while both checks are taken: %v, want ErrBusy", err)
	}
	// The refused attempt didn't count: two more failures are needed for a lock, so one more leaves it open.
	failSignIns(t, s, 1, here)
	if err := signInFrom(s, here); err != nil {
		t.Fatalf("the busy attempt counted as a failure: %v", err)
	}
}
