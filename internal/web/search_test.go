package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func (b *browser) search(text string) string {
	b.app.t.Helper()
	return b.view("/search?q=" + url.QueryEscape(text))
}

func TestSearchResults(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Coffee machine", "Fill the tank.\n\nDescaling every three months.")
	html := b.search("descale")
	mustContain(t, html, `href="`+path+`"`, "Coffee machine", "Blank",
		`Fill the tank. <mark class="rounded bg-amber-100 px-0.5 text-stone-900">Descaling</mark> every three months.`)
}

func TestSearchTitleMatchFirst(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	garden := b.create("Garden", "Irrigation starts in April.")
	irrigation := b.create("Irrigation", "Turn the tap.")
	html := b.search("irrigation")
	if i, g := strings.Index(html, `href="`+irrigation+`"`), strings.Index(html, `href="`+garden+`"`); i < 0 || g < 0 || i > g {
		t.Errorf("Irrigation (at %d) isn't listed before Garden (at %d):\n%s", i, g, html)
	}
}

func TestSearchWithNoResults(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.create("Garden", "Mow the lawn.")
	html := b.search("  boiler manual ")
	mustContain(t, html, "Nothing found for “boiler manual”.", `href="/pages/new?title=boiler&#43;manual"`)
	if strings.Contains(html, `href="/pages/1"`) {
		t.Errorf("a page is listed:\n%s", html)
	}
}

func TestSearchEscapesHTML(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.create("<i>Notes</i>", "The <script>alert(1)</script> key is under the mat.")
	html := b.search("key")
	mustContain(t, html, "&lt;i&gt;Notes&lt;/i&gt;", "The &lt;script&gt;alert(1)&lt;/script&gt; <mark")
	html = b.search(`<script>alert("x")</script>`)
	mustContain(t, html, "Nothing found for “&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;”.")
	if strings.Contains(html, "<script>alert") {
		t.Errorf("searched HTML shown unescaped:\n%s", html)
	}
}

func TestSearchWithOperatorsAndQuotes(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Notes", "Not the garden, near the shed.")
	for _, text := range []string{`NOT garden`, `"shed`, `NEAR(garden shed)`, `garden*`, `( " ) shed`} {
		if html := b.search(text); !strings.Contains(html, `href="`+path+`"`) {
			t.Errorf("%s: Notes not found:\n%s", text, html)
		}
	}
	if r := b.get("/search?q=" + url.QueryEscape(`" ( * ^`)); r.StatusCode != http.StatusOK {
		t.Errorf("search of punctuation only: %s", r.Status)
	}
}

func TestSearchFieldOnEverySignedInScreen(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Garden", "Mow the lawn.")
	for _, p := range []string{"/", path, path + "/edit", path + "/history", "/pages/new", "/deleted", "/account/password",
		"/search?q=x"} {
		mustContain(t, b.view(p), `<form method="get" action="/search" role="search"`)
	}
	mustContain(t, b.search("lawn"), `name="q" value="lawn"`)

	login := a.browser().get("/login")
	if login.StatusCode != http.StatusOK || strings.Contains(login.body, `action="/search"`) {
		t.Errorf("login page: %s, with a search field: %v", login.Status, strings.Contains(login.body, `action="/search"`))
	}
}
