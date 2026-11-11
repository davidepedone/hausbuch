package main

import (
	"bytes"
	"context"
	"path/filepath"
	"regexp"
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
	code := runUser(context.Background(), store, "", args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestUserAddWithPasswordFromStdin(t *testing.T) {
	store := newStore(t)
	if code, _, stderr := userCmd(t, store, "s3cret passphrase here\n", "add", "anna", "Anna Rossi"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	u, err := store.Authenticate(context.Background(), "anna", "s3cret passphrase here", "192.0.2.1")
	if err != nil {
		t.Fatalf("the new user cannot sign in: %v", err)
	}
	if u.DisplayName != "Anna Rossi" {
		t.Errorf("display name = %q", u.DisplayName)
	}
}

func TestUserAddDuplicateUsername(t *testing.T) {
	store := newStore(t)
	userCmd(t, store, "spoon kettle garden\n", "add", "anna", "Anna")
	code, _, stderr := userCmd(t, store, "another kettle garden\n", "add", "ANNA", "Another Anna")
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
	userCmd(t, store, "spoon kettle garden\n", "add", "anna", "Anna Rossi")
	userCmd(t, store, "spoon kettle garden\n", "add", "marco", "Marco")
	userCmd(t, store, "spoon kettle garden\n", "add", "bea", "Bea")
	// Disabling and locking have commands of their own; here only how they are listed matters.
	if _, err := store.DB.Exec("UPDATE app_user SET disabled = 1 WHERE username = 'marco'"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec("INSERT INTO sign_in_failure VALUES ('bea', '192.0.2.1', 3, ?, ?)",
		now.Add(15*time.Minute).Unix(), now.Unix()); err != nil {
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
	userCmd(t, store, "spoon kettle garden\n", "add", "anna", "Anna")
	for range 3 {
		_, _ = store.Authenticate(context.Background(), "anna", "wrong", "192.0.2.1")
	}
	if _, stdout, _ := userCmd(t, store, "", "list"); !strings.Contains(stdout, "locked until") {
		t.Fatalf("not listed as locked:\n%s", stdout)
	}
	if code, _, stderr := userCmd(t, store, "", "unlock", "anna"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if _, err := store.Authenticate(context.Background(), "anna", "spoon kettle garden", "192.0.2.1"); err != nil {
		t.Fatalf("right password right after user unlock: %v", err)
	}
	if code, _, stderr := userCmd(t, store, "", "unlock", "nobody"); code != 1 || !strings.Contains(stderr, "no account") {
		t.Errorf("unlock of an unknown user: exit %d, %q", code, stderr)
	}
}

func TestUserLink(t *testing.T) {
	store := newStore(t)
	userCmd(t, store, "spoon kettle garden\n", "add", "anna", "Anna")
	code, stdout, stderr := userCmd(t, store, "", "link", "anna")
	path := regexp.MustCompile(`\n(/link/[A-Za-z0-9_-]{43})\n`).FindStringSubmatch(stdout)
	if code != 0 || path == nil || !strings.Contains(stdout, "set HAUSBUCH_URL") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if _, err := store.UseLink(context.Background(), strings.TrimPrefix(path[1], "/link/")); err != nil {
		t.Errorf("printed link doesn't sign in: %v", err)
	}

	var out bytes.Buffer
	if code := runUser(context.Background(), store, "https://home.example.org/", []string{"link", "anna"}, strings.NewReader(""), &out, &out); code != 0 ||
		!regexp.MustCompile(`\nhttps://home\.example\.org/link/[A-Za-z0-9_-]{43}\n$`).MatchString(out.String()) {
		t.Errorf("with HAUSBUCH_URL: exit %d, %q", code, out.String())
	}

	userCmd(t, store, "", "disable", "anna")
	if code, _, stderr := userCmd(t, store, "", "link", "anna"); code == 0 || !strings.Contains(stderr, "disabled") {
		t.Errorf("disabled account: exit %d, %q", code, stderr)
	}
}

func TestUserAddRejectsAShortPassword(t *testing.T) {
	store := newStore(t)
	code, _, stderr := userCmd(t, store, strings.Repeat("a", 14)+"\n", "add", "anna", "Anna")
	if code != 1 || !strings.Contains(stderr, "at least 15 characters") {
		t.Fatalf("exit %d, stderr %q; want a failure saying the password must be at least 15 characters", code, stderr)
	}
	if users, _ := store.List(context.Background()); len(users) != 0 {
		t.Errorf("%d accounts after the refused add, want 0", len(users))
	}
}

func TestUserSetPasswordRejectsACommonPassword(t *testing.T) {
	store := newStore(t)
	userCmd(t, store, "spoon kettle garden\n", "add", "anna", "Anna")
	code, _, stderr := userCmd(t, store, "qwertyuiopasdfgh\n", "set-password", "anna")
	if code != 1 || !strings.Contains(stderr, "too common") {
		t.Fatalf("exit %d, stderr %q; want a failure saying the password is too common", code, stderr)
	}
	if _, err := store.Authenticate(context.Background(), "anna", "spoon kettle garden", "192.0.2.1"); err != nil {
		t.Errorf("the refused password replaced the old one: %v", err)
	}
}
