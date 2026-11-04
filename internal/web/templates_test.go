package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/davidepedone/hausbuch/internal/page"
)

// textarea is how the editor shows body in its text area.
func textarea(body string) string { return ">\n" + body + "</textarea>" }

// switchTo asks for the body the new page form shows after its type or language changes.
func (b *browser) switchTo(typ, lang, body, skeleton string) string {
	b.app.t.Helper()
	resp := b.post("/pages/skeleton", url.Values{"title": {"Pizza"}, "type": {typ}, "language": {lang},
		"body": {body}, "skeleton": {skeleton}})
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
}

func TestRecipeSkeleton(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	html := b.switchTo("recipe", "en", "", "")
	mustContain(t, html, textarea(page.Skeleton(page.Recipe, "en")),
		"## Servings", "## Time", "## Ingredients", "## Steps", "## Notes")
	if !strings.Contains(html, `name="skeleton" value="## Servings`) {
		t.Errorf("the new skeleton isn't remembered:\n%s", html)
	}
}

func TestItalianRecipe(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	mustContain(t, b.switchTo("recipe", "it", "", ""),
		textarea(page.Skeleton(page.Recipe, "it")),
		"## Porzioni", "## Tempo", "## Ingredienti", "## Procedimento", "## Note")
}

func TestItalianDefaultLanguagePrefillsItalian(t *testing.T) {
	a := newAppIn(t, "it")
	b := a.signedInAs("anna", "Anna")
	mustContain(t, b.switchTo("howto", "it", "", ""), textarea(page.Skeleton(page.HowTo, "it")))
}

func TestLanguageChangedBeforeWriting(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	en := page.Skeleton(page.Recipe, "en")
	// Browsers send the line breaks of form fields as CRLF.
	crlf := strings.ReplaceAll(en, "\n", "\r\n")
	html := b.switchTo("recipe", "it", crlf, crlf)
	mustContain(t, html, textarea(page.Skeleton(page.Recipe, "it")))
}

func TestChoiceChangedAfterWriting(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	en := page.Skeleton(page.Recipe, "en")
	written := strings.Replace(en, "## Time\n", "## Time\n\n20 minutes\n", 1)
	for _, c := range []struct{ typ, lang string }{{"recipe", "it"}, {"howto", "en"}} {
		html := b.switchTo(c.typ, c.lang, written, en)
		mustContain(t, html, textarea(written))
	}
	// Emptied by the user is written too.
	mustContain(t, b.switchTo("howto", "en", "", en), textarea(""))
}

func TestSkeletonIsFreeText(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	body := "## Ingredients\n\n- flour\n\n## Wine pairing\n\nRed.\n"
	resp := b.post("/pages", url.Values{"title": {"Pizza"}, "type": {"recipe"}, "language": {"en"},
		"body": {body}, "skeleton": {page.Skeleton(page.Recipe, "en")}})
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
	sk := page.Skeleton(page.WhereIsIt, "en")
	resp := b.post("/pages", url.Values{"title": {" "}, "type": {"where"}, "language": {"en"}, "body": {sk},
		"skeleton": {sk}})
	mustContain(t, resp.body, `<option value="where" selected>Where is it</option>`,
		`name="skeleton" value="## Location`, textarea(sk))
}

func TestChangeType(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	body := "## When\n\nIn spring.\n"
	resp := b.post("/pages", url.Values{"title": {"Irrigation"}, "type": {"howto"}, "language": {"en"}, "body": {body}})
	path := resp.Header.Get("Location")
	edit := b.view(path + "/edit")
	mustContain(t, edit, `<option value="howto" selected>How-to</option>`)
	if strings.Contains(edit, "/pages/skeleton") || strings.Contains(edit, `name="skeleton"`) {
		t.Error("the editor of an existing page swaps skeletons")
	}

	if resp := b.saveForm(path, url.Values{"title": {"Irrigation"}, "type": {"where"}, "language": {"en"},
		"body": {body}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("save: %s", resp.Status)
	}
	if html := b.view("/?type=where"); !strings.Contains(html, ">Irrigation<") {
		t.Errorf("not listed under Where is it:\n%s", html)
	}
	if html := b.view("/?type=howto"); strings.Contains(html, ">Irrigation<") {
		t.Error("still listed under How-to")
	}
	p, _ := a.pages.Get(context.Background(), idOf(t, path))
	if p.Body != body {
		t.Errorf("body = %q", p.Body)
	}
}
