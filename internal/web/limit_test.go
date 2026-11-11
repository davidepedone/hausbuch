package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// limitedApp stops the app's clock at the start of the current minute, which the returned function moves. It is the
// real time's, for the cookie jar to keep session cookies.
func limitedApp(a *testApp) func(time.Duration) {
	now := time.Now().Truncate(time.Minute)
	a.users.Now = func() time.Time { return now }
	return func(d time.Duration) { now = now.Add(d) }
}

func fill(t *testing.T, b *browser, n int) {
	t.Helper()
	for i := range n {
		if r := b.get("/login"); r.StatusCode != http.StatusOK {
			t.Fatalf("request %d: %d, want %d", i+1, r.StatusCode, http.StatusOK)
		}
	}
}

func TestTooManyAttempts(t *testing.T) {
	app := newApp(t)
	limitedApp(app)
	app.addUser("anna", "Anna", "spoon kettle garden")
	b := app.browser()
	b.from = "198.51.100.7"
	fill(t, b, requestLimit)
	r := b.signIn("anna", "spoon kettle garden")
	if r.StatusCode != http.StatusTooManyRequests || !strings.Contains(r.body, "Try again in a minute") {
		t.Fatalf("sign-in over the limit: %d, want %d with the message", r.StatusCode, http.StatusTooManyRequests)
	}
	if b.sessionCookie() != nil {
		t.Error("signed in over the limit")
	}
}

func TestRefusedAttemptsCheckNoPassword(t *testing.T) {
	app := newApp(t)
	advance := limitedApp(app)
	app.addUser("anna", "Anna", "spoon kettle garden")
	b := app.browser()
	b.from = "198.51.100.7"
	fill(t, b, requestLimit)
	for range 3 {
		b.signIn("anna", "wrong") // would lock the username from this address if checked
	}
	advance(time.Minute)
	if r := b.signIn("anna", "spoon kettle garden"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("right password after 3 refused wrong ones: %d, want %d", r.StatusCode, http.StatusSeeOther)
	}
}

func TestLimitOver(t *testing.T) {
	app := newApp(t)
	advance := limitedApp(app)
	b := app.browser()
	b.from = "198.51.100.7"
	fill(t, b, requestLimit)
	if r := b.get("/login"); r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("over the limit: %d", r.StatusCode)
	}
	advance(time.Minute)
	fill(t, b, 1)
}

func TestLimitIsPerAddress(t *testing.T) {
	app := newApp(t)
	limitedApp(app)
	b, other := app.browser(), app.browser()
	b.from, other.from = "198.51.100.7", "192.0.2.1"
	fill(t, b, requestLimit)
	fill(t, other, 1)
}

func TestSameIPv6BlockSharesTheLimit(t *testing.T) {
	app := newApp(t)
	limitedApp(app)
	b, neighbour := app.browser(), app.browser()
	b.from, neighbour.from = "2001:db8:1:2::1", "2001:db8:1:2:ffff::9"
	fill(t, b, requestLimit/2)
	fill(t, neighbour, requestLimit/2)
	if r := b.get("/login"); r.StatusCode != http.StatusTooManyRequests {
		t.Errorf("a request over the limit of the shared /64: %d, want %d", r.StatusCode, http.StatusTooManyRequests)
	}
}

func TestSignedInPagesAreNotLimited(t *testing.T) {
	app := newApp(t)
	limitedApp(app)
	app.addUser("anna", "Anna", "spoon kettle garden")
	b := app.browser()
	b.from = "198.51.100.7"
	b.signIn("anna", "spoon kettle garden")
	for i := range 2 * requestLimit {
		if r := b.get("/"); r.StatusCode != http.StatusOK {
			t.Fatalf("home page, request %d: %d", i+1, r.StatusCode)
		}
	}
	if r := b.get("/static/app.css"); r.StatusCode == http.StatusTooManyRequests {
		t.Error("a static asset was refused")
	}
}

func TestPasskeySignInOverTheLimit(t *testing.T) {
	app := newAppWithPasskeys(t)
	limitedApp(app)
	b := app.browser()
	b.from = "198.51.100.7"
	fill(t, b, requestLimit)
	r := b.post("/login/passkey/begin", nil)
	var answer struct{ Error string }
	if err := json.Unmarshal([]byte(r.body), &answer); err != nil || r.StatusCode != http.StatusTooManyRequests ||
		!strings.Contains(answer.Error, "Try again in a minute") {
		t.Errorf("passkey sign-in over the limit: %d %q, want %d with the message in JSON", r.StatusCode, r.body,
			http.StatusTooManyRequests)
	}
}
