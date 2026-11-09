package search

import (
	"slices"
	"testing"
)

func TestWordFormsShareAStem(t *testing.T) {
	for _, c := range []struct{ a, b, lang string }{
		{"descaling", "descale", "en"},
		{"irrigazione", "irrigare", "it"},
		{"ricetta", "ricette", "it"},
	} {
		if a, b := Stem(c.a, c.lang), Stem(c.b, c.lang); a != b {
			t.Errorf("%s: %q stems to %q, %q to %q", c.lang, c.a, a, c.b, b)
		}
	}
}

func TestFold(t *testing.T) {
	for text, want := range map[string]string{"Caffè": "caffe", "CAFFE": "caffe", "Perché": "perche", "Ünïcödé": "unicode",
		"Rice": "rice"} {
		if got := Fold(text); got != want {
			t.Errorf("Fold(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestWords(t *testing.T) {
	for text, want := range map[string][]string{
		"l'irrigazione":             {"l", "irrigazione"},
		"Descaling, every 3 months": {"descaling", "every", "3", "months"},
		"Perché la città?":          {"perché", "la", "città"},
		`"quoted" AND (NOT) x*`:     {"quoted", "and", "not", "x"},
		"<b>bold</b>":               {"b", "bold", "b"},
		" -- ":                      nil,
	} {
		if got := Words(text); !slices.Equal(got, want) {
			t.Errorf("Words(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestStems(t *testing.T) {
	// "irrigazione" stems to "irrigazion" in English and "irrig" in Italian; "parte" to "part" in both.
	if got, want := Stems("L'irrigazione parte"), "l irrigazion irrig part"; got != want {
		t.Errorf("Stems = %q, want %q", got, want)
	}
}

func TestQuery(t *testing.T) {
	for text, want := range map[string]string{
		"descale":                `"descal"`,
		"irrigare":               `("irrigar" OR "irrig")`,
		"garden water":           `"garden" AND "water"`,
		`"x" OR y* NEAR(a b) -c`: `"x" AND "or" AND "y" AND "near" AND "a" AND "b" AND "c"`,
		"<script>":               `"script"`,
		" -- ":                   "",
	} {
		if got := Query(text); got != want {
			t.Errorf("Query(%q) = %s, want %s", text, got, want)
		}
	}
}

func TestExcerpt(t *testing.T) {
	body := "# Coffee machine\n\nThe water tank is behind the lid, under the shelf in the kitchen. " +
		"Descaling every three months keeps the machine working; use the descaler kept in the cupboard under the sink, " +
		"never vinegar."
	got := MakeExcerpt(body, "descale")
	want := Excerpt{
		Before: "…tank is behind the lid, under the shelf in the kitchen. ",
		Match:  "Descaling",
		After:  " every three months keeps the machine working; use the descaler kept in the cupboard under the…",
	}
	if got != want {
		t.Errorf("MakeExcerpt = %#v, want %#v", got, want)
	}
}

func TestExcerptInItalian(t *testing.T) {
	got := MakeExcerpt("L'irrigazione parte ad aprile.", "irrigare")
	if want := (Excerpt{Before: "L'", Match: "irrigazione", After: " parte ad aprile."}); got != want {
		t.Errorf("MakeExcerpt = %#v, want %#v", got, want)
	}
}

func TestExcerptWithoutMatch(t *testing.T) {
	got := MakeExcerpt("\n  Short body\nin two lines.", "irrigation")
	if want := (Excerpt{After: "Short body in two lines."}); got != want {
		t.Errorf("MakeExcerpt = %#v, want %#v", got, want)
	}
}
