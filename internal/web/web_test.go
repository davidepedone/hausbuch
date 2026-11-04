package web

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/user"
)

type testApp struct {
	t     *testing.T
	srv   *httptest.Server
	users *user.Store
}

func newApp(t *testing.T) *testApp {
	t.Helper()
	database, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "hausbuch.db"))
	if err != nil {
		t.Fatal(err)
	}
	users := user.NewStore(database)
	srv := httptest.NewServer(NewHandler(users))
	t.Cleanup(func() {
		srv.Close()
		_ = database.Close()
	})
	return &testApp{t: t, srv: srv, users: users}
}

func (a *testApp) addUser(username, displayName, password string) user.User {
	a.t.Helper()
	u, err := a.users.Add(context.Background(), username, displayName, password)
	if err != nil {
		a.t.Fatal(err)
	}
	return u
}

// browser is a client keeping cookies, like a browser, but not following redirects, so tests can check them.
type browser struct {
	app    *testApp
	client *http.Client
}

func (a *testApp) browser() *browser {
	jar, err := cookiejar.New(nil)
	if err != nil {
		a.t.Fatal(err)
	}
	return &browser{app: a, client: &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

type response struct {
	*http.Response
	body string
}

func (b *browser) do(req *http.Request) response {
	b.app.t.Helper()
	resp, err := b.client.Do(req)
	if err != nil {
		b.app.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.app.t.Fatal(err)
	}
	return response{resp, string(body)}
}

func (b *browser) get(path string) response {
	b.app.t.Helper()
	req, err := http.NewRequest(http.MethodGet, b.app.srv.URL+path, nil)
	if err != nil {
		b.app.t.Fatal(err)
	}
	return b.do(req)
}

// post sends a form the way a browser submits one from a page of the same site.
func (b *browser) post(path string, form url.Values) response {
	b.app.t.Helper()
	req, err := http.NewRequest(http.MethodPost, b.app.srv.URL+path, strings.NewReader(form.Encode()))
	if err != nil {
		b.app.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	return b.do(req)
}

func (b *browser) signIn(username, password string) response {
	b.app.t.Helper()
	return b.post("/login", url.Values{"username": {username}, "password": {password}})
}

func (b *browser) sessionCookie() *http.Cookie {
	u, _ := url.Parse(b.app.srv.URL)
	for _, c := range b.client.Jar.Cookies(u) {
		if c.Name == sessionCookie {
			return c
		}
	}
	return nil
}

func TestHealthzAndStaticNeedNoSession(t *testing.T) {
	b := newApp(t).browser()
	if r := b.get("/healthz"); r.StatusCode != http.StatusOK {
		t.Errorf("/healthz: %d", r.StatusCode)
	}
	if r := b.get("/static/htmx.min.js"); r.StatusCode != http.StatusOK || !strings.Contains(r.body, "htmx") {
		t.Errorf("/static/htmx.min.js: %d", r.StatusCode)
	}
}

func TestAnonymousVisitorIsSentToLogin(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	b := app.browser()
	for _, path := range []string{"/", "/some/page?x=1", "/does-not-exist"} {
		r := b.get(path)
		if r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/login?next="+url.QueryEscape(path) {
			t.Errorf("GET %s: %d to %q, want a redirect to the login page", path, r.StatusCode, r.Header.Get("Location"))
		}
		if strings.Contains(r.body, "<main") {
			t.Errorf("GET %s showed a page to an anonymous visitor", path)
		}
	}
}

func TestAnonymousRequestToAContentEndpointIsRefused(t *testing.T) {
	b := newApp(t).browser()
	for _, path := range []string{"/", "/logout", "/pages"} {
		if r := b.post(path, url.Values{"body": {"x"}}); r.StatusCode != http.StatusUnauthorized {
			t.Errorf("anonymous POST %s: %d, want %d", path, r.StatusCode, http.StatusUnauthorized)
		}
	}
}

func TestSignInWithCorrectCredentials(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna Rossi", "s3cret")
	b := app.browser()

	r := b.post("/login", url.Values{"username": {"ANNA"}, "password": {"s3cret"}, "next": {"/some/page?x=1"}})
	if r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/some/page?x=1" {
		t.Fatalf("sign in: %d to %q, want a redirect to the page first asked for", r.StatusCode, r.Header.Get("Location"))
	}
	home := b.get("/")
	if home.StatusCode != http.StatusOK || !strings.Contains(home.body, "Anna Rossi") {
		t.Errorf("home after signing in: %d, display name shown: %v", home.StatusCode, strings.Contains(home.body, "Anna Rossi"))
	}
}

func TestSignInWithoutNextGoesHome(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	if r := app.browser().signIn("anna", "pw"); r.Header.Get("Location") != "/" {
		t.Errorf("redirect to %q, want /", r.Header.Get("Location"))
	}
}

func TestSignInDoesNotRedirectToAnotherSite(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	for _, next := range []string{"https://evil.example/", "//evil.example/", `/\evil.example/`, "javascript:alert(1)", "relative"} {
		r := app.browser().post("/login", url.Values{"username": {"anna"}, "password": {"pw"}, "next": {next}})
		if r.Header.Get("Location") != "/" {
			t.Errorf("next=%q redirected to %q, want /", next, r.Header.Get("Location"))
		}
	}
}

func TestSessionCookieFlags(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	r := app.browser().signIn("anna", "pw")
	var c *http.Cookie
	for _, rc := range r.Cookies() {
		if rc.Name == sessionCookie {
			c = rc
		}
	}
	if c == nil {
		t.Fatal("no session cookie set")
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Secure || c.Path != "/" || c.Expires.IsZero() {
		t.Errorf("cookie %+v, want HttpOnly, SameSite=Lax, not Secure over HTTP, Path=/, with an expiry", c)
	}

	req, _ := http.NewRequest(http.MethodPost, app.srv.URL+"/login",
		strings.NewReader(url.Values{"username": {"anna"}, "password": {"pw"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-Proto", "https")
	resp := app.browser().do(req)
	for _, rc := range resp.Cookies() {
		if rc.Name == sessionCookie && !rc.Secure {
			t.Error("cookie set behind an HTTPS proxy is not Secure")
		}
	}
}

func TestWrongCredentialsLookTheSame(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	unknown := app.browser().signIn("nobody", "pw")
	wrong := app.browser().signIn("anna", "wrong")
	for name, r := range map[string]response{"unknown username": unknown, "wrong password": wrong} {
		if r.StatusCode != http.StatusUnauthorized || !strings.Contains(r.body, "Invalid username or password.") {
			t.Errorf("%s: %d, message shown: %v", name, r.StatusCode, strings.Contains(r.body, "Invalid username or password."))
		}
		for _, c := range r.Cookies() {
			if c.Name == sessionCookie {
				t.Errorf("%s: a session cookie was set", name)
			}
		}
	}
	strip := func(s string) string { return strings.ReplaceAll(s, `value="nobody"`, `value="anna"`) }
	if strip(unknown.body) != wrong.body {
		t.Error("the page for an unknown username differs from the one for a wrong password")
	}
}

func TestDisabledAccountCannotSignIn(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	if _, err := app.users.DB.Exec("UPDATE app_user SET disabled = 1"); err != nil {
		t.Fatal(err)
	}
	if r := app.browser().signIn("anna", "pw"); r.StatusCode != http.StatusUnauthorized {
		t.Errorf("disabled account: %d, want %d", r.StatusCode, http.StatusUnauthorized)
	}
}

func TestLoginPageWithoutAccounts(t *testing.T) {
	app := newApp(t)
	r := app.browser().get("/login")
	if !strings.Contains(r.body, "There are no accounts yet") || !strings.Contains(r.body, "hausbuch user add") {
		t.Error("login page of a fresh installation does not explain how to create the first account")
	}
	app.addUser("anna", "Anna", "pw")
	if r := app.browser().get("/login"); strings.Contains(r.body, "There are no accounts yet") {
		t.Error("login page still says there are no accounts after one was added")
	}
}

func TestSignOut(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	b := app.browser()
	b.signIn("anna", "pw")
	token := b.sessionCookie().Value

	if r := b.post("/logout", nil); r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/login" {
		t.Fatalf("sign out: %d to %q", r.StatusCode, r.Header.Get("Location"))
	}
	if r := b.get("/"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("after signing out, GET /: %d, want a redirect to the login page", r.StatusCode)
	}
	if _, err := app.users.LookupSession(context.Background(), token); err == nil {
		t.Error("the session still exists after signing out")
	}
}

func TestCrossOriginPostIsRefused(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	req, _ := http.NewRequest(http.MethodPost, app.srv.URL+"/login",
		strings.NewReader(url.Values{"username": {"anna"}, "password": {"pw"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	if r := app.browser().do(req); r.StatusCode != http.StatusForbidden {
		t.Errorf("cross-site POST /login: %d, want %d", r.StatusCode, http.StatusForbidden)
	}
}

func TestChangePassword(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "old pw")
	b := app.browser()
	b.signIn("anna", "old pw")

	r := b.post("/account/password", url.Values{"current": {"old pw"}, "new": {"new pw"}, "repeat": {"new pw"}})
	if r.StatusCode != http.StatusOK || !strings.Contains(r.body, "Password changed.") {
		t.Fatalf("change: %d, confirmation shown: %v", r.StatusCode, strings.Contains(r.body, "Password changed."))
	}
	if r := app.browser().signIn("anna", "new pw"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("sign in with the new password: %d", r.StatusCode)
	}
	if r := app.browser().signIn("anna", "old pw"); r.StatusCode != http.StatusUnauthorized {
		t.Errorf("sign in with the old password: %d, want %d", r.StatusCode, http.StatusUnauthorized)
	}
}

func TestChangePasswordRefused(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "pw")
	b := app.browser()
	b.signIn("anna", "pw")

	for name, form := range map[string]url.Values{
		"wrong current password": {"current": {"nope"}, "new": {"new pw"}, "repeat": {"new pw"}},
		"new passwords differ":   {"current": {"pw"}, "new": {"new pw"}, "repeat": {"new pv"}},
		"empty new password":     {"current": {"pw"}, "new": {""}, "repeat": {""}},
	} {
		if r := b.post("/account/password", form); r.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: %d, want %d", name, r.StatusCode, http.StatusBadRequest)
		}
	}
	if r := app.browser().signIn("anna", "pw"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("the password changed after refused attempts: sign in with it gives %d", r.StatusCode)
	}
}
