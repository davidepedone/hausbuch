package user

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestWebAuthnIDIsStable(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	anna, marco := mustAdd(t, s, "anna", "Anna", "pw"), mustAdd(t, s, "marco", "Marco", "pw")
	first, err := s.WebAuthnID(ctx, anna.ID)
	if err != nil || len(first) != 32 {
		t.Fatalf("handle %x, %v", first, err)
	}
	again, _ := s.WebAuthnID(ctx, anna.ID)
	other, _ := s.WebAuthnID(ctx, marco.ID)
	if !bytes.Equal(first, again) || bytes.Equal(first, other) {
		t.Errorf("anna %x then %x, marco %x", first, again, other)
	}
	if u, err := s.UserByHandle(ctx, first); err != nil || u.ID != anna.ID {
		t.Errorf("by handle: %+v, %v", u, err)
	}
	if _, err := s.UserByHandle(ctx, []byte("unknown")); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("unknown handle: %v", err)
	}
}

func TestPasskeys(t *testing.T) {
	s, now := newStore(t), time.Date(2026, 11, 10, 9, 0, 0, 0, time.Local)
	s.Now = func() time.Time { return now }
	ctx := context.Background()
	anna, marco := mustAdd(t, s, "anna", "Anna", "pw"), mustAdd(t, s, "marco", "Marco", "pw")
	if err := s.AddPasskey(ctx, anna.ID, []byte("k1"), []byte(`{"n":1}`)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	if err := s.AddPasskey(ctx, anna.ID, []byte("k2"), []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(24 * time.Hour)
	if err := s.UsePasskey(ctx, []byte("k1"), []byte(`{"n":2}`)); err != nil {
		t.Fatal(err)
	}
	keys, err := s.Passkeys(ctx, anna.ID)
	if err != nil || len(keys) != 2 {
		t.Fatalf("keys %+v, %v", keys, err)
	}
	if string(keys[0].ID) != "k1" || string(keys[0].Credential) != `{"n":2}` || !keys[0].LastUsedAt.Equal(now) ||
		!keys[0].CreatedAt.Equal(now.Add(-25*time.Hour)) || string(keys[1].ID) != "k2" {
		t.Errorf("keys %+v", keys)
	}

	// Removing another account's passkey is refused.
	if err := s.RemovePasskey(ctx, marco.ID, []byte("k1")); !errors.Is(err, ErrNoPasskey) {
		t.Errorf("marco removed anna's: %v", err)
	}
	if err := s.RemovePasskey(ctx, anna.ID, []byte("k1")); err != nil {
		t.Fatal(err)
	}
	if keys, _ := s.Passkeys(ctx, anna.ID); len(keys) != 1 || string(keys[0].ID) != "k2" {
		t.Errorf("after removing: %+v", keys)
	}
	if _, err := s.Authenticate(ctx, "anna", "pw"); err != nil {
		t.Errorf("password after removing a passkey: %v", err)
	}
}
