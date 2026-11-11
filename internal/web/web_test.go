package web

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/page"
	"github.com/davidepedone/hausbuch/internal/stock"
	"github.com/davidepedone/hausbuch/internal/user"
)

type testApp struct {
	t     *testing.T
	srv   *httptest.Server
	users *user.Store
	pages *page.Store
	stock *stock.Store
}

func newApp(t *testing.T) *testApp {
	return newAppAt(t, "dev")
}

// newAppAt starts the app running as version.
func newAppAt(t *testing.T, version string) *testApp {
	return startApp(t, version, false)
}

// newAppWithPasskeys starts the app with passkeys on, reached as localhost, where browsers allow them over HTTP.
func newAppWithPasskeys(t *testing.T) *testApp {
	return startApp(t, "dev", true)
}

func startApp(t *testing.T, version string, passkeys bool) *testApp {
	t.Helper()
	database, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "hausbuch.db"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	users, pages, items := user.NewStore(database), page.NewStore(database), stock.NewStore(database)
	srv := httptest.NewUnstartedServer(nil)
	var config *webauthn.WebAuthn
	if passkeys {
		l, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		srv.Listener.Close()
		srv.Listener = l
		if config, err = PasskeyConfig("http://localhost:" + strconv.Itoa(l.Addr().(*net.TCPAddr).Port)); err != nil {
			t.Fatal(err)
		}
	}
	// The test server is its own trusted proxy, so that a browser can come from any address (see browser.from).
	proxies, err := ParseTrustedProxies("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	srv.Config.Handler = NewHandler(users, pages, items, version, config, proxies)
	srv.Start()
	if passkeys {
		srv.URL = strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)
	}
	t.Cleanup(func() {
		srv.Close()
		_ = database.Close()
	})
	return &testApp{t: t, srv: srv, users: users, pages: pages, stock: items}
}

func (a *testApp) addUser(username, displayName, password string) user.User {
	a.t.Helper()
	u, err := a.users.Add(context.Background(), username, displayName, password)
	if err != nil {
		a.t.Fatal(err)
	}
	return u
}

// browser is a client keeping cookies, like a browser, but not following redirects, so tests can check them. Its
// requests come from the address from when set, as reported by a proxy.
type browser struct {
	app    *testApp
	client *http.Client
	from   string
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
	if b.from != "" {
		req.Header.Set("X-Forwarded-For", b.from)
	}
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
	app.addUser("anna", "Anna", "spoon kettle garden")
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
	app.addUser("anna", "Anna Rossi", "s3cret passphrase")
	b := app.browser()

	r := b.post("/login", url.Values{"username": {"ANNA"}, "password": {"s3cret passphrase"}, "next": {"/some/page?x=1"}})
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
	app.addUser("anna", "Anna", "spoon kettle garden")
	if r := app.browser().signIn("anna", "spoon kettle garden"); r.Header.Get("Location") != "/" {
		t.Errorf("redirect to %q, want /", r.Header.Get("Location"))
	}
}

func TestSignInDoesNotRedirectToAnotherSite(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	for _, next := range []string{"https://evil.example/", "//evil.example/", `/\evil.example/`, "javascript:alert(1)", "relative"} {
		r := app.browser().post("/login", url.Values{"username": {"anna"}, "password": {"spoon kettle garden"}, "next": {next}})
		if r.Header.Get("Location") != "/" {
			t.Errorf("next=%q redirected to %q, want /", next, r.Header.Get("Location"))
		}
	}
}

func TestSessionCookieFlags(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	r := app.browser().signIn("anna", "spoon kettle garden")
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
		strings.NewReader(url.Values{"username": {"anna"}, "password": {"spoon kettle garden"}}.Encode()))
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
	app.addUser("anna", "Anna", "spoon kettle garden")
	unknown := app.browser().signIn("nobody", "spoon kettle garden")
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
	app.addUser("anna", "Anna", "spoon kettle garden")
	if _, err := app.users.DB.Exec("UPDATE app_user SET disabled = 1"); err != nil {
		t.Fatal(err)
	}
	if r := app.browser().signIn("anna", "spoon kettle garden"); r.StatusCode != http.StatusUnauthorized {
		t.Errorf("disabled account: %d, want %d", r.StatusCode, http.StatusUnauthorized)
	}
}

func TestLoginPageWithoutAccounts(t *testing.T) {
	app := newApp(t)
	r := app.browser().get("/login")
	if !strings.Contains(r.body, "There are no accounts yet") || !strings.Contains(r.body, "hausbuch user add") {
		t.Error("login page of a fresh installation does not explain how to create the first account")
	}
	app.addUser("anna", "Anna", "spoon kettle garden")
	if r := app.browser().get("/login"); strings.Contains(r.body, "There are no accounts yet") {
		t.Error("login page still says there are no accounts after one was added")
	}
}

func TestSignOut(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	b := app.browser()
	b.signIn("anna", "spoon kettle garden")
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
	app.addUser("anna", "Anna", "spoon kettle garden")
	req, _ := http.NewRequest(http.MethodPost, app.srv.URL+"/login",
		strings.NewReader(url.Values{"username": {"anna"}, "password": {"spoon kettle garden"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	if r := app.browser().do(req); r.StatusCode != http.StatusForbidden {
		t.Errorf("cross-site POST /login: %d, want %d", r.StatusCode, http.StatusForbidden)
	}
}

func TestChangePassword(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "old spoon kettle garden")
	b := app.browser()
	b.signIn("anna", "old spoon kettle garden")

	r := b.post("/account/password", url.Values{"current": {"old spoon kettle garden"}, "new": {"new spoon kettle garden"}, "repeat": {"new spoon kettle garden"}})
	if r.StatusCode != http.StatusOK || !strings.Contains(r.body, "Password changed.") {
		t.Fatalf("change: %d, confirmation shown: %v", r.StatusCode, strings.Contains(r.body, "Password changed."))
	}
	if r := app.browser().signIn("anna", "new spoon kettle garden"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("sign in with the new password: %d", r.StatusCode)
	}
	if r := app.browser().signIn("anna", "old spoon kettle garden"); r.StatusCode != http.StatusUnauthorized {
		t.Errorf("sign in with the old password: %d, want %d", r.StatusCode, http.StatusUnauthorized)
	}
}

func TestChangePasswordRefused(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	b := app.browser()
	b.signIn("anna", "spoon kettle garden")

	for name, form := range map[string]url.Values{
		"wrong current password": {"current": {"nope"}, "new": {"new spoon kettle garden"}, "repeat": {"new spoon kettle garden"}},
		"new passwords differ":   {"current": {"spoon kettle garden"}, "new": {"new spoon kettle garden"}, "repeat": {"new spoon kettle gardem"}},
		"empty new password":     {"current": {"spoon kettle garden"}, "new": {""}, "repeat": {""}},
	} {
		if r := b.post("/account/password", form); r.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: %d, want %d", name, r.StatusCode, http.StatusBadRequest)
		}
	}
	if r := app.browser().signIn("anna", "spoon kettle garden"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("the password changed after refused attempts: sign in with it gives %d", r.StatusCode)
	}
}

// The sign-in page shows the icons too, so everything it refers to under /static/, the manifest's icons included,
// must be reachable without a session.
func TestLoginPageAssetsNeedNoSession(t *testing.T) {
	b := newApp(t).browser()
	page := b.get("/login").body + b.get("/static/manifest.json").body
	paths := regexp.MustCompile(`/static/[\w.-]+`).FindAllString(page, -1)
	for _, want := range []string{"/static/favicon.svg", "/static/apple-touch-icon.png", "/static/manifest.json",
		"/static/icon-192.png", "/static/icon-512.png"} {
		if !slices.Contains(paths, want) {
			t.Errorf("%s not referred to", want)
		}
	}
	for _, p := range paths {
		if r := b.get(p); r.StatusCode != http.StatusOK {
			t.Errorf("%s: %s", p, r.Status)
		}
	}
}

// Browsers that colour their own bars take the theme colour, which must match the header's background.
func TestHomeScreenAppTakesTheHeaderColour(t *testing.T) {
	b := newApp(t).browser()
	if page := b.get("/login").body; !strings.Contains(page, `<meta name="theme-color" content="#dde8d8">`) {
		t.Error("the pages declare no theme colour matching the header")
	}
	manifest := b.get("/static/manifest.json").body
	for _, want := range []string{`"display": "standalone"`, `"theme_color": "#dde8d8"`} {
		if !strings.Contains(manifest, want) {
			t.Errorf("manifest lacks %s", want)
		}
	}
}

// iOS 26 blurs the top of a page scrolling under its status bar, header included, unless an opaque element is stuck
// to the top: the header must stay there.
func TestHeaderStaysAtTheTop(t *testing.T) {
	page := newApp(t).browser().get("/login").body
	if !regexp.MustCompile(`<header class="[^"]*\bsticky top-0\b[^"]*\bbg-sage-100\b`).MatchString(page) {
		t.Error("the header is not stuck to the top with an opaque background")
	}
}

func TestSignInLockIsPerAddress(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	stranger, anna := app.browser(), app.browser()
	stranger.from, anna.from = "198.51.100.7", "192.0.2.1"
	for range 3 {
		stranger.signIn("anna", "wrong")
	}
	if r := stranger.signIn("anna", "spoon kettle garden"); r.StatusCode != http.StatusUnauthorized {
		t.Errorf("right password from the locked address: %d, want %d", r.StatusCode, http.StatusUnauthorized)
	}
	if r := anna.signIn("anna", "spoon kettle garden"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("right password from another address: %d, want %d", r.StatusCode, http.StatusSeeOther)
	}
}
