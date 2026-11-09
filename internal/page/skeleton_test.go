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
	want := map[Type][]string{
		Recipe:    {"Servings", "Time", "Ingredients", "Steps", "Notes"},
		HowTo:     {"When", "What you need", "Steps", "Notes"},
		WhereIsIt: {"Location", "How to find it", "Notes"},
	}
	for _, typ := range Types {
		if got := headings(Skeleton(typ)); !slices.Equal(got, want[typ]) {
			t.Errorf("%s: headings %q, want %q", typ, got, want[typ])
		}
	}
	if s := Skeleton(Blank); s != "" {
		t.Errorf("Blank skeleton = %q, want empty", s)
	}
}
