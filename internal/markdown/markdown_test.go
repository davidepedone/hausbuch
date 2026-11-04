package markdown

import (
	"slices"
	"strings"
	"testing"
)

func render(t *testing.T, src string) string {
	t.Helper()
	html, err := Render(src, func(title string) (Link, error) {
		if title == "Coffee machine" {
			return Link{Href: "/pages/1"}, nil
		}
		return Link{Href: "/pages/new?title=" + title, Missing: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(html)
}

func TestFormattedBody(t *testing.T) {
	got := render(t, `# Descaling

Use **citric** acid, *not* vinegar.

- one
- two

1. first
2. second

- [x] done
- [ ] to do

| Step | Minutes |
| ---- | ------- |
| Soak | 20      |

Run `+"`descale`"+`, see [the manual](https://example.com/manual).
`)
	for _, want := range []string{
		"<h1>Descaling</h1>",
		"<strong>citric</strong>",
		"<em>not</em>",
		"<ul>\n<li>one</li>",
		"<ol>\n<li>first</li>",
		`<input checked="" disabled="" type="checkbox"> done`,
		`<input disabled="" type="checkbox"> to do`,
		"<table>",
		"<td>Soak</td>",
		"<code>descale</code>",
		`<a href="https://example.com/manual">the manual</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestHTMLInBodyIsShownAsText(t *testing.T) {
	for _, src := range []string{
		"<script>alert(1)</script>",
		"Before <script>alert(1)</script> after",
		"<div onclick=\"alert(1)\">\nhi\n</div>",
		"Click <img src=x onerror=alert(1)>",
	} {
		got := render(t, src)
		if strings.Contains(got, "<script") || strings.Contains(got, "<img") || strings.Contains(got, "<div") {
			t.Errorf("%q rendered raw HTML: %s", src, got)
		}
		if !strings.Contains(got, "&lt;") {
			t.Errorf("%q was not shown as text: %s", src, got)
		}
	}
}

func TestJavaScriptLinkHasNoDestination(t *testing.T) {
	got := render(t, "[click](javascript:alert(1)) and <javascript:alert(1)> and [x](JavaScript:alert(1))")
	if n := strings.Count(got, `<a href="">`); n != 3 {
		t.Errorf("want 3 links without destination, got %d: %s", n, got)
	}
}

func TestWikilinks(t *testing.T) {
	got := render(t, "See [[Coffee machine]], [[Coffee machine|the machine]] and [[Tomato sauce]].")
	for _, want := range []string{
		`<a href="/pages/1">Coffee machine</a>`,
		`<a href="/pages/1">the machine</a>`,
		`<a href="/pages/new?title=Tomato sauce" class="missing">Tomato sauce</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestLinks(t *testing.T) {
	got := Links("[[Pasta]] with [[Tomato sauce|sauce]], [[Pasta]] again, `[[not a link]]`")
	if want := []string{"Pasta", "Tomato sauce"}; !slices.Equal(got, want) {
		t.Errorf("Links = %q, want %q", got, want)
	}
}
