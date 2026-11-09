package page

import (
	"slices"
	"testing"
)

func (f *fixture) createIn(lang, title, body string) int64 {
	f.t.Helper()
	id, err := f.pages.Create(ctx, f.anna.ID, Content{Title: title, Type: Blank, Language: lang, Body: body})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) search(text string) []int64 {
	f.t.Helper()
	results, err := f.pages.Search(ctx, text)
	if err != nil {
		f.t.Fatalf("search %q: %v", text, err)
	}
	var ids []int64
	for _, r := range results {
		ids = append(ids, r.ID)
	}
	return ids
}

func TestSearchEnglishWordForms(t *testing.T) {
	f := newFixture(t)
	id := f.createIn("en", "Coffee machine", "Descaling every three months.")
	if got := f.search("descale"); !slices.Contains(got, id) {
		t.Errorf("descale: %v, want Coffee machine (%d) among them", got, id)
	}
}

func TestSearchItalianWordForms(t *testing.T) {
	f := newFixture(t)
	id := f.createIn("it", "Giardino", "L'irrigazione parte ad aprile.")
	if got := f.search("irrigare"); !slices.Contains(got, id) {
		t.Errorf("irrigare: %v, want Giardino (%d) among them", got, id)
	}
}

func TestSearchWithoutChoosingALanguage(t *testing.T) {
	f := newFixture(t)
	a := f.createIn("it", "Ricette di casa", "Le ricette della nonna.")
	b := f.createIn("it", "Tiramisù", "La ricetta: mascarpone, uova, savoiardi.")
	f.createIn("en", "Coffee machine", "Descaling.")
	if got, want := f.search("ricetta"), []int64{a, b}; !sameIDs(got, want) {
		t.Errorf("ricetta: %v, want %v", got, want)
	}
}

func TestSearchNeedsAllWords(t *testing.T) {
	f := newFixture(t)
	f.createIn("en", "Garden", "Mow the lawn in spring.")
	both := f.createIn("en", "Irrigation", "Water the garden at dawn.")
	if got := f.search("garden water"); !slices.Equal(got, []int64{both}) {
		t.Errorf("garden water: %v, want only %d", got, both)
	}
}

func TestSearchIgnoresCaseAndAccents(t *testing.T) {
	f := newFixture(t)
	id := f.createIn("it", "Città", "Perché il caffè è buono.")
	for _, text := range []string{"CITTÀ", "citta", "caffe", "PERCHE"} {
		if got := f.search(text); !slices.Equal(got, []int64{id}) {
			t.Errorf("%s: %v, want %d", text, got, id)
		}
	}
}

func TestSearchSkipsDeletedPages(t *testing.T) {
	f := newFixture(t)
	id := f.createIn("en", "Old boiler", "The boiler manual is in the cellar.")
	if err := f.pages.Delete(ctx, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	if got := f.search("boiler"); len(got) != 0 {
		t.Errorf("boiler: %v, want nothing", got)
	}
}

func TestSearchPutsTitleMatchesFirst(t *testing.T) {
	f := newFixture(t)
	garden := f.createIn("en", "Garden", "Irrigation starts in April; irrigation stops in October. Irrigation!")
	irrigation := f.createIn("en", "Irrigation", "Turn the tap.")
	if got, want := f.search("irrigation"), []int64{irrigation, garden}; !slices.Equal(got, want) {
		t.Errorf("irrigation: %v, want %v", got, want)
	}
}

func TestSearchExcerpt(t *testing.T) {
	f := newFixture(t)
	f.createIn("en", "Coffee machine", "Fill the tank.\n\nDescaling every three months.")
	results, err := f.pages.Search(ctx, "descale")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Excerpt.Match != "Descaling" {
		t.Errorf("results %+v, want one with Descaling matched", results)
	}
}

func TestSearchTakesQueryOperatorsLiterally(t *testing.T) {
	f := newFixture(t)
	id := f.createIn("en", "Notes", "Not the garden, near the shed: the key.")
	for text, want := range map[string][]int64{
		`NOT garden`:             {id},
		`garden OR nothing`:      nil,
		`NEAR(garden shed)`:      {id},
		`"garden`:                {id},
		`title_stem : garden`:    nil,
		`ke*`:                    nil,
		`<b>key</b>`:             nil,
		`key -- " ' ( ) { } ^ +`: {id},
		`" ( *`:                  nil,
	} {
		if got := f.search(text); !slices.Equal(got, want) {
			t.Errorf("%s: %v, want %v", text, got, want)
		}
	}
}

func sameIDs(a, b []int64) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func TestSearchPartOfATitle(t *testing.T) {
	f := newFixture(t)
	irrigation := f.createIn("en", "Irrigation", "Turn the tap.")
	coffee := f.createIn("it", "Caffè con la moka", "Riempi la base.")
	f.createIn("en", "Garden", "Mow the lawn.")
	if got := f.search("irri"); !slices.Equal(got, []int64{irrigation}) {
		t.Errorf("irri: %v, want Irrigation (%d)", got, irrigation)
	}
	if got := f.search("CAFFE MOK"); !slices.Equal(got, []int64{coffee}) {
		t.Errorf("CAFFE MOK: %v, want Caffè con la moka (%d)", got, coffee)
	}
	if got := f.search("irri lawn"); len(got) != 0 {
		t.Errorf("irri lawn: %v, want nothing", got)
	}
}

func TestSearchPutsPartialTitleMatchesFirst(t *testing.T) {
	f := newFixture(t)
	body := f.createIn("en", "Garden", "Watering every evening.")
	stem := f.createIn("en", "Watering", "Turn the tap.")
	part := f.createIn("en", "Waterproof boots", "In the cellar.")
	got := f.search("water")
	if len(got) != 3 || got[2] != body || !slices.Contains(got[:2], stem) || !slices.Contains(got[:2], part) {
		t.Errorf("water: %v, want %d and %d before %d", got, stem, part, body)
	}
}

func TestSearchSkipsDeletedTitles(t *testing.T) {
	f := newFixture(t)
	id := f.createIn("en", "Irrigation", "Turn the tap.")
	if err := f.pages.Delete(ctx, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	if got := f.search("irri"); len(got) != 0 {
		t.Errorf("irri: %v, want nothing", got)
	}
}
