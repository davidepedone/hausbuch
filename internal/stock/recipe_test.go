package stock

import (
	"slices"
	"testing"
)

func TestCheck(t *testing.T) {
	rows := []Item{
		{Values: Values{Name: "Flour", Location: "Cupboard"}},
		{Values: Values{Name: "Olive oil", Location: "Shelf"}},
		{Values: Values{Name: "Sesame oil", Location: "Shelf"}},
		{Values: Values{Name: "Bag of rice", Location: "Shelf"}},
	}
	entries := []Entry{{Name: "Butter"}, {Name: "Milk", Bought: true}, {Name: "flour"}}
	checked := Check([]string{"200 g flour", "Eggs", "50 g butter", "Oil", "1 l milk", "½", "1 tbsp oil", "A pinch of salt", "Farina 00"}, rows, entries)

	rowNames := func(i Ingredient) []string {
		var names []string
		for _, it := range i.Rows {
			names = append(names, it.Name)
		}
		return names
	}
	cases := []struct {
		rows    []string
		listed  bool
		missing bool
	}{
		{rows: []string{"Flour"}}, // in the pantry, quantity words ignored; the entry doesn't count
		{missing: true},           // missing
		{listed: true},            // already on the list
		{rows: []string{"Olive oil", "Sesame oil"}}, // shorter ingredient
		{missing: true}, // a bought entry is not on the list
		{missing: true}, // no words, no match
		{rows: []string{"Olive oil", "Sesame oil"}}, // longer ingredient
		{missing: true}, // filler words don't match
		{missing: true}, // "00" holds digits, "farina" is not "flour"
	}
	for i, c := range cases {
		got := checked[i]
		if !slices.Equal(rowNames(got), c.rows) || got.Listed != c.listed || got.Missing() != c.missing {
			t.Errorf("%q: rows %q, listed %v, missing %v; want %q, %v, %v",
				got.Text, rowNames(got), got.Listed, got.Missing(), c.rows, c.listed, c.missing)
		}
	}
}
