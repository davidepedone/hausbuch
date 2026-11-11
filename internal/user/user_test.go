package user

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/davidepedone/hausbuch/internal/db"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	database, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "hausbuch.db"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return NewStore(database)
}

func TestAddRejectsUsernamesEqualIgnoringCase(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if _, err := s.Add(ctx, "Élise", "Élise", "secret passphrase"); err != nil {
		t.Fatal(err)
	}
	for _, username := range []string{"élise", "ÉLISE", "  élise "} {
		if _, err := s.Add(ctx, username, "Other", "secret passphrase"); !errors.Is(err, ErrUsernameTaken) {
			t.Errorf("Add(%q) = %v, want ErrUsernameTaken", username, err)
		}
	}
	if _, err := s.Add(ctx, "elise", "Elise", "secret passphrase"); err != nil {
		t.Errorf("a username differing by an accent was refused: %v", err)
	}
}

func TestAddRejectsEmptyFields(t *testing.T) {
	s := newStore(t)
	for _, f := range [][3]string{{" ", "Anna", "spoon kettle garden"}, {"anna", " ", "spoon kettle garden"}} {
		if _, err := s.Add(context.Background(), f[0], f[1], f[2]); !errors.Is(err, ErrInvalid) {
			t.Errorf("Add(%q, %q, %q) = %v, want ErrInvalid", f[0], f[1], f[2], err)
		}
	}
}

func TestListOrdersByUsername(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	for _, name := range []string{"marco", "Anna", "bea"} {
		if _, err := s.Add(ctx, name, name, "spoon kettle garden"); err != nil {
			t.Fatal(err)
		}
	}
	users, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, u := range users {
		got = append(got, u.Username)
	}
	if want := []string{"Anna", "bea", "marco"}; !slices.Equal(got, want) {
		t.Errorf("List = %v, want %v", got, want)
	}
}

func TestHashVerifiesOnlyItsOwnPassword(t *testing.T) {
	hash, err := hashPassword(context.Background(), "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	other, err := hashPassword(context.Background(), "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if hash == other {
		t.Error("two hashes of the same password are equal: the salt is not random")
	}
	for password, want := range map[string]bool{"correct horse": true, "correct hors": false, "Correct horse": false, "": false} {
		if ok, err := checkPassword(context.Background(), hash, password); err != nil || ok != want {
			t.Errorf("checkPassword(%q) = %v, %v; want %v", password, ok, err, want)
		}
	}
	if _, err := checkPassword(context.Background(), "$argon2id$v=19$garbage", "x"); err == nil {
		t.Error("a malformed hash was accepted")
	}
}
