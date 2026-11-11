package user

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSignInLink(t *testing.T) {
	s, now := newStore(t), time.Date(2026, 11, 10, 9, 0, 0, 0, time.Local)
	s.Now = func() time.Time { return now }
	ctx := context.Background()
	anna := mustAdd(t, s, "anna", "Anna", "spoon kettle garden")
	token, err := s.CreateLink(ctx, "ANNA")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(5 * time.Minute)
	// Looking doesn't use it up.
	for range 2 {
		if u, err := s.LinkUser(ctx, token); err != nil || u.ID != anna.ID {
			t.Fatalf("link user %+v, %v", u, err)
		}
	}
	if u, err := s.UseLink(ctx, token); err != nil || u.ID != anna.ID {
		t.Fatalf("use %+v, %v", u, err)
	}
	// Link used twice.
	if _, err := s.UseLink(ctx, token); !errors.Is(err, ErrNoLink) {
		t.Errorf("used twice: %v", err)
	}
	if _, err := s.LinkUser(ctx, token); !errors.Is(err, ErrNoLink) {
		t.Errorf("used link shown: %v", err)
	}
}

func TestSignInLinkExpires(t *testing.T) {
	s, now := newStore(t), time.Date(2026, 11, 10, 9, 0, 0, 0, time.Local)
	s.Now = func() time.Time { return now }
	mustAdd(t, s, "anna", "Anna", "spoon kettle garden")
	token, _ := s.CreateLink(context.Background(), "anna")
	now = now.Add(16 * time.Minute)
	if _, err := s.UseLink(context.Background(), token); !errors.Is(err, ErrNoLink) {
		t.Errorf("expired: %v", err)
	}
}

func TestNewerLinkReplacesTheOlder(t *testing.T) {
	s := newStore(t)
	mustAdd(t, s, "anna", "Anna", "spoon kettle garden")
	mustAdd(t, s, "marco", "Marco", "spoon kettle garden")
	ctx := context.Background()
	first, _ := s.CreateLink(ctx, "anna")
	marcos, _ := s.CreateLink(ctx, "marco")
	second, _ := s.CreateLink(ctx, "anna")
	if _, err := s.UseLink(ctx, first); !errors.Is(err, ErrNoLink) {
		t.Errorf("first link: %v", err)
	}
	for _, token := range []string{second, marcos} {
		if _, err := s.UseLink(ctx, token); err != nil {
			t.Errorf("still valid link: %v", err)
		}
	}
}

func TestNoLinkForADisabledAccount(t *testing.T) {
	s := newStore(t)
	mustAdd(t, s, "anna", "Anna", "spoon kettle garden")
	ctx := context.Background()
	if _, err := s.CreateLink(ctx, "nobody"); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("unknown user: %v", err)
	}
	token, _ := s.CreateLink(ctx, "anna")
	if err := s.Disable(ctx, "anna"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateLink(ctx, "anna"); !errors.Is(err, ErrDisabled) {
		t.Errorf("new link for a disabled account: %v", err)
	}
	if _, err := s.UseLink(ctx, token); !errors.Is(err, ErrNoLink) {
		t.Errorf("link of a disabled account: %v", err)
	}
}

func mustAdd(t *testing.T, s *Store, username, displayName, password string) User {
	t.Helper()
	u, err := s.Add(context.Background(), username, displayName, password)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
