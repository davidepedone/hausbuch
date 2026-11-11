package web

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestPlainHTTPInstallation(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	if r := b.post("/logout", nil); r.StatusCode != http.StatusSeeOther {
		t.Fatal(r.Status)
	}
	for _, body := range []string{b.get("/login").body, a.signedInAs("marco", "Marco").view("/account/password")} {
		if strings.Contains(strings.ToLower(body), "passkey") {
			t.Errorf("passkeys mentioned:\n%s", body)
		}
	}
	if r := b.post("/login/passkey/begin", nil); r.StatusCode != http.StatusNotFound {
		t.Errorf("begin: %s", r.Status)
	}
}

func TestOfferAfterThePassword(t *testing.T) {
	a := newAppWithPasskeys(t)
	a.addUser("anna", "Anna", "spoon kettle garden")
	b := a.browser()
	r := b.post("/login", url.Values{"username": {"anna"}, "password": {"spoon kettle garden"}, "next": {"/pantry"}})
	if want := "/account/passkeys/offer?next=%2Fpantry"; r.Header.Get("Location") != want {
		t.Fatalf("after the password: %s, want %s", r.Header.Get("Location"), want)
	}
	mustContain(t, b.view(r.Header.Get("Location")), "Sign in faster next time", `<a href="/pantry"`, "Not now")
}

func TestNoOfferWithAPasskey(t *testing.T) {
	a := newAppWithPasskeys(t)
	anna := a.addUser("anna", "Anna", "spoon kettle garden")
	if err := a.users.AddPasskey(context.Background(), anna.ID, []byte("k1"), []byte("{}")); err != nil {
		t.Fatal(err)
	}
	r := a.browser().post("/login", url.Values{"username": {"anna"}, "password": {"spoon kettle garden"}, "next": {"/pantry"}})
	if r.Header.Get("Location") != "/pantry" {
		t.Errorf("after the password: %s", r.Header.Get("Location"))
	}
}

func TestOnlyOwnPasskeysRemoved(t *testing.T) {
	a := newAppWithPasskeys(t)
	anna := a.addUser("anna", "Anna", "spoon kettle garden")
	if err := a.users.AddPasskey(context.Background(), anna.ID, []byte("k1"), []byte("{}")); err != nil {
		t.Fatal(err)
	}
	id := base64.RawURLEncoding.EncodeToString([]byte("k1"))
	mustContain(t, a.signedInAs("anna2", "Anna 2").view("/account/password"), "Add a passkey")
	marco := a.signedInAs("marco", "Marco")
	for _, v := range []string{id, "", "%%%"} {
		if r := marco.post("/account/passkeys/remove", url.Values{"id": {v}}); r.StatusCode != http.StatusNotFound {
			t.Errorf("remove %q: %s", v, r.Status)
		}
	}
	if keys, _ := a.users.Passkeys(context.Background(), anna.ID); len(keys) != 1 {
		t.Errorf("anna's passkeys: %d", len(keys))
	}
}
