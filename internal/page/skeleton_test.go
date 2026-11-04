package page

import (
	"slices"
	"strings"
	"testing"
)

func headings(md string) []string {
	var hs []string
	for line := range strings.Lines(md) {
		if h, ok := strings.CutPrefix(line, "## "); ok {
			hs = append(hs, strings.TrimSpace(h))
		}
	}
	return hs
}

func TestSkeletonHeadings(t *testing.T) {
	want := map[Type]map[string][]string{
		Recipe: {
			"en": {"Servings", "Time", "Ingredients", "Steps", "Notes"},
			"it": {"Porzioni", "Tempo", "Ingredienti", "Procedimento", "Note"},
		},
		HowTo: {
			"en": {"When", "What you need", "Steps", "Notes"},
			"it": {"Quando", "Cosa serve", "Procedimento", "Note"},
		},
		WhereIsIt: {
			"en": {"Location", "How to find it", "Notes"},
			"it": {"Dove si trova", "Come trovarlo", "Note"},
		},
	}
	for _, typ := range Types {
		for _, lang := range Languages {
			got := headings(Skeleton(typ, lang))
			if !slices.Equal(got, want[typ][lang]) {
				t.Errorf("%s in %s: headings %q, want %q", typ, lang, got, want[typ][lang])
			}
		}
	}
	if s := Skeleton(Blank, "en") + Skeleton(Blank, "it"); s != "" {
		t.Errorf("Blank skeleton = %q, want empty", s)
	}
}
