package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/user"
)

func newStore(t *testing.T) *user.Store {
	t.Helper()
	database, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "hausbuch.db"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return user.NewStore(database)
}

// userCmd runs `hausbuch user args...` with stdin as standard input and returns its exit code and output.
func userCmd(t *testing.T, store *user.Store, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runUser(context.Background(), store, args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestUserAddWithPasswordFromStdin(t *testing.T) {
	store := newStore(t)
	if code, _, stderr := userCmd(t, store, "s3cret pass\n", "add", "anna", "Anna Rossi"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	u, err := store.Authenticate(context.Background(), "anna", "s3cret pass")
	if err != nil {
		t.Fatalf("the new user cannot sign in: %v", err)
	}
	if u.DisplayName != "Anna Rossi" {
		t.Errorf("display name = %q", u.DisplayName)
	}
}

func TestUserAddDuplicateUsername(t *testing.T) {
	store := newStore(t)
	userCmd(t, store, "pw\n", "add", "anna", "Anna")
	code, _, stderr := userCmd(t, store, "other\n", "add", "ANNA", "Another Anna")
	if code == 0 || !strings.Contains(stderr, "taken") {
		t.Fatalf("exit %d, stderr %q; want a failure saying the username is taken", code, stderr)
	}
	if users, _ := store.List(context.Background()); len(users) != 1 {
		t.Errorf("%d accounts after the refused add, want 1", len(users))
	}
}

func TestUserAddRejectsEmptyPassword(t *testing.T) {
	store := newStore(t)
	if code, _, _ := userCmd(t, store, "\n", "add", "anna", "Anna"); code == 0 {
		t.Fatal("an empty password was accepted")
	}
}

func TestUserList(t *testing.T) {
	store := newStore(t)
	now := time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local)
	store.Now = func() time.Time { return now }
	userCmd(t, store, "pw\n", "add", "anna", "Anna Rossi")
	userCmd(t, store, "pw\n", "add", "marco", "Marco")
	userCmd(t, store, "pw\n", "add", "bea", "Bea")
	// Disabling and locking have commands of their own; here only how they are listed matters.
	if _, err := store.DB.Exec("UPDATE app_user SET disabled = 1 WHERE username = 'marco'"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec("UPDATE app_user SET locked_until = ? WHERE username = 'bea'", now.Add(15*time.Minute).Unix()); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := userCmd(t, store, "", "list")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"Anna Rossi", "active", "marco", "disabled", "bea", "locked until 10:15"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list output lacks %q:\n%s", want, stdout)
		}
	}
}

func TestUserUsage(t *testing.T) {
	store := newStore(t)
	for _, args := range [][]string{{}, {"nope"}, {"add", "only-username"}, {"add", "anna", "Anna", "password-as-argument"}} {
		if code, _, stderr := userCmd(t, store, "", args...); code != 2 || !strings.Contains(stderr, "Usage") {
			t.Errorf("user %v: exit %d, stderr %q; want usage and exit 2", args, code, stderr)
		}
	}
}

func TestUserUnlock(t *testing.T) {
	store := newStore(t)
	userCmd(t, store, "pw\n", "add", "anna", "Anna")
	for range 3 {
		_, _ = store.Authenticate(context.Background(), "anna", "wrong")
	}
	if _, stdout, _ := userCmd(t, store, "", "list"); !strings.Contains(stdout, "locked until") {
		t.Fatalf("not listed as locked:\n%s", stdout)
	}
	if code, _, stderr := userCmd(t, store, "", "unlock", "anna"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if _, err := store.Authenticate(context.Background(), "anna", "pw"); err != nil {
		t.Fatalf("right password right after user unlock: %v", err)
	}
	if code, _, stderr := userCmd(t, store, "", "unlock", "nobody"); code != 1 || !strings.Contains(stderr, "no account") {
		t.Errorf("unlock of an unknown user: exit %d, %q", code, stderr)
	}
}
