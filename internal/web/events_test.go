package web

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// logged is the server's log while a test runs.
type logged struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logged) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *logged) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func captureLog(t *testing.T) *logged {
	t.Helper()
	l := &logged{}
	saved := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(l, nil)))
	t.Cleanup(func() { slog.SetDefault(saved) })
	return l
}

// line is the log's line with message msg, or "" when there is none.
func (l *logged) line(msg string) string {
	for line := range strings.Lines(l.String()) {
		if strings.Contains(line, `msg="`+msg+`"`) || strings.Contains(line, "msg="+msg+" ") {
			return line
		}
	}
	return ""
}

func mustLog(t *testing.T, l *logged, msg string, wants ...string) {
	t.Helper()
	line := l.line(msg)
	if line == "" {
		t.Fatalf("no %q line in the log:\n%s", msg, l)
	}
	for _, want := range wants {
		if !strings.Contains(line, want) {
			t.Errorf("%q line lacks %q:\n%s", msg, want, line)
		}
	}
}

func mustNotLog(t *testing.T, l *logged, secrets ...string) {
	t.Helper()
	for _, s := range secrets {
		if strings.Contains(l.String(), s) {
			t.Errorf("the log holds %q:\n%s", s, l)
		}
	}
}

func TestFailedSignInLogged(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	l := captureLog(t)
	b := app.browser()
	b.from = "198.51.100.7"
	b.signIn("anna", "wrong kettle garden")
	mustLog(t, l, "sign-in failed", "method=password", `reason="wrong password"`, "user=anna", "address=198.51.100.7")
	mustNotLog(t, l, "wrong kettle garden")

	for range 2 {
		b.signIn("anna", "wrong kettle garden")
	}
	mustLog(t, l, "sign-in lock started", "user=anna", "address=198.51.100.7")
}

func TestUnknownUsernameLogged(t *testing.T) {
	app := newApp(t)
	l := captureLog(t)
	b := app.browser()
	b.from = "198.51.100.7"
	b.signIn("spoon kettle garden", "anna") // username and password swapped
	mustLog(t, l, "sign-in failed", `reason="unknown user"`, "user=unknown", "address=198.51.100.7")
	mustNotLog(t, l, "spoon kettle garden")
}

func TestSignInLogged(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	l := captureLog(t)
	app.browser().signIn("anna", "spoon kettle garden")
	mustLog(t, l, "signed in", "method=password", "user=anna", "address=127.0.0.1")
	mustNotLog(t, l, "spoon kettle garden")
}

func TestLinkUsedLogged(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	path := app.link("anna")
	l := captureLog(t)
	b := app.browser()
	b.from = "198.51.100.7"
	b.post(path, nil)
	mustLog(t, l, "signed in", "method=link", "user=anna", "address=198.51.100.7")
	b.get(path) // used up
	mustLog(t, l, "sign-in link not valid", "address=198.51.100.7")
	mustNotLog(t, l, strings.TrimPrefix(path, "/link/"))
}

func TestRequestLimitLoggedOnce(t *testing.T) {
	app := newApp(t)
	limitedApp(app)
	l := captureLog(t)
	b := app.browser()
	b.from = "198.51.100.7"
	fill(t, b, requestLimit)
	for range 3 {
		b.get("/link/a-token-to-keep-out-of-the-log")
	}
	mustLog(t, l, "request limit hit", "address=198.51.100.7", `route="GET /link/{token}"`)
	if n := strings.Count(l.String(), "request limit hit"); n != 1 {
		t.Errorf("%d lines for 3 refused requests, want 1", n)
	}
	mustNotLog(t, l, "a-token-to-keep-out-of-the-log")
}

func TestAccountChangesLogged(t *testing.T) {
	app := newApp(t)
	b := app.signedInAs("anna", "Anna")
	l := captureLog(t)
	changeTo(b, "lemon chair river clouds")
	mustLog(t, l, "password changed", "user=anna")
	mustNotLog(t, l, "lemon chair river clouds", "spoon kettle garden")
}
