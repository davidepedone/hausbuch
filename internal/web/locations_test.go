package web

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidepedone/hausbuch/internal/stock"
)

func TestLocationsAcrossSections(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "location": {"Cellar"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Beans"}, "location": {"Cellar"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Milk"}, "location": {"Fridge"}})
	b.addItem(stock.Supplies, url.Values{"name": {"Detergent"}, "location": {"cellar"}})
	mustContain(t, b.view("/"), `href="/locations"`)
	body := b.view("/locations")
	mustContain(t, body, `href="/locations/Cellar"`, ">Cellar</span>", "3 items", `href="/locations/Fridge"`, "1 item<")
	if strings.Index(body, ">Cellar<") > strings.Index(body, ">Fridge<") {
		t.Error("not in alphabetical order")
	}
}

func TestNoLocationsYet(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}})
	mustContain(t, b.view("/locations"), "No item has a location yet.")
}

func TestWhatIsInTheCellar(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "location": {"Cellar"}})
	b.addItem(stock.Supplies, url.Values{"name": {"Detergent"}, "location": {"cellar"}})
	b.addItem(stock.Supplies, url.Values{"name": {"Soap"}, "location": {"Bathroom"}})
	passata, detergent := a.item(stock.Pantry, "Passata"), a.item(stock.Supplies, "Detergent")
	body := b.view("/locations/Cellar")
	mustContain(t, body, ">Pantry</h2>", ">Supplies</h2>",
		`href="/pantry#item-`+strconv.FormatInt(passata.ID, 10)+`"`, `href="/supplies#item-`+strconv.FormatInt(detergent.ID, 10)+`"`)
	if strings.Index(body, "Passata") > strings.Index(body, "Detergent") || strings.Contains(body, "Soap") {
		t.Errorf("wrong rows or order:\n%s", body)
	}
}

func TestFromARowToItsLocation(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "location": {"Cellar"}})
	mustContain(t, b.view("/pantry"), `<a href="/locations/Cellar"`)
}

func TestALocationLeftEmpty(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Supplies, url.Values{"name": {"Bike pump"}, "location": {"Garage"}})
	b.post(itemURL(a.item(stock.Supplies, "Bike pump"))+"/remove", nil)
	mustContain(t, b.view("/locations/Garage"), "Nothing is kept here.")
}

func TestALocationWithASlash(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Milk"}, "location": {"Fridge/door"}})
	mustContain(t, b.view("/locations"), `href="/locations/Fridge%2Fdoor"`)
	mustContain(t, b.view("/locations/Fridge%2Fdoor"), ">Fridge/door</h1>", "Milk")
}

func TestLocationChoicesOnTheForms(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	if got := choices(b.view("/medicines"), "location"); len(got) != 0 {
		t.Errorf("nothing used yet: %q", got)
	}
	b.addItem(stock.Supplies, url.Values{"name": {"Detergent"}, "location": {"cellar"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "location": {"Cellar"}, "category": {"Long-life"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Milk"}, "location": {"Fridge"}})
	want := []string{"Cellar", "Fridge"}
	passata := a.item(stock.Pantry, "Passata")
	e := b.boughtEntry(url.Values{"name": {"Aspirin"}, "section": {"medicines"}})
	for _, path := range []string{"/medicines", itemURL(passata), itemURL(passata) + "/move", entryPath(e) + "/put-away"} {
		body := b.view(path)
		if got := choices(body, "location"); strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
		if got := choices(body, "category"); slices.Contains(got, "Cellar") {
			t.Errorf("%s: location among the category choices %q", path, got)
		}
	}
}
