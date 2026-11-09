package web

import (
	"context"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/stock"
)

// stockClock fixes the stock store's clock on 6 November 2026 and returns it, to move it.
func (a *testApp) stockClock() *time.Time {
	now := time.Date(2026, 11, 6, 9, 0, 0, 0, time.Local)
	a.stock.Now = func() time.Time { return now }
	return &now
}

// addItem submits the add form of a section.
func (b *browser) addItem(section stock.Section, form url.Values) response {
	b.app.t.Helper()
	if form.Get("quantity") == "" {
		form.Set("quantity", "1")
	}
	return b.post("/"+string(section)+"/items", form)
}

// item returns the only row of the section with that name.
func (a *testApp) item(section stock.Section, name string) stock.Item {
	a.t.Helper()
	items, err := a.stock.List(context.Background(), section)
	if err != nil {
		a.t.Fatal(err)
	}
	var found []stock.Item
	for _, it := range items {
		if it.Name == name {
			found = append(found, it)
		}
	}
	if len(found) != 1 {
		a.t.Fatalf("%d rows named %q in %s", len(found), name, section)
	}
	return found[0]
}

// hx posts the way HTMX does from a row's button.
func (b *browser) hx(path string, form url.Values) response {
	b.app.t.Helper()
	req, err := http.NewRequest(http.MethodPost, b.app.srv.URL+path, strings.NewReader(form.Encode()))
	if err != nil {
		b.app.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("HX-Request", "true")
	return b.do(req)
}

func itemPath(it stock.Item) string {
	return "/" + string(it.Section) + "/items/" + strconv.FormatInt(it.ID, 10)
}

func TestSectionPages(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	for _, path := range []string{"/medicines", "/pantry"} {
		mustContain(t, b.view(path), "Add item", "Nothing here yet.")
	}
}

func TestAddAnItem(t *testing.T) {
	a := newApp(t)
	a.stockClock()
	b := a.signedInAs("anna", "Anna")
	resp := b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "quantity": {"3"}, "unit": {"bottles"},
		"location": {"Pantry, top shelf"}, "expires_on": {"2027-09-30"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/pantry" {
		t.Fatalf("add: %s %s", resp.Status, resp.Header.Get("Location"))
	}
	html := b.view("/pantry")
	mustContain(t, html, ">Passata<", "bottles · Pantry, top shelf", "Expires 30 Sep 2027", ">3<")
	if strings.Contains(b.view("/medicines"), "Passata") {
		t.Error("pantry item listed in medicines")
	}
	if it := a.item(stock.Pantry, "Passata"); it.CreatedBy != "Anna" {
		t.Errorf("added by %q", it.CreatedBy)
	}
}

func TestAddADecimalQuantity(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Flour"}, "quantity": {"1,5"}, "unit": {"kg"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}, "quantity": {"2.0"}})
	mustContain(t, b.view("/pantry"), ">1.5<", ">2<")
	it := a.item(stock.Pantry, "Flour")
	mustContain(t, b.view(itemPath(it)), `name="quantity" value="1.5"`)
}

func TestAddAnItemWithoutExpiryDate(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Salt"}})
	html := b.view("/pantry")
	mustContain(t, html, ">Salt<")
	if strings.Contains(html, "Expires") {
		t.Error("expiry shown for an item without one")
	}
}

func TestAddRefused(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	for _, c := range []struct {
		form url.Values
		msg  string
	}{
		{url.Values{"name": {" "}, "quantity": {"2"}, "location": {"Cellar"}}, "Give the item a name."},
		{url.Values{"name": {"Rice"}, "quantity": {"0"}, "location": {"Cellar"}}, "The quantity must be above 0"},
		{url.Values{"name": {"Rice"}, "quantity": {"-1"}, "location": {"Cellar"}}, "The quantity must be above 0"},
		{url.Values{"name": {"Rice"}, "quantity": {"two"}, "location": {"Cellar"}}, "The quantity must be above 0"},
		{url.Values{"name": {"Rice"}, "quantity": {"1.25"}, "location": {"Cellar"}}, "The quantity must be above 0"},
		{url.Values{"name": {"Rice"}, "quantity": {"1"}, "location": {"Cellar"}, "expires_on": {"31/09/2027"}}, "A date isn&#39;t valid."},
	} {
		resp := b.addItem(stock.Pantry, c.form)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%v: %s", c.form, resp.Status)
		}
		mustContain(t, resp.body, c.msg, `name="location" value="Cellar"`)
	}
	if items, _ := a.stock.List(context.Background(), stock.Pantry); len(items) != 0 {
		t.Errorf("saved: %v", items)
	}
}

func TestEditAnItem(t *testing.T) {
	a := newApp(t)
	now := a.stockClock()
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	anna.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "quantity": {"3"}, "location": {"Pantry"}})
	it := a.item(stock.Pantry, "Passata")
	mustContain(t, marco.view(itemPath(it)), `name="location" value="Pantry"`, "Added by Anna")
	*now = now.Add(time.Hour)
	resp := marco.post(itemPath(it), url.Values{"name": {"Passata"}, "quantity": {"3"}, "location": {"Cellar"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("save: %s", resp.Status)
	}
	if it := a.item(stock.Pantry, "Passata"); it.Location != "Cellar" || it.ChangedBy != "Marco" {
		t.Errorf("got %+v", it)
	}
	resp = marco.post(itemPath(it), url.Values{"name": {""}, "quantity": {"3"}, "location": {"Attic"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty name: %s", resp.Status)
	}
	mustContain(t, resp.body, "Give the item a name.", `name="location" value="Attic"`)
}

func TestItemOfAnotherSection(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Medicines, url.Values{"name": {"Paracetamol"}})
	it := a.item(stock.Medicines, "Paracetamol")
	wrong := "/pantry/items/" + strconv.FormatInt(it.ID, 10)
	if r := b.get(wrong); r.StatusCode != http.StatusNotFound {
		t.Errorf("GET: %s", r.Status)
	}
	if r := b.hx(wrong+"/decrease", nil); r.StatusCode != http.StatusNotFound {
		t.Errorf("decrease: %s", r.Status)
	}
	if it := a.item(stock.Medicines, "Paracetamol"); it.Quantity != stock.One {
		t.Errorf("quantity %v", it.Quantity)
	}
}

func TestWarningsInTheSections(t *testing.T) {
	a := newApp(t)
	a.stockClock()
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Yogurt"}, "expires_on": {"2026-11-05"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Milk"}, "expires_on": {"2026-11-09"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}, "expires_on": {"2027-11-09"}})
	html := b.view("/pantry")
	mustContain(t, html, ">Expired</span>", ">expires in 3 days</span>")
	_, rice, _ := strings.Cut(html, ">Rice<")
	rice, _, _ = strings.Cut(rice, "</li>")
	if strings.Contains(rice, ">Expired<") || strings.Contains(rice, "expires in") {
		t.Errorf("Rice marked:\n%s", rice)
	}
}

func TestUseOneFromTheList(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "quantity": {"3"}})
	it := a.item(stock.Pantry, "Passata")
	resp := b.hx(itemPath(it)+"/decrease", nil)
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(strings.TrimSpace(resp.body), `<div id="items">`) {
		t.Fatalf("decrease: %s\n%s", resp.Status, resp.body)
	}
	mustContain(t, resp.body, ">2<")
	if strings.Contains(resp.body, "<html") || strings.Contains(resp.body, "Undo") {
		t.Errorf("not just the list:\n%s", resp.body)
	}
	b.hx(itemPath(it)+"/increase", nil)
	if q := a.item(stock.Pantry, "Passata").Quantity; q != 3*stock.One {
		t.Errorf("after increase: %v", q)
	}
	// Without HTMX, the button still works and leads back to the section.
	if r := b.post(itemPath(it)+"/decrease", nil); r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != "/pantry" {
		t.Errorf("without HTMX: %s %s", r.Status, r.Header.Get("Location"))
	}
}

var hiddenField = regexp.MustCompile(`<input type="hidden" name="(\w+)" value="([^"]*)">`)

// undoForm returns the fields of the undo offer in a list.
func undoForm(t *testing.T, body string) url.Values {
	t.Helper()
	_, offer, ok := strings.Cut(body, `action="/pantry/items/restore"`)
	offer, _, _ = strings.Cut(offer, "</form>")
	if !ok {
		t.Fatalf("no undo offer:\n%s", body)
	}
	form := url.Values{}
	for _, m := range hiddenField.FindAllStringSubmatch(offer, -1) {
		form.Set(m[1], html.UnescapeString(m[2]))
	}
	return form
}

func TestUseTheLastOneAndUndo(t *testing.T) {
	a := newApp(t)
	a.stockClock()
	anna := a.signedInAs("anna", "Anna")
	marco := a.signedInAs("marco", "Marco")
	anna.addItem(stock.Pantry, url.Values{"name": {"Passata & co"}, "unit": {"bottles"}, "location": {"Pantry"},
		"expires_on": {"2027-09-30"}, "opened_days": {"4"}})
	anna.hx(itemPath(a.item(stock.Pantry, "Passata & co"))+"/open", nil)
	before := a.item(stock.Pantry, "Passata & co")
	resp := marco.hx(itemPath(before)+"/decrease", nil)
	mustContain(t, resp.body, "Removed Passata &amp; co.", "Undo", `class="fade-away`)
	if strings.Contains(resp.body, `id="item-`) {
		t.Error("row still listed")
	}
	resp = marco.hx("/pantry/items/restore", undoForm(t, resp.body))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("undo: %s\n%s", resp.Status, resp.body)
	}
	after := a.item(stock.Pantry, "Passata & co")
	if after.Values != before.Values || after.CreatedBy != "Anna" || !after.CreatedAt.Equal(before.CreatedAt) {
		t.Errorf("restored %+v, was %+v", after, before)
	}
	if strings.Contains(resp.body, "Undo") {
		t.Error("undo offered again after undoing")
	}
}

func TestRemoveAndUndo(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Yogurt"}, "quantity": {"2"}})
	resp := b.hx(itemPath(a.item(stock.Pantry, "Yogurt"))+"/remove", nil)
	mustContain(t, resp.body, "Removed Yogurt.")
	if items, _ := a.stock.List(context.Background(), stock.Pantry); len(items) != 0 {
		t.Fatalf("still there: %v", items)
	}
	b.hx("/pantry/items/restore", undoForm(t, resp.body))
	if it := a.item(stock.Pantry, "Yogurt"); it.Quantity != 2*stock.One {
		t.Errorf("restored quantity %v", it.Quantity)
	}
}

func TestUndoAPart(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Flour"}, "quantity": {"0,5"}})
	resp := b.hx(itemPath(a.item(stock.Pantry, "Flour"))+"/decrease", nil)
	mustContain(t, resp.body, "Removed Flour.")
	b.hx("/pantry/items/restore", undoForm(t, resp.body))
	if it := a.item(stock.Pantry, "Flour"); it.Quantity != 5 {
		t.Errorf("restored quantity %v", it.Quantity)
	}
}

func TestOpenedTodayFromTheList(t *testing.T) {
	a := newApp(t)
	a.stockClock()
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "quantity": {"3"}, "expires_on": {"2027-09-30"},
		"opened_days": {"4"}})
	resp := b.hx(itemPath(a.item(stock.Pantry, "Passata"))+"/open", nil)
	mustContain(t, resp.body, "Expires 10 Nov 2026 · opened 6 Nov 2026", "Expires 30 Sep 2027")
	if n := strings.Count(resp.body, `id="item-`); n != 2 {
		t.Errorf("%d rows", n)
	}
	if n := strings.Count(resp.body, ">Opened today<"); n != 1 {
		t.Errorf("%d rows can still be opened", n)
	}
}

func TestBuySomethingAgain(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "location": {"Pantry, top shelf"}, "opened_days": {"4"}})
	b.hx(itemPath(a.item(stock.Pantry, "Passata"))+"/remove", nil)
	mustContain(t, b.view("/pantry"), `<option value="Passata">`, `<option value="Pantry, top shelf">`)
	resp := b.get("/pantry/product?name=passata")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("known product: %s", resp.Status)
	}
	mustContain(t, resp.body, `id="product-fields"`, `name="location" value="Pantry, top shelf"`,
		`name="opened_days" value="4"`)
	if resp := b.get("/pantry/product?name=Pesto"); resp.StatusCode != http.StatusNoContent || resp.body != "" {
		t.Errorf("unknown product: %s %q", resp.Status, resp.body)
	}
	if resp := b.get("/medicines/product?name=Passata"); resp.StatusCode != http.StatusNoContent {
		t.Errorf("product of another section: %s", resp.Status)
	}
}

func TestExpiringBoxOnTheHomePage(t *testing.T) {
	a := newApp(t)
	a.stockClock()
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Medicines, url.Values{"name": {"Eye drops"}, "expires_on": {"2026-11-16"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Yogurt"}, "quantity": {"2"}, "unit": {"pots"}, "expires_on": {"2026-11-05"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}, "expires_on": {"2027-11-05"}})
	html := b.view("/")
	_, box, ok := strings.Cut(html, `aria-labelledby="expiring"`)
	box, _, _ = strings.Cut(box, "</section>")
	if !ok {
		t.Fatalf("no box:\n%s", html)
	}
	yogurt, drops := a.item(stock.Pantry, "Yogurt"), a.item(stock.Medicines, "Eye drops")
	mustContain(t, box, `href="/pantry#item-`+strconv.FormatInt(yogurt.ID, 10)+`"`, "· 2 · Pantry", "expired · 5 Nov 2026",
		`href="/medicines#item-`+strconv.FormatInt(drops.ID, 10)+`"`, "· 1 · Medicines", "expires in 10 days · 16 Nov 2026")
	if strings.Index(box, "Yogurt") > strings.Index(box, "Eye drops") {
		t.Error("expired item not first")
	}
	// "1 bottles": the unit is free text and doesn't agree with the number, so the box leaves it out.
	if strings.Contains(box, "pots") {
		t.Error("unit shown in the box")
	}
	if strings.Contains(box, "Rice") {
		t.Error("item expiring next year listed")
	}
	mustContain(t, html, `href="/medicines"`, `href="/pantry"`, "Recently changed")
}

func TestSuppliesInTheBox(t *testing.T) {
	a := newApp(t)
	a.stockClock()
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Supplies, url.Values{"name": {"Descaler"}, "expires_on": {"2026-11-09"}})
	_, box, ok := strings.Cut(b.view("/"), `aria-labelledby="expiring"`)
	box, _, _ = strings.Cut(box, "</section>")
	if !ok {
		t.Fatal("no box")
	}
	descaler := a.item(stock.Supplies, "Descaler")
	mustContain(t, box, `href="/supplies#item-`+strconv.FormatInt(descaler.ID, 10)+`"`, "Descaler", "· 1 · Supplies")
}

func TestSectionsOnTheHomePage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	_, nav, _ := strings.Cut(b.view("/"), `aria-label="Sections"`)
	nav, _, _ = strings.Cut(nav, "</nav>")
	for _, sec := range []string{"medicines", "pantry", "supplies"} {
		mustContain(t, nav, `href="/`+sec+`"`)
	}
	mustContain(t, nav, "Medicines", "Pantry", "Supplies")
	if resp := b.get("/supplies"); resp.StatusCode != http.StatusOK || !strings.Contains(resp.body, ">Supplies</h1>") {
		t.Errorf("supplies section: %s", resp.Status)
	}
}

func TestNoExpiringBox(t *testing.T) {
	a := newApp(t)
	a.stockClock()
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}, "expires_on": {"2027-11-05"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Salt"}})
	if html := b.view("/"); strings.Contains(html, `aria-labelledby="expiring"`) {
		t.Errorf("box shown:\n%s", html)
	}
}
