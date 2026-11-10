package stock

import (
	"strings"
	"unicode"
)

// Ingredient is a recipe's ingredient checked against the pantry and the shopping list.
type Ingredient struct {
	Text   string
	Rows   []Item // the pantry rows it matches
	Listed bool   // whether it matches an entry to buy; false when some row matches
}

// Missing reports whether the ingredient is neither in the pantry nor on the list.
func (i Ingredient) Missing() bool { return len(i.Rows) == 0 && !i.Listed }

// Check matches each ingredient against the pantry's rows and the list's entries. Only presence counts: "200 g flour"
// matches a "Flour" row whatever its quantity.
func Check(ingredients []string, rows []Item, entries []Entry) []Ingredient {
	checked := make([]Ingredient, len(ingredients))
	for i, text := range ingredients {
		checked[i].Text = text
		for _, it := range rows {
			if matches(text, it.Name) {
				checked[i].Rows = append(checked[i].Rows, it)
			}
		}
		for _, e := range entries {
			if len(checked[i].Rows) == 0 && !e.Bought && matches(text, e.Name) {
				checked[i].Listed = true
			}
		}
	}
	return checked
}

// matches reports whether two names share a word.
func matches(a, b string) bool {
	wb := words(b)
	for w := range words(a) {
		if wb[w] {
			return true
		}
	}
	return false
}

// fillers are the English and Italian words too common in names to tell two products apart.
var fillers = map[string]bool{
	"a": true, "an": true, "and": true, "for": true, "in": true, "of": true, "or": true, "the": true, "to": true,
	"with": true,
	"al": true, "alla": true, "con": true, "d": true, "da": true, "dei": true, "del": true, "della": true, "delle": true,
	"di": true, "e": true, "gli": true, "i": true, "il": true, "l": true, "la": true, "le": true, "lo": true, "o": true,
	"per": true, "un": true, "una": true, "uno": true,
}

// words are a name's runs of letters and digits, lower-cased, without those holding a digit and the fillers.
func words(name string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if !fillers[w] && !strings.ContainsFunc(w, unicode.IsDigit) {
			set[w] = true
		}
	}
	return set
}
