package web

import (
	"strings"
	"testing"
)

func TestFooterOfAReleasedBuild(t *testing.T) {
	a := newAppWith(t, "en", "2026.11.3")
	b := a.signedInAs("anna", "Anna")
	for _, p := range []string{"/", "/pantry", "/search?q=x", "/account/password"} {
		mustContain(t, b.view(p), "<span>hausbuch 2026.11.3</span>",
			`<a href="https://github.com/davidepedone/hausbuch/releases/tag/v2026.11.3" class="underline hover:text-stone-900">What's new</a>`,
			`<a href="https://github.com/davidepedone/hausbuch/issues/new/choose" class="underline hover:text-stone-900">Report a problem</a>`)
	}
}

func TestFooterOfADevBuild(t *testing.T) {
	a := newApp(t)
	html := a.signedInAs("anna", "Anna").view("/")
	mustContain(t, html, "<span>hausbuch dev</span>", `<a href="https://github.com/davidepedone/hausbuch/releases" class`)
	_, footer, _ := strings.Cut(html, "<footer")
	if strings.Contains(footer, "target=") {
		t.Errorf("links open elsewhere:\n%s", footer)
	}
}

func TestNoVersionWhenNotSignedIn(t *testing.T) {
	a := newAppWith(t, "en", "2026.11.3")
	login := a.browser().get("/login")
	if strings.Contains(login.body, "2026.11.3") || strings.Contains(login.body, "<footer") {
		t.Errorf("version shown on the sign-in page:\n%s", login.body)
	}
}
