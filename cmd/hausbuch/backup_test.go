package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/page"
	"github.com/davidepedone/hausbuch/internal/stock"
	"github.com/davidepedone/hausbuch/internal/user"
	"github.com/davidepedone/hausbuch/internal/web"
)

func TestServeRefusesInvalidBackupKeep(t *testing.T) {
	t.Setenv("HAUSBUCH_DB_PATH", filepath.Join(t.TempDir(), "hausbuch.db"))
	t.Setenv("HAUSBUCH_LISTEN", "127.0.0.1:0")
	for _, v := range []string{"0", "-1", "seven", "1.5"} {
		t.Setenv("HAUSBUCH_BACKUP_KEEP", v)
		if code := run([]string{"serve"}, nil, io.Discard, io.Discard); code != 1 {
			t.Errorf("HAUSBUCH_BACKUP_KEEP=%s: exit code %d, want 1", v, code)
		}
	}
}

// startServer runs the server as the image does, in the background, and waits until it answers. It keeps running
// until the tests end: the server has no way to stop.
func startServer(t *testing.T) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	t.Setenv("HAUSBUCH_LISTEN", addr)
	go run([]string{"serve"}, nil, io.Discard, io.Discard)
	waitFor(t, "the server to answer", func() bool { return healthcheck(addr) == nil })
}

func waitFor(t *testing.T, what string, done func() bool) {
	t.Helper()
	for range 100 {
		if done() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestServerKeepsDailyCopiesNextToTheDatabase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HAUSBUCH_DB_PATH", filepath.Join(dir, "hausbuch.db"))
	startServer(t)
	copyPath := filepath.Join(dir, "backups", "daily-"+time.Now().Format(time.DateOnly)+".db")
	waitFor(t, copyPath, func() bool {
		_, err := os.Stat(copyPath)
		return err == nil
	})
}

func TestServerRunsWhenTheDailyCopyFails(t *testing.T) {
	dir := t.TempDir()
	notADir := filepath.Join(dir, "file")
	if err := os.WriteFile(notADir, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HAUSBUCH_DB_PATH", filepath.Join(dir, "hausbuch.db"))
	t.Setenv("HAUSBUCH_BACKUP_DIR", filepath.Join(notADir, "backups"))
	startServer(t)
	// The copy is attempted right after the database opens, before the server listens; it still answers a moment
	// later.
	time.Sleep(200 * time.Millisecond)
	if err := healthcheck(os.Getenv("HAUSBUCH_LISTEN")); err != nil {
		t.Errorf("the server stopped after a failed daily copy: %v", err)
	}
}

// residentWorks checks that a resident can sign in, create, edit, list and search pages on the database at path.
func residentWorks(t *testing.T, path, password string, wantPages ...string) {
	t.Helper()
	database, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()
	pages := page.NewStore(database)
	if err := pages.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(web.NewHandler(user.NewStore(database), pages, stock.NewStore(database), "dev"))
	defer srv.Close()
	c := newClient(t)
	if status := signIn(t, c, srv, "anna", password); status != http.StatusSeeOther {
		t.Fatalf("sign in: %d", status)
	}

	created := post(t, c, srv, "/pages", url.Values{"title": {"Boiler"}, "type": {"blank"},
		"body": {"The manual is in the cellar."}})
	if created.StatusCode != http.StatusSeeOther {
		t.Fatalf("create: %s", created.Status)
	}
	path = created.Header.Get("Location")
	edited := post(t, c, srv, path, url.Values{"title": {"Boiler"}, "type": {"blank"},
		"body": {"The manual is in the drawer."}, "revision": {"1"}})
	if edited.StatusCode != http.StatusSeeOther {
		t.Fatalf("edit: %s", edited.Status)
	}
	list := get(t, c, srv, "/")
	for _, want := range append([]string{"Boiler"}, wantPages...) {
		if !strings.Contains(list, want) {
			t.Errorf("the page list doesn't show %q", want)
		}
	}
	if found := get(t, c, srv, "/search?q=drawer"); !strings.Contains(found, `href="`+path+`"`) {
		t.Errorf("searching the edited text doesn't find %s", path)
	}
}

func post(t *testing.T, c *http.Client, srv *httptest.Server, path string, form url.Values) *http.Response {
	t.Helper()
	resp, err := c.PostForm(srv.URL+path, form)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp
}

func get(t *testing.T, c *http.Client, srv *httptest.Server, path string) string {
	t.Helper()
	resp, err := c.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: %s", path, resp.Status)
	}
	return string(b)
}

// withData returns a database with an account, anna with password "pw", and a page.
func withData(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hausbuch.db")
	database, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	u, err := user.NewStore(database).Add(context.Background(), "anna", "Anna", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := page.NewStore(database).Create(context.Background(), u.ID,
		page.Content{Title: "Garden", Type: page.Blank, Body: "Water at dawn."}); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRestoringACopy(t *testing.T) {
	path := withData(t)
	database, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := db.Backup(context.Background(), database, dir, "copy.db"); err != nil {
		t.Fatal(err)
	}
	// Changes made after the copy are not in it.
	if _, err := database.Exec("UPDATE app_user SET password_hash = 'changed'"); err != nil {
		t.Fatal(err)
	}
	_ = database.Close()

	residentWorks(t, filepath.Join(dir, "copy.db"), "pw", "Garden")
}

// A release adding a table and columns the way migrations are allowed to: the current code must keep working on the
// database it leaves.
const futureMigration = `
CREATE TABLE reminder (id INTEGER PRIMARY KEY, page_id INTEGER NOT NULL REFERENCES page (id) ON DELETE CASCADE, due INTEGER NOT NULL);
CREATE INDEX reminder_due ON reminder (due);
ALTER TABLE page ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE app_user ADD COLUMN email TEXT;
INSERT INTO goose_db_version (version_id, is_applied) VALUES (99999, 1);
`

func TestRollingBack(t *testing.T) {
	path := withData(t)
	database, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(futureMigration); err != nil {
		t.Fatal(err)
	}
	_ = database.Close()

	residentWorks(t, path, "pw", "Garden")
}
