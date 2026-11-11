package web

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func (a *testApp) link(username string) string {
	a.t.Helper()
	token, err := a.users.CreateLink(context.Background(), username)
	if err != nil {
		a.t.Fatal(err)
	}
	return "/link/" + token
}

func TestSignInWithALink(t *testing.T) {
	a := newApp(t)
	a.addUser("anna", "Anna Rossi", "pw")
	path := a.link("anna")
	b := a.browser()
	// Opening is not signing in.
	mustContain(t, b.view(path), "signs you in as <strong>Anna Rossi</strong>", `<form method="post" action="`+path+`">`)
	if r := b.get("/"); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("signed in by opening the link: %s", r.Status)
	}
	if r := b.post(path, nil); r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/" {
		t.Fatalf("sign in: %s %s", r.Status, r.Header.Get("Location"))
	}
	mustContain(t, b.view("/"), "Anna Rossi")

	// Used up.
	other := a.browser()
	for _, r := range []response{other.get(path), other.post(path, nil)} {
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("used link: %s", r.Status)
		}
		mustContain(t, r.body, "This link is no longer valid")
	}
	if r := other.get("/"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("signed in by a used link: %s", r.Status)
	}
}

func TestExpiredLink(t *testing.T) {
	a := newApp(t)
	a.addUser("anna", "Anna", "pw")
	now := time.Now()
	a.users.Now = func() time.Time { return now }
	path := a.link("anna")
	now = now.Add(16 * time.Minute)
	b := a.browser()
	if r := b.post(path, nil); r.StatusCode != http.StatusNotFound {
		t.Errorf("expired link: %s", r.Status)
	}
	mustContain(t, b.get(path).body, "This link is no longer valid")
}

func TestLinkOfADisabledAccount(t *testing.T) {
	a := newApp(t)
	a.addUser("anna", "Anna", "pw")
	path := a.link("anna")
	if err := a.users.Disable(context.Background(), "anna"); err != nil {
		t.Fatal(err)
	}
	b := a.browser()
	if r := b.post(path, nil); r.StatusCode != http.StatusNotFound {
		t.Errorf("disabled account: %s", r.Status)
	}
	if r := b.get("/"); r.StatusCode != http.StatusSeeOther {
		t.Errorf("signed in: %s", r.Status)
	}
}
