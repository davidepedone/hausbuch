package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/davidepedone/hausbuch/internal/stock"
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
	b.addItem(stock.Pantry, url.Values{"name": {"Boiler descaler"}})
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
		"/search?q=x", "/pantry"} {
		mustContain(t, b.view(p), `<form method="get" action="/search" role="search"`,
			`hx-get="/search" hx-trigger="input changed delay:200ms, search" hx-target="#search-panel"`,
			`<div id="search-panel"`, `<script src="/static/search.js" defer>`)
	}
	mustContain(t, b.search("lawn"), `name="q" value="lawn"`)

	login := a.browser().get("/login")
	if login.StatusCode != http.StatusOK || strings.Contains(login.body, `action="/search"`) {
		t.Errorf("login page: %s, with a search field: %v", login.Status, strings.Contains(login.body, `action="/search"`))
	}
}

func TestGroupedResults(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Coffee machine", "Descale it.")
	b.addItem(stock.Pantry, url.Values{"name": {"Coffee"}})
	b.addItem(stock.Supplies, url.Values{"name": {"Coffee machine descaler"}})
	html := b.search("coffee")
	pages, pantry, supplies := strings.Index(html, ">Pages</h2>"), strings.Index(html, ">Pantry</h2>"), strings.Index(html, ">Supplies</h2>")
	if pages < 0 || pages > pantry || pantry > supplies {
		t.Errorf("groups out of order: %d %d %d\n%s", pages, pantry, supplies, html)
	}
	if strings.Contains(html, ">Medicines</h2>") || strings.Contains(html, "Nothing found") {
		t.Errorf("empty group or no-results message:\n%s", html)
	}
	coffee, descaler := a.item(stock.Pantry, "Coffee"), a.item(stock.Supplies, "Coffee machine descaler")
	mustContain(t, html, `href="`+path+`"`, `href="/pantry#item-`+strconv.FormatInt(coffee.ID, 10)+`"`,
		`href="/supplies#item-`+strconv.FormatInt(descaler.ID, 10)+`"`)
	if strings.Index(html, `href="/pantry#item-`) > strings.Index(html, `href="/supplies#item-`) {
		t.Error("pantry row not before supplies row")
	}
}

func TestStockResult(t *testing.T) {
	a := newApp(t)
	a.stockClock()
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "quantity": {"3"}, "unit": {"bottles"}, "category": {"Long-life"},
		"location": {"Pantry, top shelf"}, "expires_on": {"2027-09-30"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Milk"}, "expires_on": {"2026-11-07"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Yogurt"}, "expires_on": {"2026-11-05"}})
	passata := a.item(stock.Pantry, "Passata")
	html := b.search("PASS")
	mustContain(t, html, `href="/pantry#item-`+strconv.FormatInt(passata.ID, 10)+`"`, ">Passata<",
		"3 bottles · Long-life · Pantry, top shelf", "Expires 30 Sep 2027")
	mustContain(t, b.search("milk"), `text-amber-900">expires tomorrow · 7 Nov 2026<`)
	mustContain(t, b.search("yogurt"), `text-red-800">expired · 5 Nov 2026<`)
}

func TestSearchLabel(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	html := b.view("/")
	mustContain(t, html, `placeholder="Search" aria-label="Search"`)
	if strings.Contains(html, "Search pages") {
		t.Error("still labelled Search pages")
	}
}

// typing asks for the results the way the header's field does while the user types.
func (b *browser) typing(text string) response {
	b.app.t.Helper()
	req, err := http.NewRequest(http.MethodGet, b.app.srv.URL+"/search?q="+url.QueryEscape(text), nil)
	if err != nil {
		b.app.t.Fatal(err)
	}
	req.Header.Set("HX-Request", "true")
	return b.do(req)
}

func TestResultsWhileTyping(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Irrigation", "Turn the tap.")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Pasta"}})
	resp := b.typing("pas")
	if resp.StatusCode != http.StatusOK || strings.Contains(resp.body, "<html") || strings.Contains(resp.body, "<h1") {
		t.Fatalf("not just the results: %s\n%s", resp.Status, resp.body)
	}
	mustContain(t, resp.body, ">Pantry</h2>", ">Passata<", ">Pasta<")
	mustContain(t, b.typing("irri").body, ">Pages</h2>", `href="`+path+`"`)
	mustContain(t, b.typing("boiler").body, "Nothing found for “boiler”.")
}

func TestClearingTheFieldClosesThePanel(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}})
	if resp := b.typing("  "); resp.StatusCode != http.StatusOK || resp.body != "" {
		t.Errorf("empty search: %s %q", resp.Status, resp.body)
	}
}
