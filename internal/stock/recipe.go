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

// matches reports whether every word of the shorter of two names is a word of the other, ignoring case.
func matches(a, b string) bool {
	wa, wb := words(a), words(b)
	if len(wa) > len(wb) {
		wa, wb = wb, wa
	}
	if len(wa) == 0 {
		return false
	}
	for w := range wa {
		if !wb[w] {
			return false
		}
	}
	return true
}

// words are a name's runs of letters and digits, lower-cased.
func words(name string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		set[w] = true
	}
	return set
}
