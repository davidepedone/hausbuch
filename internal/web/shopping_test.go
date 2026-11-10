package web

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// entry returns the only entry of the list with that name.
func (a *testApp) entry(name string) stock.Entry {
	a.t.Helper()
	list, err := a.stock.Entries(context.Background())
	if err != nil {
		a.t.Fatal(err)
	}
	var found []stock.Entry
	for _, e := range list {
		if e.Name == name {
			found = append(found, e)
		}
	}
	if len(found) != 1 {
		a.t.Fatalf("%d entries named %q: %+v", len(found), name, list)
	}
	return found[0]
}

func (a *testApp) noEntries() {
	a.t.Helper()
	if list, err := a.stock.Entries(context.Background()); err != nil || len(list) != 0 {
		a.t.Fatalf("list %+v, %v", list, err)
	}
}

func entryPath(e stock.Entry) string { return "/shopping/entries/" + strconv.FormatInt(e.ID, 10) }

// addEntry submits the list's add form and checks it was accepted.
func (b *browser) addEntry(form url.Values) {
	b.app.t.Helper()
	if resp := b.post("/shopping/entries", form); resp.StatusCode != http.StatusSeeOther ||
		resp.Header.Get("Location") != "/shopping" {
		b.app.t.Fatalf("add entry: %s %s\n%s", resp.Status, resp.Header.Get("Location"), resp.body)
	}
}

// boughtEntry adds an entry and ticks it.
func (b *browser) boughtEntry(form url.Values) stock.Entry {
	b.app.t.Helper()
	b.addEntry(form)
	b.post(entryPath(b.app.entry(form.Get("name")))+"/tick", nil)
	e := b.app.entry(form.Get("name"))
	if !e.Bought {
		b.app.t.Fatalf("not bought: %+v", e)
	}
	return e
}

func TestTileOnTheHomePage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	_, nav, _ := strings.Cut(b.view("/"), `aria-label="Sections"`)
	nav, _, _ = strings.Cut(nav, "</nav>")
	mustContain(t, nav, `href="/shopping"`, "Shopping list")
	mustContain(t, b.view("/shopping"), ">Shopping list</h1>")
}

func TestEmptyList(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	mustContain(t, b.view("/shopping"), "Nothing to buy.")
}

func TestSharedByTheHousehold(t *testing.T) {
	a := newApp(t)
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	anna.addEntry(url.Values{"name": {"Rice & beans"}, "quantity": {"2"}, "unit": {"kg"}, "section": {"pantry"},
		"shop": {"Migros"}})
	body := marco.view("/shopping")
	mustContain(t, body, "Rice &amp; beans", "2 kg · Migros", ">Pantry</h2>")
	if strings.Contains(body, "Nothing to buy.") {
		t.Error("says nothing to buy")
	}
}

func TestEntryRefusedOnThePage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	for _, c := range []struct {
		form url.Values
		msg  string
	}{
		{url.Values{"name": {" "}, "shop": {"Coop"}}, "Give the item a name."},
		{url.Values{"name": {"Rice"}, "quantity": {"0"}, "shop": {"Coop"}}, "The quantity must be above 0"},
		{url.Values{"name": {"Rice"}, "section": {"garage"}, "shop": {"Coop"}}, "Choose one of the sections."},
	} {
		resp := b.post("/shopping/entries", c.form)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%v: %s", c.form, resp.Status)
		}
		mustContain(t, resp.body, c.msg, `name="shop" value="Coop"`)
	}
	a.noEntries()
}

func TestEntryActions(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addEntry(url.Values{"name": {"Rice"}})
	e := a.entry("Rice")

	resp := b.hx(entryPath(e)+"/tick", nil)
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(strings.TrimSpace(resp.body), `<div id="entries"`) {
		t.Fatalf("tick: %s\n%s", resp.Status, resp.body)
	}
	mustContain(t, resp.body, ">Bought</h2>", "Not bought yet: Rice")
	if !a.entry("Rice").Bought {
		t.Error("not bought")
	}
	resp = b.post(entryPath(e)+"/untick", nil)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/shopping" || a.entry("Rice").Bought {
		t.Errorf("untick: %s %s", resp.Status, resp.Header.Get("Location"))
	}
	resp = b.hx(entryPath(e)+"/remove", nil)
	mustContain(t, resp.body, "Nothing to buy.")
	a.noEntries()

	if resp := b.post(entryPath(e)+"/tick", nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown entry: %s", resp.Status)
	}
	if resp := b.hx(entryPath(e)+"/tick", nil); resp.StatusCode != http.StatusOK ||
		!strings.Contains(resp.body, `id="entries"`) {
		t.Errorf("unknown entry with HTMX: %s", resp.Status)
	}
	if resp := b.post(entryPath(e)+"/eat", nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown action: %s", resp.Status)
	}
}

func TestClearFromTheList(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.boughtEntry(url.Values{"name": {"Passata"}, "section": {"pantry"}})
	mustContain(t, b.view("/shopping"), "Put away")
	if strings.Contains(b.view("/shopping"), "/shopping/clear") {
		t.Error("clearing offered with every bought entry in a section")
	}
	b.boughtEntry(url.Values{"name": {"Bread"}})
	mustContain(t, b.view("/shopping"), "Clear those without a section")
	resp := b.hx("/shopping/clear", nil)
	mustContain(t, resp.body, "Passata")
	if strings.Contains(resp.body, "Bread") {
		t.Error("Bread still listed")
	}
	a.entry("Passata")
}

func TestListProductFromThePage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}})
	b.addEntry(url.Values{"name": {"Bread"}, "shop": {"Bakery"}})
	resp := b.get("/shopping/product?name=passata")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("known product: %s", resp.Status)
	}
	mustContain(t, resp.body, `id="product-fields"`, `name="section" value="pantry"`)
	mustContain(t, b.get("/shopping/product?name=bread").body, `name="shop" value="Bakery"`)
	if resp := b.get("/shopping/product?name=Pesto"); resp.StatusCode != http.StatusNoContent || resp.body != "" {
		t.Errorf("unknown product: %s %q", resp.Status, resp.body)
	}
	mustContain(t, b.view("/shopping"), `<option value="Bread">`, `<option value="Passata">`,
		`<datalist id="shops"><option value="Bakery">`)
}

func TestShopChipsOnTheList(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addEntry(url.Values{"name": {"Bread"}})
	if strings.Contains(b.view("/shopping"), `aria-label="Shops"`) {
		t.Error("chips without shops")
	}
	b.addEntry(url.Values{"name": {"Butter"}, "shop": {"Migros"}})
	b.addEntry(url.Values{"name": {"Milk"}, "shop": {"migros"}})
	b.addEntry(url.Values{"name": {"Paracetamol"}, "shop": {"Pharmacy"}})
	body := b.view("/shopping")
	if n := strings.Count(body, `data-filter-chip="migros"`); n != 1 {
		t.Errorf("%d migros chips", n)
	}
	mustContain(t, body, `data-filter-chip="pharmacy"`, `data-chip="migros" class=`, `data-chip="" data-chip-any`)
}

func TestListRefreshes(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	mustContain(t, b.view("/shopping"), `<div id="entries" hx-get="/shopping/entries" hx-trigger="every 10s"`)
	b.addEntry(url.Values{"name": {"Butter"}})
	resp := b.get("/shopping/entries")
	if !strings.HasPrefix(strings.TrimSpace(resp.body), `<div id="entries"`) || !strings.Contains(resp.body, "Butter") {
		t.Errorf("refresh: %s\n%s", resp.Status, resp.body)
	}
	if strings.Contains(resp.body, `action="/shopping/entries"`) {
		t.Error("the refresh holds the add form")
	}
}

func TestOnlyTheExpiryLeft(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "category": {"Long-life"}, "location": {"Pantry, top shelf"},
		"opened_days": {"4"}})
	b.hx(itemPath(a.item(stock.Pantry, "Passata"))+"/remove", nil)
	e := b.boughtEntry(url.Values{"name": {"Passata"}, "quantity": {"3"}, "unit": {"bottles"}, "section": {"pantry"}})
	body := b.view(entryPath(e) + "/put-away")
	mustContain(t, body, "Put away Passata", "Into Pantry", `name="name" value="Passata"`, `name="quantity" value="3"`,
		`name="unit" value="bottles"`, `name="category" value="Long-life"`, `name="location" value="Pantry, top shelf"`,
		`name="opened_days" value="4"`, `name="expires_on" value=""`)
}

func TestPutAwayFromThePage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	e := b.boughtEntry(url.Values{"name": {"Rice"}, "section": {"pantry"}})
	mustContain(t, b.view(entryPath(e)+"/put-away"), `name="quantity" value="1"`)

	resp := b.post(entryPath(e)+"/put-away", url.Values{"name": {""}, "quantity": {"2"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("refused: %s", resp.Status)
	}
	mustContain(t, resp.body, "Give the item a name.", `name="quantity" value="2"`)
	a.entry("Rice")

	resp = b.post(entryPath(e)+"/put-away", url.Values{"name": {"Rice"}, "quantity": {"2"}, "expires_on": {"2027-05-31"}})
	it := a.item(stock.Pantry, "Rice")
	if want := "/pantry#item-" + strconv.FormatInt(it.ID, 10); resp.StatusCode != http.StatusSeeOther ||
		resp.Header.Get("Location") != want {
		t.Errorf("saved: %s %s, want %s", resp.Status, resp.Header.Get("Location"), want)
	}
	if it.Quantity != 2*stock.One || it.ExpiresOn.IsZero() {
		t.Errorf("row %+v", it)
	}
	a.noEntries()
	if resp := b.get(entryPath(e) + "/put-away"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("put away twice: %s", resp.Status)
	}
}

func TestNoPutAwayWithoutASection(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addEntry(url.Values{"name": {"Rice"}, "section": {"pantry"}})
	if resp := b.get(entryPath(a.entry("Rice")) + "/put-away"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("not bought: %s", resp.Status)
	}
	e := b.boughtEntry(url.Values{"name": {"Bread"}})
	if strings.Contains(b.view("/shopping"), "/put-away") {
		t.Error("put away offered")
	}
	if resp := b.get(entryPath(e) + "/put-away"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("no section: %s", resp.Status)
	}
	if resp := b.post(entryPath(e)+"/put-away", url.Values{"name": {"Bread"}, "quantity": {"1"}}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("no section, posted: %s", resp.Status)
	}
}

func TestRunningLowFromTheSection(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Medicines, url.Values{"name": {"Paracetamol"}, "unit": {"packs"}})
	it := a.item(stock.Medicines, "Paracetamol")
	mustContain(t, b.view("/medicines"), `action="`+itemPath(it)+`/list"`, "Add to shopping list")
	resp := b.hx(itemPath(it)+"/list", nil)
	mustContain(t, resp.body, "Added Paracetamol to the shopping list.", `id="item-`)
	if e := a.entry("Paracetamol"); e.Section != stock.Medicines || e.Unit != "packs" || e.Quantity != 0 {
		t.Errorf("entry %+v", e)
	}
	if a.item(stock.Medicines, "Paracetamol").Quantity != stock.One {
		t.Error("the row changed")
	}
	if resp := b.post(itemPath(it)+"/list", nil); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("without HTMX: %s", resp.Status)
	}
}

func TestUsedTheLastOne(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "unit": {"bottles"}})
	resp := b.hx(itemPath(a.item(stock.Pantry, "Passata"))+"/decrease", nil)
	mustContain(t, resp.body, `hx-post="/pantry/items/to-list"`, "Add to shopping list")
	resp = b.hx("/pantry/items/to-list", undoForm(t, resp.body))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("add to list: %s\n%s", resp.Status, resp.body)
	}
	mustContain(t, resp.body, "Added Passata to the shopping list.")
	if strings.Contains(resp.body, "Undo") {
		t.Error("undo still offered")
	}
	if e := a.entry("Passata"); e.Section != stock.Pantry || e.Unit != "bottles" || e.Bought {
		t.Errorf("entry %+v", e)
	}
	if items, _ := a.stock.List(context.Background(), stock.Pantry); len(items) != 0 {
		t.Errorf("row back: %+v", items)
	}
}
