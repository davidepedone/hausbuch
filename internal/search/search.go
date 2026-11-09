// Package search turns text into the stems the full-text index holds, and searched words into index queries.
package search

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/blevesearch/snowballstem"
	"github.com/blevesearch/snowballstem/english"
	"github.com/blevesearch/snowballstem/italian"
	"golang.org/x/text/unicode/norm"
)

// span is a word's position in a text, as byte offsets.
type span struct{ start, end int }

// spans returns the words of text: runs of letters and digits, so that "l'irrigazione" is two words.
func spans(text string) []span {
	var words []span
	start := -1
	for i, r := range text {
		inWord := unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
		switch {
		case inWord && start < 0:
			start = i
		case !inWord && start >= 0:
			words = append(words, span{start, i})
			start = -1
		}
	}
	if start >= 0 {
		words = append(words, span{start, len(text)})
	}
	return words
}

// Words returns the words of text, lower-cased.
func Words(text string) []string {
	var words []string
	for _, s := range spans(text) {
		words = append(words, strings.ToLower(text[s.start:s.end]))
	}
	return words
}

// Fold makes text comparable ignoring case and accents: "Caffè" and "CAFFE" both fold to "caffe".
func Fold(text string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(text) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// Stem returns the stem of a lower-cased word in the language, "en" or "it".
func Stem(word, lang string) string {
	env := snowballstem.NewEnv(word)
	if lang == "it" {
		italian.Stem(env)
	} else {
		english.Stem(env)
	}
	return env.Current()
}

// Stems returns the stems of the words of text in the language, separated by spaces: what the index holds.
func Stems(text, lang string) string {
	words := Words(text)
	for i, w := range words {
		words[i] = Stem(w, lang)
	}
	return strings.Join(words, " ")
}

// Query returns the full-text query for searched text: every word required, each matching its English or its Italian
// stem. Each stem is quoted, so that nothing in the text acts as a query operator. It is empty when the text has no
// words.
func Query(text string) string {
	var terms []string
	for _, w := range Words(text) {
		en, it := quote(Stem(w, "en")), quote(Stem(w, "it"))
		if en == it {
			terms = append(terms, en)
		} else {
			terms = append(terms, "("+en+" OR "+it+")")
		}
	}
	return strings.Join(terms, " AND ")
}

func quote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// Excerpt is a short piece of a page's text, around the first word matching a search when there is one.
type Excerpt struct {
	Before, Match, After string
}

// How much text an excerpt shows around the match, in bytes, give or take a word.
const (
	excerptBefore = 60
	excerptAfter  = 100
)

// MakeExcerpt returns the excerpt of text, written in lang, for the searched words: around the first word of text
// sharing a stem with one of them, or the start of text when none does (a page matching by its title).
func MakeExcerpt(text, lang, searched string) Excerpt {
	stems := map[string]bool{}
	for _, w := range Words(searched) {
		stems[Stem(w, "en")] = true
		stems[Stem(w, "it")] = true
	}
	for _, s := range spans(text) {
		if stems[Stem(strings.ToLower(text[s.start:s.end]), lang)] {
			return Excerpt{
				Before: textBefore(text, s.start),
				Match:  text[s.start:s.end],
				After:  textAfter(text, s.end, excerptAfter),
			}
		}
	}
	text = strings.TrimSpace(text)
	return Excerpt{After: textAfter(text, 0, excerptBefore+excerptAfter)}
}

// textBefore returns about excerptBefore bytes of text before end, starting at a word.
func textBefore(text string, end int) string {
	start := end - excerptBefore
	if start <= 0 {
		return collapse(text[:end])
	}
	// Starting after a space also keeps the cut off a UTF-8 sequence.
	space := strings.IndexFunc(text[start:end], unicode.IsSpace)
	if space < 0 {
		return ""
	}
	return "…" + collapse(strings.TrimLeftFunc(text[start+space:end], unicode.IsSpace))
}

// textAfter returns about n bytes of text from start, ending at a word.
func textAfter(text string, start, n int) string {
	end := start + n
	if end >= len(text) {
		return collapse(text[start:])
	}
	space := strings.LastIndexFunc(text[start:end], unicode.IsSpace)
	if space < 0 {
		return ""
	}
	return collapse(text[start:start+space]) + "…"
}

// collapse turns every run of white space, line breaks included, into one space.
func collapse(s string) string {
	return whiteSpace.ReplaceAllString(s, " ")
}

var whiteSpace = regexp.MustCompile(`\s+`)
