package web

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/page"
)

// signedInAs adds an account and returns a browser signed in with it.
func (a *testApp) signedInAs(username, displayName string) *browser {
	a.t.Helper()
	a.addUser(username, displayName, "pw")
	b := a.browser()
	if resp := b.signIn(username, "pw"); resp.StatusCode != http.StatusSeeOther {
		a.t.Fatalf("sign in: %s", resp.Status)
	}
	return b
}

// create submits the new page form and returns the created page's path.
func (b *browser) create(title, body string) string {
	b.app.t.Helper()
	resp := b.post("/pages", url.Values{"title": {title}, "language": {"en"}, "body": {body}})
	if resp.StatusCode != http.StatusSeeOther {
		b.app.t.Fatalf("create %q: %s\n%s", title, resp.Status, resp.body)
	}
	return resp.Header.Get("Location")
}

func (b *browser) save(path, title, body string) response {
	b.app.t.Helper()
	return b.saveForm(path, url.Values{"title": {title}, "language": {"en"}, "body": {body}})
}

// saveForm submits the editor of the page at path, started from its current revision.
func (b *browser) saveForm(path string, form url.Values) response {
	b.app.t.Helper()
	p, err := b.app.pages.Get(context.Background(), idOf(b.app.t, path))
	if err != nil {
		b.app.t.Fatal(err)
	}
	form.Set("revision", strconv.Itoa(p.Revision))
	return b.post(path, form)
}

// view returns the page at path, failing unless it is shown.
func (b *browser) view(path string) string {
	b.app.t.Helper()
	resp := b.get(path)
	if resp.StatusCode != http.StatusOK {
		b.app.t.Fatalf("GET %s: %s", path, resp.Status)
	}
	return resp.body
}

// renderedBody returns the HTML of the page body inside a page view.
func renderedBody(t *testing.T, html string) string {
	t.Helper()
	_, rest, ok := strings.Cut(html, `<div id="body" class="prose prose-sage max-w-none break-words">`)
	body, _, ok2 := strings.Cut(rest, "</div>\n</article>")
	if !ok || !ok2 {
		t.Fatalf("no page body in:\n%s", html)
	}
	return body
}

func mustContain(t *testing.T, html string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
}

func TestCreatePage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Coffee machine", "Descale **monthly**.")
	if !strings.HasPrefix(path, "/pages/") {
		t.Fatalf("redirected to %q", path)
	}
	mustContain(t, b.view(path), "Coffee machine", "<strong>monthly</strong>", "created by Anna")
}

func TestCreatePageWithUsedTitle(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Coffee machine", "")
	resp := b.post("/pages", url.Values{"title": {" coffee MACHINE "}, "language": {"en"}, "body": {"My notes"}})
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %s", resp.Status)
	}
	mustContain(t, resp.body, `<a href="`+path+`" class="font-medium underline">Coffee machine</a>`, "My notes")
}

func TestCreatePageWithEmptyTitle(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	resp := b.post("/pages", url.Values{"title": {"   "}, "language": {"en"}, "body": {"My notes"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %s", resp.Status)
	}
	mustContain(t, resp.body, "Give the page a title.", "My notes")
}

func TestDefaultLanguage(t *testing.T) {
	for lang, want := range map[string]string{"it": `<option value="it" selected>Italian</option>`,
		"en": `<option value="en" selected>English</option>`} {
		a := newAppIn(t, lang)
		b := a.signedInAs("anna", "Anna")
		mustContain(t, b.view("/pages/new"), want)
	}
}

func TestChangeLanguage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Pasta", "Boil water.")
	resp := b.saveForm(path, url.Values{"title": {"Pasta"}, "language": {"it"}, "body": {"Boil water."}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("save: %s", resp.Status)
	}
	html := b.view(path)
	mustContain(t, html, "Italian")
	if got := renderedBody(t, html); got != "<p>Boil water.</p>\n" {
		t.Errorf("body = %q", got)
	}
	mustContain(t, b.view(path+"/edit"), `<option value="it" selected>Italian</option>`)
}

func TestFormattedBodyAndHTML(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Pizza", "# Dough\n\n- flour\n- water\n\n| g | what |\n| - | - |\n| 500 | flour |\n\n<script>alert(1)</script>\n")
	body := renderedBody(t, b.view(path))
	mustContain(t, body, "<h1>Dough</h1>", "<li>flour</li>", "<td>flour</td>", "&lt;script&gt;alert(1)&lt;/script&gt;")
	if strings.Contains(body, "<script") {
		t.Errorf("script rendered: %s", body)
	}
}

func TestLinksBetweenPages(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	machine := b.create("Coffee machine", "")
	path := b.create("Descaling", "See [[Coffee machine]] and [[Citric acid|the acid]].")
	body := renderedBody(t, b.view(path))
	mustContain(t, body,
		`<a href="`+machine+`">Coffee machine</a>`,
		`<a href="/pages/new?title=Citric+acid" class="missing">the acid</a>`)

	mustContain(t, b.view("/pages/new?title=Citric+acid"), `name="title" value="Citric acid"`)
}

func TestBacklinks(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	sauce := b.create("Tomato sauce", "")
	pasta := b.create("Pasta", "With [[Tomato sauce]].")
	menu := b.create("Weekly menu", "Monday: [[Tomato sauce]].")
	mustContain(t, b.view(sauce), "Pages linking here",
		`<a href="`+pasta+`" class="text-sage-700 underline hover:text-sage-800">Pasta</a>`, `<a href="`+menu+`" class="text-sage-700 underline hover:text-sage-800">Weekly menu</a>`)
	if strings.Contains(b.view(pasta), "Pages linking here") {
		t.Error("a page nobody links shows backlinks")
	}
}

func TestEditBody(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	path := anna.create("Pasta", "Boil water.")
	mustContain(t, marco.view(path+"/edit"), `name="title" value="Pasta"`, ">\nBoil water.</textarea>")
	if resp := marco.save(path, "Pasta", "Boil salted water."); resp.StatusCode != http.StatusSeeOther ||
		resp.Header.Get("Location") != path {
		t.Fatalf("save: %s to %q", resp.Status, resp.Header.Get("Location"))
	}
	html := anna.view(path)
	mustContain(t, html, "Boil salted water.", "by Marco")
}

func TestEditorKeepsBodyLineBreaks(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	path := b.create("Pasta", "\nfirst\r\nsecond\r\n")
	p, err := a.pages.Get(context.Background(), idOf(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if p.Body != "\nfirst\nsecond\n" {
		t.Errorf("stored body = %q", p.Body)
	}
	// A leading line break must survive the round trip through the text area, whose first line break is dropped.
	mustContain(t, b.view(path+"/edit"), ">\n\nfirst\nsecond\n</textarea>")
}

func TestRenamedPageKeepsLinks(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	machine := b.create("Coffee maker", "")
	descale := b.create("Descaling", "See [[Coffee maker]].")
	if resp := b.save(machine, "Coffee machine", ""); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("rename: %s", resp.Status)
	}
	mustContain(t, renderedBody(t, b.view(descale)), `<a href="`+machine+`">Coffee maker</a>`)
	mustContain(t, b.view(machine), "Coffee machine", `<a href="`+descale+`" class="text-sage-700 underline hover:text-sage-800">Descaling</a>`)
}

func TestRenameToUsedTitle(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	pasta := b.create("Pasta", "")
	pizza := b.create("Pizza", "")
	resp := b.save(pizza, "pasta", "Dough")
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %s", resp.Status)
	}
	mustContain(t, resp.body, `<a href="`+pasta+`" class="font-medium underline">Pasta</a>`, `action="`+pizza+`"`, "Dough")
}

func TestPreviewMatchesPageView(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.create("Coffee machine", "")
	src := "# Steps\n\n1. [[Coffee machine]]\n2. [[Missing]]\n\n<b>raw</b>\n"
	path := b.create("Descaling", src)
	resp := b.post("/preview", url.Values{"body": {strings.ReplaceAll(src, "\n", "\r\n")}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preview: %s", resp.Status)
	}
	if want := renderedBody(t, b.view(path)); resp.body != want {
		t.Errorf("preview:\n%s\npage view:\n%s", resp.body, want)
	}
}

func TestHomeListsPagesNewestFirst(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	a.pages.Now = func() time.Time { return now }
	pasta := anna.create("Pasta", "")
	now = now.Add(time.Minute)
	anna.create("Irrigation", "")
	now = now.Add(time.Minute)
	marco.save(pasta, "Pasta", "Boil water.")

	html := anna.view("/")
	i, j := strings.Index(html, ">Pasta<"), strings.Index(html, ">Irrigation<")
	if i < 0 || j < 0 || i > j {
		t.Errorf("Pasta at %d, Irrigation at %d:\n%s", i, j, html)
	}
	mustContain(t, html, "Blank · changed "+now.Local().Format("2 Jan 2006, 15:04")+" by Marco")
}

func TestHomeFiltersByType(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.create("Irrigation", "")
	anna, _ := a.users.Authenticate(context.Background(), "anna", "pw")
	if _, err := a.pages.Create(context.Background(), anna.ID,
		page.Content{Title: "Pizza", Type: page.Recipe, Language: "en"}); err != nil {
		t.Fatal(err)
	}
	html := b.view("/?type=recipe")
	if !strings.Contains(html, ">Pizza<") || strings.Contains(html, ">Irrigation<") {
		t.Errorf("recipes:\n%s", html)
	}
	mustContain(t, html, `<a href="/?type=recipe" class="flex flex-col items-center gap-1 rounded-lg border px-3 py-3 font-medium border-sage-700 bg-sage-700 text-white" aria-current="page">`,
		`<a href="/" class="text-sm text-sage-700 underline hover:text-sage-800">Show all</a>`)
	if html := b.view("/?type=nonsense"); !strings.Contains(html, ">Pizza<") || !strings.Contains(html, ">Irrigation<") {
		t.Errorf("unknown type doesn't list all:\n%s", html)
	}
}

func idOf(t *testing.T, path string) int64 {
	t.Helper()
	id, err := strconv.ParseInt(strings.TrimPrefix(path, "/pages/"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestDeleteAndRestorePage(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	a.pages.Now = func() time.Time { return now }
	sauce := anna.create("Tomato sauce", "Simmer.")
	pasta := anna.create("Pasta", "With [[Tomato sauce]].")
	anna.save(sauce, "Tomato sauce", "Simmer slowly.")

	if resp := marco.post(sauce+"/delete", nil); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete: %s", resp.Status)
	}
	if strings.Contains(anna.view("/"), ">Tomato sauce<") {
		t.Error("deleted page still listed")
	}
	if resp := anna.get(sauce); resp.StatusCode != http.StatusNotFound {
		t.Errorf("deleted page: %s", resp.Status)
	}
	mustContain(t, renderedBody(t, anna.view(pasta)), `class="missing">Tomato sauce</a>`)
	mustContain(t, anna.view("/deleted"), "Tomato sauce",
		"Deleted "+now.Local().Format("2 Jan 2006, 15:04")+" by Marco · removed on "+
			now.Add(30*24*time.Hour).Local().Format("2 Jan 2006"),
		`action="`+sauce+`/restore"`)

	// Creating a page with its title offers to restore it.
	resp := anna.post("/pages", url.Values{"title": {"Tomato sauce"}, "language": {"en"}})
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %s", resp.Status)
	}
	mustContain(t, resp.body, "A deleted page is called", `action="`+sauce+`/restore"`)

	if resp := anna.post(sauce+"/restore", nil); resp.StatusCode != http.StatusSeeOther ||
		resp.Header.Get("Location") != sauce {
		t.Fatalf("restore: %s to %q", resp.Status, resp.Header.Get("Location"))
	}
	mustContain(t, anna.view(sauce), "Simmer slowly.")
	p, err := a.pages.Get(context.Background(), idOf(t, sauce))
	if err != nil || p.Revision != 2 {
		t.Errorf("restored page: revision %d, %v", p.Revision, err)
	}
	mustContain(t, renderedBody(t, anna.view(pasta)), `<a href="`+sauce+`">Tomato sauce</a>`)
	if strings.Contains(anna.view("/deleted"), "Tomato sauce") {
		t.Error("restored page still among deleted pages")
	}
}

func TestDeletedPageGoneAfter30Days(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	a.pages.Now = func() time.Time { return now }
	path := b.create("Old notes", "")
	b.post(path+"/delete", nil)
	now = now.Add(30*24*time.Hour + time.Second)

	if strings.Contains(b.view("/deleted"), "Old notes") {
		t.Error("still among deleted pages")
	}
	if resp := b.post(path+"/restore", nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("restore: %s", resp.Status)
	}
	if again := b.create("Old notes", ""); again == path {
		t.Errorf("new page has the removed page's address %q", again)
	}
}
