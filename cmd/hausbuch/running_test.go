package main

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/page"
	"github.com/davidepedone/hausbuch/internal/stock"
	"github.com/davidepedone/hausbuch/internal/user"
	"github.com/davidepedone/hausbuch/internal/web"
)

// runningServer starts the web server on a database file and returns a store opened separately on the same file,
// the way `docker exec … hausbuch user …` reaches the database of a running server.
func runningServer(t *testing.T) (*httptest.Server, *user.Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hausbuch.db")
	serverDB, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cliDB, err := db.Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(web.NewHandler(user.NewStore(serverDB), page.NewStore(serverDB), stock.NewStore(serverDB), "en"))
	t.Cleanup(func() {
		srv.Close()
		_ = serverDB.Close()
		_ = cliDB.Close()
	})
	return srv, user.NewStore(cliDB)
}

func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func signIn(t *testing.T, c *http.Client, srv *httptest.Server, username, password string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/login",
		strings.NewReader(url.Values{"username": {username}, "password": {password}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func home(t *testing.T, c *http.Client, srv *httptest.Server) int {
	t.Helper()
	resp, err := c.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestSetPasswordWhileServerRuns(t *testing.T) {
	srv, cli := runningServer(t)
	userCmd(t, cli, "old pw\n", "add", "anna", "Anna")

	if code, _, stderr := userCmd(t, cli, "new pw\n", "set-password", "anna"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if status := signIn(t, newClient(t), srv, "anna", "new pw"); status != http.StatusSeeOther {
		t.Errorf("sign in with the new password: %d", status)
	}
	if status := signIn(t, newClient(t), srv, "anna", "old pw"); status != http.StatusUnauthorized {
		t.Errorf("sign in with the old password: %d, want %d", status, http.StatusUnauthorized)
	}
}

func TestDisableAndEnableWhileServerRuns(t *testing.T) {
	srv, cli := runningServer(t)
	userCmd(t, cli, "pw\n", "add", "anna", "Anna")
	browser := newClient(t)
	signIn(t, browser, srv, "anna", "pw")
	if status := home(t, browser, srv); status != http.StatusOK {
		t.Fatalf("home while signed in: %d", status)
	}

	if code, _, stderr := userCmd(t, cli, "", "disable", "anna"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if status := home(t, browser, srv); status != http.StatusSeeOther {
		t.Errorf("home after disable: %d, want a redirect to the login page", status)
	}
	if status := signIn(t, newClient(t), srv, "anna", "pw"); status != http.StatusUnauthorized {
		t.Errorf("sign in of a disabled account: %d, want %d", status, http.StatusUnauthorized)
	}
	if _, stdout, _ := userCmd(t, cli, "", "list"); !strings.Contains(stdout, "disabled") {
		t.Errorf("not listed as disabled:\n%s", stdout)
	}

	// Re-enabling doesn't bring the ended sessions back, it allows signing in again.
	if code, _, stderr := userCmd(t, cli, "", "enable", "anna"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if status := home(t, browser, srv); status != http.StatusSeeOther {
		t.Errorf("old session after enable: %d, want it still ended", status)
	}
	if status := signIn(t, newClient(t), srv, "anna", "pw"); status != http.StatusSeeOther {
		t.Errorf("sign in after enable: %d", status)
	}
}

func TestAccountCommandsOnUnknownUser(t *testing.T) {
	_, cli := runningServer(t)
	for _, args := range [][]string{{"set-password", "nobody"}, {"disable", "nobody"}, {"enable", "nobody"}} {
		if code, _, stderr := userCmd(t, cli, "pw\n", args...); code != 1 || !strings.Contains(stderr, "no account") {
			t.Errorf("user %v: exit %d, %q", args, code, stderr)
		}
	}
}
