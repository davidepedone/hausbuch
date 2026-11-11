package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/user"
)

func changeTo(b *browser, password string) response {
	return b.post("/account/password",
		url.Values{"current": {"spoon kettle garden"}, "new": {password}, "repeat": {password}})
}

func TestPasswordRulesShown(t *testing.T) {
	b := newApp(t).signedInAs("anna", "Anna")
	mustContain(t, b.view("/account/password"), "At least 15 characters", `minlength="15"`, `maxlength="256"`)
}

func TestNewPasswordRefusals(t *testing.T) {
	app := newApp(t)
	b := app.signedInAs("anna.rossi.bianchi", "Anna")
	for password, want := range map[string]string{
		strings.Repeat("a", 14):  "must be at least 15 characters",
		strings.Repeat("a", 257): "can be at most 256 characters",
		"qwertyuiopasdfgh":       "too common",
		"ANNA.ROSSI.BIANCHI":     "must differ from your username and display name",
	} {
		r := changeTo(b, password)
		if r.StatusCode != http.StatusBadRequest || !strings.Contains(r.body, want) {
			t.Errorf("%.20q: %d, want %d saying %q", password, r.StatusCode, http.StatusBadRequest, want)
		}
	}
	if r := app.browser().signIn("anna.rossi.bianchi", "spoon kettle garden"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("a refused password replaced the old one: %d", r.StatusCode)
	}
}

func TestPassphraseAccepted(t *testing.T) {
	app := newApp(t)
	b := app.signedInAs("anna", "Anna")
	if r := changeTo(b, "lemon chair river clouds"); r.StatusCode != http.StatusOK ||
		!strings.Contains(r.body, "Password changed.") {
		t.Fatalf("passphrase: %d", r.StatusCode)
	}
	if r := app.browser().signIn("anna", "lemon chair river clouds"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("sign in with the passphrase: %d", r.StatusCode)
	}
}

func TestBusySignIn(t *testing.T) {
	app := newApp(t)
	app.addUser("anna", "Anna", "spoon kettle garden")
	b := app.browser()
	release := user.TakePasswordChecks(50 * time.Millisecond)
	r := b.signIn("anna", "spoon kettle garden")
	release()
	if r.StatusCode != http.StatusServiceUnavailable || !strings.Contains(r.body, "Too many sign-ins at once") {
		t.Errorf("sign-in with every password check taken: %d, want %d with the message", r.StatusCode,
			http.StatusServiceUnavailable)
	}
}

func TestBusyPasswordChange(t *testing.T) {
	b := newApp(t).signedInAs("anna", "Anna")
	release := user.TakePasswordChecks(50 * time.Millisecond)
	r := changeTo(b, "lemon chair river clouds")
	release()
	if r.StatusCode != http.StatusServiceUnavailable || !strings.Contains(r.body, "Too many sign-ins at once") {
		t.Errorf("password change with every password check taken: %d, want %d with the message", r.StatusCode,
			http.StatusServiceUnavailable)
	}
}
