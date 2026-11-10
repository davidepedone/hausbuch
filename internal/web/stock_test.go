package web

import (
	"context"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"slices"
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
	anna.addItem(stock.Pantry, url.Values{"name": {"Passata & co"}, "unit": {"bottles"}, "category": {"Long-life"}, "location": {"Pantry"},
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
	b.addItem(stock.Pantry, url.Values{"name": {"Passata"}, "category": {"Long-life"}, "location": {"Pantry, top shelf"},
		"opened_days": {"4"}})
	b.hx(itemPath(a.item(stock.Pantry, "Passata"))+"/remove", nil)
	mustContain(t, b.view("/pantry"), `<option value="Passata">`, `<option value="Pantry, top shelf">`,
		`<option value="Long-life">`)
	resp := b.get("/pantry/product?name=passata")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("known product: %s", resp.Status)
	}
	mustContain(t, resp.body, `id="product-fields"`, `name="category" value="Long-life"`,
		`name="location" value="Pantry, top shelf"`,
		`name="opened_days" value="4"`)
	if resp := b.get("/pantry/product?name=Pesto"); resp.StatusCode != http.StatusNoContent || resp.body != "" {
		t.Errorf("unknown product: %s %q", resp.Status, resp.body)
	}
	if resp := b.get("/medicines/product?name=Passata"); resp.StatusCode != http.StatusNoContent {
		t.Errorf("product of another section: %s", resp.Status)
	}
}

func TestItemWithACategory(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Milk"}, "category": {"Fresh"}})
	milk := a.item(stock.Pantry, "Milk")
	if milk.Category != "Fresh" {
		t.Errorf("category %q", milk.Category)
	}
	_, group, ok := strings.Cut(b.view("/pantry"), `<h2 class="mb-1 font-display text-lg font-semibold">Fresh</h2>`)
	group, _, _ = strings.Cut(group, "</section>")
	if !ok || !strings.Contains(group, `id="item-`+strconv.FormatInt(milk.ID, 10)+`" data-entry data-chip="fresh"`) {
		t.Errorf("not in the Fresh group:\n%s", group)
	}
	edit := b.view(itemPath(milk))
	mustContain(t, edit, `name="category" value="Fresh"`, `<option value="Fresh">`)
	if r := b.post(itemPath(milk), url.Values{"name": {"Milk"}, "quantity": {"1"}, "category": {"Dairy"}}); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("save: %s", r.Status)
	}
	if got := a.item(stock.Pantry, "Milk").Category; got != "Dairy" {
		t.Errorf("after edit %q", got)
	}
}

func TestChipsOfASection(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Salt"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}, "category": {"Long-life"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Milk"}, "category": {"Fresh"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Yogurt"}, "category": {"fresh"}})
	html := b.view("/pantry")
	mustContain(t, html, `data-filter-all aria-pressed="true"`,
		`data-filter-chip="fresh" aria-pressed="false" class="chip">Fresh<`,
		`data-filter-chip="long-life" aria-pressed="false" class="chip">Long-life<`,
		`data-filter-chip="" aria-pressed="false" class="chip">Other<`, `data-filter-none hidden`,
		`<script src="/static/listfilter.js" defer>`)
	fresh, longLife, other := strings.Index(html, ">Fresh</h2>"), strings.Index(html, ">Long-life</h2>"), strings.Index(html, ">Other</h2>")
	if fresh < 0 || fresh > longLife || longLife > other {
		t.Errorf("groups out of order: %d %d %d", fresh, longLife, other)
	}
	if n := strings.Count(html, "<section data-group"); n != 3 {
		t.Errorf("%d groups", n)
	}
}

func TestNoCategories(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}})
	html := b.view("/pantry")
	mustContain(t, html, `data-entry data-chip=""`)
	if strings.Contains(html, "data-filter-chip") || strings.Contains(html, "data-filter-all") ||
		strings.Contains(html, "data-group") {
		t.Errorf("chips or groups without categories:\n%s", html)
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

func TestMoveFromTheEditPage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Dishwasher tablets"}, "category": {"Cleaning"}})
	it := a.item(stock.Pantry, "Dishwasher tablets")
	mustContain(t, b.view(itemPath(it)), `<select name="section"`, `<option value="pantry" selected>Pantry</option>`,
		`<option value="supplies">Supplies</option>`)
	r := b.post(itemPath(it), url.Values{"name": {"Dishwasher tablets"}, "quantity": {"1"}, "category": {"Cleaning"},
		"section": {"supplies"}})
	if want := "/supplies#item-" + strconv.FormatInt(it.ID, 10); r.StatusCode != http.StatusSeeOther || r.Header.Get("Location") != want {
		t.Fatalf("save: %s %s, want %s", r.Status, r.Header.Get("Location"), want)
	}
	if moved := a.item(stock.Supplies, "Dishwasher tablets"); moved.ID != it.ID || moved.Category != "Cleaning" {
		t.Errorf("moved %+v", moved)
	}
}

func TestMoveOntoTheSameProductFromTheEditPage(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Supplies, url.Values{"name": {"Descaler"}, "expires_on": {"2026-11-14"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Descaler"}, "quantity": {"2"}, "expires_on": {"2026-11-14"}})
	target, it := a.item(stock.Supplies, "Descaler"), a.item(stock.Pantry, "Descaler")
	r := b.post(itemPath(it), url.Values{"name": {"Descaler"}, "quantity": {"2"}, "expires_on": {"2026-11-14"},
		"section": {"supplies"}})
	if want := "/supplies#item-" + strconv.FormatInt(target.ID, 10); r.Header.Get("Location") != want {
		t.Errorf("redirect %s, want %s", r.Header.Get("Location"), want)
	}
	if got := a.item(stock.Supplies, "Descaler").Quantity; got != 3*stock.One {
		t.Errorf("quantity %v", got)
	}
}

func TestMoveToAnUnknownSection(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}})
	it := a.item(stock.Pantry, "Rice")
	r := b.post(itemPath(it), url.Values{"name": {"Rice"}, "quantity": {"4"}, "section": {"garage"}})
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("save: %s", r.Status)
	}
	mustContain(t, r.body, "Choose one of the sections.", `name="quantity" value="4"`)
	if got := a.item(stock.Pantry, "Rice").Quantity; got != stock.One {
		t.Errorf("changed anyway: %v", got)
	}
}

func TestMoveFromTheList(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Pasta"}, "quantity": {"3"}, "location": {"Cellar"}, "expires_on": {"2027-05-31"}})
	b.addItem(stock.Pantry, url.Values{"name": {"Rice"}, "location": {"Kitchen"}})
	it := a.item(stock.Pantry, "Pasta")
	mustContain(t, b.view("/pantry"), `href="`+itemPath(it)+`/move"`)
	// Default quantity, locations suggested.
	mustContain(t, b.view(itemPath(it)+"/move"), "Move Pasta", "3 in Cellar", `name="quantity" value="1"`,
		`<option value="Cellar">`, `<option value="Kitchen">`)
	r := b.post(itemPath(it)+"/move", url.Values{"quantity": {"2"}, "location": {"Kitchen"}})
	if r.StatusCode != http.StatusSeeOther {
		t.Fatalf("move: %s", r.Status)
	}
	items, err := a.stock.List(context.Background(), stock.Pantry)
	if err != nil {
		t.Fatal(err)
	}
	var moved stock.Item
	for _, i := range items {
		if i.Name == "Pasta" && i.Location == "Kitchen" {
			moved = i
		}
	}
	if moved.Quantity != 2*stock.One || a.stockQuantity(it) != stock.One {
		t.Fatalf("rows: %+v", items)
	}
	if want := "/pantry#item-" + strconv.FormatInt(moved.ID, 10); r.Header.Get("Location") != want {
		t.Errorf("redirect %s, want %s", r.Header.Get("Location"), want)
	}
	// Moving the rest onto it lands on the receiving row.
	r = b.post(itemPath(it)+"/move", url.Values{"quantity": {"1"}, "location": {"kitchen"}})
	if want := "/pantry#item-" + strconv.FormatInt(moved.ID, 10); r.Header.Get("Location") != want {
		t.Errorf("redirect %s, want %s", r.Header.Get("Location"), want)
	}
	if got := a.item(stock.Pantry, "Pasta"); got.ID != moved.ID || got.Quantity != 3*stock.One {
		t.Errorf("got %+v", got)
	}
}

// stockQuantity returns the quantity a row has now.
func (a *testApp) stockQuantity(it stock.Item) stock.Quantity {
	a.t.Helper()
	got, err := a.stock.Get(context.Background(), it.Section, it.ID)
	if err != nil {
		a.t.Fatal(err)
	}
	return got.Quantity
}

func TestDefaultQuantityOfAPart(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Flour"}, "quantity": {"0,5"}})
	mustContain(t, b.view(itemPath(a.item(stock.Pantry, "Flour"))+"/move"), `name="quantity" value="0.5"`)
}

func TestMoveRefused(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Pasta"}, "quantity": {"3"}, "location": {"Cellar"}})
	it := a.item(stock.Pantry, "Pasta")
	for _, c := range []struct {
		quantity, location, msg string
	}{
		{"4", "Kitchen", "The quantity must be above 0 and at most 3"},
		{"0", "Kitchen", "The quantity must be above 0 and at most 3"},
		{"1.25", "Kitchen", "The quantity must be above 0 and at most 3"},
		{"two", "Kitchen", "The quantity must be above 0 and at most 3"},
		{"1", "cellar", "It&#39;s already kept there."},
	} {
		r := b.post(itemPath(it)+"/move", url.Values{"quantity": {c.quantity}, "location": {c.location}})
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("%s to %s: %s", c.quantity, c.location, r.Status)
		}
		mustContain(t, r.body, c.msg, `name="quantity" value="`+c.quantity+`"`, `name="location" value="`+c.location+`"`)
	}
	if got := a.item(stock.Pantry, "Pasta"); got.Quantity != 3*stock.One || got.Location != "Cellar" {
		t.Errorf("changed anyway: %+v", got)
	}
}

// categoryOptions returns the options of a page's category suggestions.
func categoryOptions(t *testing.T, body string) []string {
	t.Helper()
	_, list, ok := strings.Cut(body, `<datalist id="categories">`)
	list, _, _ = strings.Cut(list, "</datalist>")
	if !ok {
		t.Fatalf("no category suggestions:\n%s", body)
	}
	var options []string
	for _, m := range regexp.MustCompile(`<option value="([^"]*)">`).FindAllStringSubmatch(list, -1) {
		options = append(options, html.UnescapeString(m[1]))
	}
	return options
}

func TestCommonCategoriesOnTheForms(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	want := []string{"Cleaning", "Laundry", "Paper & bags", "Personal care"}
	if got := categoryOptions(t, b.view("/supplies")); !slices.Equal(got, want) {
		t.Errorf("add form: %q, want %q", got, want)
	}
	b.addItem(stock.Medicines, url.Values{"name": {"Paracetamol"}})
	want = []string{"Cold & flu", "First aid", "Pain & fever", "Stomach"}
	if got := categoryOptions(t, b.view(itemPath(a.item(stock.Medicines, "Paracetamol")))); !slices.Equal(got, want) {
		t.Errorf("edit page: %q, want %q", got, want)
	}
}

// categoryChoices returns the choices under a page's category field.
func categoryChoices(body string) []string {
	var choices []string
	for _, m := range regexp.MustCompile(`<button type="button" data-suggestion class="chip">([^<]*)</button>`).FindAllStringSubmatch(body, -1) {
		choices = append(choices, html.UnescapeString(m[1]))
	}
	return choices
}

func TestCategoryChoicesOnTheForms(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	want := []string{"Cleaning", "Laundry", "Paper & bags", "Personal care"}
	if got := categoryChoices(b.view("/supplies")); !slices.Equal(got, want) {
		t.Errorf("add form: %q, want %q", got, want)
	}
	b.addItem(stock.Supplies, url.Values{"name": {"Soap"}, "category": {"Bathroom"}})
	want = []string{"Bathroom", "Cleaning", "Laundry", "Paper & bags", "Personal care"}
	if got := categoryChoices(b.view(itemPath(a.item(stock.Supplies, "Soap")))); !slices.Equal(got, want) {
		t.Errorf("edit page: %q, want %q", got, want)
	}
	e := b.boughtEntry(url.Values{"name": {"Detergent"}, "section": {"supplies"}})
	if got := categoryChoices(b.view(entryPath(e) + "/put-away")); !slices.Equal(got, want) {
		t.Errorf("put-away page: %q, want %q", got, want)
	}
}

func TestOnlySuggested(t *testing.T) {
	a := newApp(t)
	b := a.signedInAs("anna", "Anna")
	b.addItem(stock.Pantry, url.Values{"name": {"Milk"}, "category": {"Fresh"}})
	body := b.view("/pantry")
	if strings.Contains(body, `data-filter-chip="frozen"`) || strings.Contains(body, ">Frozen</h2>") {
		t.Error("an unused common category shows as a group or a chip")
	}
	mustContain(t, body, `data-filter-chip="fresh"`, `<option value="Frozen">`)
}
