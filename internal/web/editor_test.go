package web

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

// editorJS loads the editor's script, without a page, with a helper that runs an edit on a body whose selection is
// marked with « and », and returns the body after it, its new selection marked the same way.
func editorJS(t *testing.T) *goja.Runtime {
	t.Helper()
	src, err := os.ReadFile("static/editor.js")
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	if _, err := vm.RunScript("editor.js", string(src)); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.RunString(`
		function run(marked, edit) {
			const start = marked.indexOf("«"), end = marked.indexOf("»") - 1;
			const text = marked.replace("«", "").replace("»", "");
			const e = edit(text, start, end);
			if (!e) return marked;
			const out = text.slice(0, e.start) + e.insert + text.slice(e.end);
			return out.slice(0, e.selStart) + "«" + out.slice(e.selStart, e.selEnd) + "»" + out.slice(e.selEnd);
		}`); err != nil {
		t.Fatal(err)
	}
	return vm
}

func TestFormattingEdits(t *testing.T) {
	vm := editorJS(t)
	for _, c := range []struct{ name, edit, body, want string }{
		{"bold selected text", "edits.bold", "water at «boiling» point", "water at **«boiling»** point"},
		{"bold with nothing selected", "edits.bold", "water «»", "water **«»**"},
		{"link to a page", "edits.pageLink", "see «Coffee machine»", "see [[«Coffee machine»]]"},
		{"list from selected lines", `(t, s, e) => edits.linePrefix(t, s, e, "- ")`,
			"Shopping\nmi«lk\neggs\nbre»ad\nend", "Shopping\n«- milk\n- eggs\n- bread»\nend"},
		{"selection ending after a line break", `(t, s, e) => edits.linePrefix(t, s, e, "- ")`,
			"«milk\n»eggs", "«- milk»\neggs"},
		{"checklist on the current line", `(t, s, e) => edits.linePrefix(t, s, e, "- [ ] ")`,
			"Todo\nBuy «»salt", "Todo\n- [ ] Buy «»salt"},
		{"heading", `(t, s, e) => edits.linePrefix(t, s, e, "## ")`, "Ingredients«»\nflour", "## Ingredients«»\nflour"},
		{"link selected text", `(t, s, e) => edits.link(t, s, e, "https://example.com/manual.pdf")`,
			"read «the manual» first", "read [the manual](https://example.com/manual.pdf)«» first"},
		{"link with nothing selected", `(t, s, e) => edits.link(t, s, e, "https://example.com")`,
			"see «»", "see [https://example.com](https://example.com)«»"},
		{"address without scheme", `(t, s, e) => edits.link(t, s, e, " example.com ")`,
			"buy at «the shop»", "buy at [the shop](https://example.com)«»"},
		{"other scheme kept", `(t, s, e) => edits.link(t, s, e, "mailto:a@example.com")`,
			"«write»", "[write](mailto:a@example.com)«»"},
		{"cancel", `(t, s, e) => edits.link(t, s, e, null)`, "read «the manual»", "read «the manual»"},
		{"empty address", `(t, s, e) => edits.link(t, s, e, "  ")`, "read «the manual»", "read «the manual»"},
	} {
		got, err := vm.RunString("run(" + quote(c.body) + ", " + c.edit + ")")
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.String() != c.want {
			t.Errorf("%s: %q, want %q", c.name, got.String(), c.want)
		}
	}
}

func TestPastedLink(t *testing.T) {
	vm := editorJS(t)
	for pasted, want := range map[string]string{
		"https://example.com/manual.pdf":   "https://example.com/manual.pdf",
		" http://example.com\n":            "http://example.com",
		"the booklet":                      "",
		"example.com":                      "",
		"https://example.com and the rest": "",
	} {
		got, err := vm.RunString("edits.pastedLink(" + quote(pasted) + ")")
		if err != nil {
			t.Fatal(err)
		}
		if got.String() != want {
			t.Errorf("%q: %q, want %q", pasted, got.String(), want)
		}
	}
}

// quote writes s as a JavaScript string literal.
func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestEditorHasFormattingButtons(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	html := b.view("/pages/new")
	mustContain(t, html, `<script src="/static/editor.js" defer></script>`, `role="toolbar"`)
	for _, format := range []string{"link", "page", "bold", "heading", "list", "checklist"} {
		mustContain(t, html, `<button type="button" data-format="`+format+`"`)
	}
	if r := b.get("/static/editor.js"); r.StatusCode != http.StatusOK {
		t.Errorf("script: %s", r.Status)
	}
	if strings.Contains(b.view("/"), "editor.js") {
		t.Error("script loaded outside the editor")
	}
}
