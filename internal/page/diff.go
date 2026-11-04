package page

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffLine is a line of a diff: in both texts, only in the new one (Added) or only in the old one (Removed).
type DiffLine struct {
	Text    string
	Added   bool
	Removed bool
}

// Diff compares two texts line by line.
func Diff(old, new string) []DiffLine {
	dmp := diffmatchpatch.New()
	a, b, lines := dmp.DiffLinesToChars(old, new)
	var out []DiffLine
	for _, d := range dmp.DiffCharsToLines(dmp.DiffMain(a, b, false), lines) {
		for _, line := range strings.SplitAfter(d.Text, "\n") {
			if line == "" {
				continue
			}
			out = append(out, DiffLine{
				Text:    strings.TrimSuffix(line, "\n"),
				Added:   d.Type == diffmatchpatch.DiffInsert,
				Removed: d.Type == diffmatchpatch.DiffDelete,
			})
		}
	}
	return out
}
