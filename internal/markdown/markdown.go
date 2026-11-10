// Package markdown turns page bodies into HTML, finds the pages they link to and reads the lists in them.
package markdown

import (
	"bytes"
	htmltemplate "html/template"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

// Link is where a [[Title]] leads: the page with that title, or the creation of one when Missing.
type Link struct {
	Href    string
	Missing bool
}

// Resolver tells where the [[Title]] links of a body lead.
type Resolver func(title string) (Link, error)

// The link parser of goldmark is at priority 200: the wikilink one must come first to see "[[".
var wikilinkParser = util.Prioritized(&wikilink.Parser{}, 199)

// Render returns the body as HTML. Raw HTML in the body is shown as the text that was typed, and goldmark already
// leaves out the destination of links with dangerous schemes such as javascript:.
func Render(src string, resolve Resolver) (htmltemplate.HTML, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithInlineParsers(wikilinkParser)),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(util.Prioritized(&nodeRenderer{resolve}, 100)),
			// Pages are written line by line, often on a phone: a line break stays a line break.
			html.WithHardWraps(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return htmltemplate.HTML(buf.String()), nil // safe: raw HTML in the body was escaped by nodeRenderer
}

// Links returns the titles the body links to with [[Title]], each once, in order of appearance.
func Links(src string) []string {
	p := goldmark.New(goldmark.WithParserOptions(parser.WithInlineParsers(wikilinkParser))).Parser()
	var titles []string
	seen := map[string]bool{}
	_ = ast.Walk(p.Parse(text.NewReader([]byte(src))), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if l, ok := n.(*wikilink.Node); ok && entering && !seen[string(l.Target)] {
			seen[string(l.Target)] = true
			titles = append(titles, string(l.Target))
		}
		return ast.WalkContinue, nil
	})
	return titles
}

type nodeRenderer struct {
	resolve Resolver
}

func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, r.htmlBlock)
	reg.Register(ast.KindRawHTML, r.rawHTML)
	reg.Register(wikilink.Kind, r.wikilink)
}

func (r *nodeRenderer) htmlBlock(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.HTMLBlock)
	if !entering {
		_, _ = w.WriteString("</p>\n")
		return ast.WalkContinue, nil
	}
	_, _ = w.WriteString("<p>")
	for i := range n.Lines().Len() {
		seg := n.Lines().At(i)
		_, _ = w.Write(util.EscapeHTML(seg.Value(src)))
	}
	if n.HasClosure() {
		_, _ = w.Write(util.EscapeHTML(n.ClosureLine.Value(src)))
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) rawHTML(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		segs := node.(*ast.RawHTML).Segments
		for i := range segs.Len() {
			seg := segs.At(i)
			_, _ = w.Write(util.EscapeHTML(seg.Value(src)))
		}
	}
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) wikilink(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		_, _ = w.WriteString("</a>")
		return ast.WalkContinue, nil
	}
	link, err := r.resolve(string(node.(*wikilink.Node).Target))
	if err != nil {
		return ast.WalkStop, err
	}
	_, _ = w.WriteString(`<a href="`)
	_, _ = w.Write(util.EscapeHTML([]byte(link.Href)))
	if link.Missing {
		_, _ = w.WriteString(`" class="missing`)
	}
	_, _ = w.WriteString(`">`)
	return ast.WalkContinue, nil
}

// ListUnder returns the text of the list items under the body's first heading that reads heading, ignoring case, up to
// the next heading of the same or a higher level. Items of nested lists count too; a [[Title]] link counts as its title.
func ListUnder(src, heading string) []string {
	p := goldmark.New(goldmark.WithParserOptions(parser.WithInlineParsers(wikilinkParser))).Parser()
	source := []byte(src)
	var items []string
	level := 0 // of the heading found, 0 before it
	for n := p.Parse(text.NewReader(source)).FirstChild(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*ast.Heading); ok {
			if level > 0 && h.Level <= level {
				break
			}
			if level == 0 && strings.EqualFold(strings.TrimSpace(inlineText(h, source)), heading) {
				level = h.Level
			}
			continue
		}
		if level > 0 {
			items = appendItems(items, n, source)
		}
	}
	return items
}

// appendItems appends the text of the list items in n, outer items before the items nested in them.
func appendItems(items []string, n ast.Node, src []byte) []string {
	list, ok := n.(*ast.List)
	if !ok {
		return items
	}
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		var nested []ast.Node
		var parts []string
		for c := item.FirstChild(); c != nil; c = c.NextSibling() {
			if _, ok := c.(*ast.List); ok {
				nested = append(nested, c)
			} else {
				parts = append(parts, inlineText(c, src))
			}
		}
		if t := strings.TrimSpace(strings.Join(parts, " ")); t != "" {
			items = append(items, t)
		}
		for _, l := range nested {
			items = appendItems(items, l, src)
		}
	}
	return items
}

// inlineText is the plain text of n's inline content.
func inlineText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *wikilink.Node:
			b.Write(n.Target)
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			b.Write(n.Segment.Value(src))
			if n.SoftLineBreak() || n.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(n.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}
