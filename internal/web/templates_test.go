package web

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/davidepedone/hausbuch/internal/page"
	"github.com/davidepedone/hausbuch/internal/stock"
)

// textarea is how the editor shows body in its text area.
func textarea(body string) string { return ">\n" + body + "</textarea>" }

// switchTo asks for the body the new page form shows after its type changes.
func (b *browser) switchTo(typ, body, skeleton string) string {
	b.app.t.Helper()
	resp := b.post("/pages/skeleton", url.Values{"title": {"Pizza"}, "type": {typ}, "body": {body}, "skeleton": {skeleton}})
	if resp.StatusCode != http.StatusOK {
		b.app.t.Fatalf("skeleton: %s", resp.Status)
	}
	return resp.body
}

func TestChoosingAType(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	html := b.view("/pages/new")
	mustContain(t, html, `<option value="blank" selected>Blank</option>`, `<option value="recipe">Recipe</option>`,
		`<option value="howto">How-to</option>`, `<option value="where">Where is it</option>`,
		`hx-post="/pages/skeleton"`, `name="skeleton" value=""`, textarea(""))
	if n := strings.Count(html, `hx-post="/pages/skeleton"`); n != 1 {
		t.Errorf("%d fields swap the skeleton, want only the type", n)
	}
}

func TestRecipeSkeleton(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	html := b.switchTo("recipe", "", "")
	mustContain(t, html, textarea(page.Skeleton(page.Recipe)),
		"## Servings", "## Time", "## Ingredients", "## Steps", "## Notes")
	if !strings.Contains(html, `name="skeleton" value="## Servings`) {
		t.Errorf("the new skeleton isn't remembered:\n%s", html)
	}
}

func TestTypeChangedBeforeWriting(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	recipe := page.Skeleton(page.Recipe)
	// Browsers send the line breaks of form fields as CRLF.
	crlf := strings.ReplaceAll(recipe, "\n", "\r\n")
	mustContain(t, b.switchTo("howto", crlf, crlf), textarea(page.Skeleton(page.HowTo)))
}

func TestTypeChangedAfterWriting(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	recipe := page.Skeleton(page.Recipe)
	written := strings.Replace(recipe, "## Time\n", "## Time\n\n20 minutes\n", 1)
	mustContain(t, b.switchTo("howto", written, recipe), textarea(written))
	// Emptied by the user is written too.
	mustContain(t, b.switchTo("howto", "", recipe), textarea(""))
}

func TestSkeletonIsFreeText(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	body := "## Ingredients\n\n- flour\n\n## Wine pairing\n\nRed.\n"
	resp := b.post("/pages", url.Values{"title": {"Pizza"}, "type": {"recipe"},
		"body": {body}, "skeleton": {page.Skeleton(page.Recipe)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create: %s", resp.Status)
	}
	p, err := a.pages.Get(context.Background(), idOf(t, resp.Header.Get("Location")))
	if err != nil {
		t.Fatal(err)
	}
	if p.Body != body || p.Type != page.Recipe {
		t.Errorf("saved %s %q", p.Type, p.Body)
	}
}

func TestRefusedCreateRemembersSkeleton(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	sk := page.Skeleton(page.WhereIsIt)
	resp := b.post("/pages", url.Values{"title": {" "}, "type": {"where"}, "body": {sk},
		"skeleton": {sk}})
	mustContain(t, resp.body, `<option value="where" selected>Where is it</option>`,
		`name="skeleton" value="## Location`, textarea(sk))
}

func TestChangeType(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	body := "## When\n\nIn spring.\n"
	resp := b.post("/pages", url.Values{"title": {"Irrigation"}, "type": {"howto"}, "body": {body}})
	path := resp.Header.Get("Location")
	edit := b.view(path + "/edit")
	mustContain(t, edit, `<option value="howto" selected>How-to</option>`)
	if strings.Contains(edit, "/pages/skeleton") || strings.Contains(edit, `name="skeleton"`) {
		t.Error("the editor of an existing page swaps skeletons")
	}

	if resp := b.saveForm(path, url.Values{"title": {"Irrigation"}, "type": {"where"},
		"body": {body}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("save: %s", resp.Status)
	}
	if html := b.view("/"); !strings.Contains(html, `data-chip="where"`) {
		t.Errorf("not listed under Where is it:\n%s", html)
	}
	p, _ := a.pages.Get(context.Background(), idOf(t, path))
	if p.Body != body {
		t.Errorf("body = %q", p.Body)
	}
}

func TestEveryTypeHasAnIcon(t *testing.T) {
	tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(files, "templates/parts.html"))
	for _, typ := range page.Types {
		var b strings.Builder
		if err := tmpl.ExecuteTemplate(&b, "typeIcon", typ); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), "<path") {
			t.Errorf("%s has no icon: %s", typ, b.String())
		}
	}
}

func TestEverySectionHasAnIcon(t *testing.T) {
	tmpl := template.Must(template.New("").Funcs(funcs).ParseFS(files, "templates/parts.html"))
	for _, sec := range stock.Sections {
		var b strings.Builder
		if err := tmpl.ExecuteTemplate(&b, "sectionIcon", sec); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), "<path") {
			t.Errorf("%s has no icon: %s", sec, b.String())
		}
	}
}
