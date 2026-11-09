package stock

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/davidepedone/hausbuch/internal/db"
	"github.com/davidepedone/hausbuch/internal/user"
)

var ctx = context.Background()

type fixture struct {
	t     *testing.T
	stock *Store
	now   time.Time
	anna  user.User
	marco user.User
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "hausbuch.db"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	users := user.NewStore(database)
	f := &fixture{t: t, now: time.Date(2026, 11, 6, 9, 0, 0, 0, time.Local)}
	f.stock = NewStore(database)
	f.stock.Now = func() time.Time { return f.now }
	for _, u := range []struct {
		to   *user.User
		name string
	}{{&f.anna, "anna"}, {&f.marco, "marco"}} {
		if *u.to, err = users.Add(ctx, u.name, u.name, "pw"); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func (f *fixture) add(section Section, v Values) int64 {
	f.t.Helper()
	id, err := f.stock.Add(ctx, section, f.anna.ID, v)
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) list(section Section) []Item {
	f.t.Helper()
	items, err := f.stock.List(ctx, section)
	if err != nil {
		f.t.Fatal(err)
	}
	return items
}

func (f *fixture) get(section Section, id int64) Item {
	f.t.Helper()
	it, err := f.stock.Get(ctx, section, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return it
}

func names(items []Item) []string {
	var list []string
	for _, it := range items {
		list = append(list, it.Name)
	}
	return list
}

func TestItemsListedByName(t *testing.T) {
	f := newFixture(t)
	f.add(Pantry, Values{Name: "Salt", Quantity: 1 * One})
	f.add(Pantry, Values{Name: "Rice", Quantity: 1 * One, ExpiresOn: day(2027, 1, 1)})
	late := f.add(Pantry, Values{Name: "Passata", Quantity: 1 * One, ExpiresOn: day(2027, 9, 30)})
	early := f.add(Pantry, Values{Name: "passata", Quantity: 1 * One, ExpiresOn: day(2027, 3, 31)})
	items := f.list(Pantry)
	if got := names(items); !slices.Equal(got, []string{"passata", "Passata", "Rice", "Salt"}) {
		t.Fatalf("order: %v", got)
	}
	if items[0].ID != early || items[1].ID != late {
		t.Errorf("earlier expiry not first: %d, %d", items[0].ID, items[1].ID)
	}
}

func TestRowsWithoutExpiryLast(t *testing.T) {
	f := newFixture(t)
	none := f.add(Pantry, Values{Name: "Flour", Quantity: 1 * One})
	dated := f.add(Pantry, Values{Name: "Flour", Quantity: 1 * One, ExpiresOn: day(2027, 1, 1)})
	if items := f.list(Pantry); items[0].ID != dated || items[1].ID != none {
		t.Errorf("order: %v", items)
	}
}

func TestSectionsAreSeparate(t *testing.T) {
	f := newFixture(t)
	id := f.add(Medicines, Values{Name: "Paracetamol", Quantity: 1 * One})
	if items := f.list(Pantry); len(items) != 0 {
		t.Errorf("pantry: %v", items)
	}
	if _, err := f.stock.Get(ctx, Pantry, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("medicine got from the pantry: %v", err)
	}
	if err := f.stock.Increase(ctx, Pantry, id, f.anna.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("medicine changed through the pantry: %v", err)
	}
}

func TestSuppliesKeptApartFromFood(t *testing.T) {
	f := newFixture(t)
	f.add(Supplies, Values{Name: "Dishwasher tablets", Quantity: 1 * One})
	if items := f.list(Supplies); len(items) != 1 || items[0].Name != "Dishwasher tablets" {
		t.Errorf("supplies: %v", items)
	}
	if items := f.list(Pantry); len(items) != 0 {
		t.Errorf("pantry: %v", items)
	}
}

func TestNewItem(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: " Passata ", Quantity: 3 * One, Unit: "bottles", Location: "Pantry, top shelf",
		ExpiresOn: day(2027, 9, 30)})
	it := f.get(Pantry, id)
	want := Values{Name: "Passata", Quantity: 3 * One, Unit: "bottles", Location: "Pantry, top shelf",
		ExpiresOn: day(2027, 9, 30)}
	if it.Values != want || it.CreatedBy != "anna" || it.ChangedBy != "anna" || !it.CreatedAt.Equal(f.now) {
		t.Errorf("got %+v", it)
	}
}

func TestItemWithoutExpiryDate(t *testing.T) {
	f := newFixture(t)
	if it := f.get(Pantry, f.add(Pantry, Values{Name: "Salt", Quantity: 1 * One})); !it.ExpiresOn.IsZero() || it.State != Fine {
		t.Errorf("got %+v", it)
	}
}

func TestMissingNameOrQuantity(t *testing.T) {
	f := newFixture(t)
	for _, c := range []struct {
		v   Values
		err error
	}{
		{Values{Name: "  ", Quantity: 1 * One}, ErrEmptyName},
		{Values{Name: "Rice", Quantity: 0 * One}, ErrQuantity},
		{Values{Name: "Rice", Quantity: 1 * One, OpenedDays: -1}, ErrOpenedDays},
	} {
		if _, err := f.stock.Add(ctx, Pantry, f.anna.ID, c.v); !errors.Is(err, c.err) {
			t.Errorf("%+v: %v, want %v", c.v, err, c.err)
		}
	}
	if items := f.list(Pantry); len(items) != 0 {
		t.Errorf("saved: %v", items)
	}
}

func TestSameProductDifferentExpiry(t *testing.T) {
	f := newFixture(t)
	f.add(Medicines, Values{Name: "Paracetamol", Quantity: 1 * One, ExpiresOn: day(2027, 3, 31)})
	f.add(Medicines, Values{Name: "Paracetamol", Quantity: 1 * One, ExpiresOn: day(2026, 11, 30)})
	if items := f.list(Medicines); len(items) != 2 {
		t.Errorf("rows: %v", items)
	}
}

func TestSameProductSameExpiry(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Passata", Quantity: 2 * One, ExpiresOn: day(2027, 9, 30)})
	f.now = f.now.Add(time.Hour)
	if _, err := f.stock.Add(ctx, Pantry, f.marco.ID, Values{Name: "passata", Quantity: 1 * One, ExpiresOn: day(2027, 9, 30)}); err != nil {
		t.Fatal(err)
	}
	items := f.list(Pantry)
	if len(items) != 1 || items[0].ID != id || items[0].Quantity != 3*One || items[0].Name != "Passata" ||
		items[0].ChangedBy != "marco" {
		t.Errorf("got %+v", items)
	}
}

func TestSameProductWithoutExpiryMerges(t *testing.T) {
	f := newFixture(t)
	f.add(Pantry, Values{Name: "Salt", Quantity: 1 * One})
	f.add(Pantry, Values{Name: "Salt", Quantity: 1 * One})
	if items := f.list(Pantry); len(items) != 1 || items[0].Quantity != 2*One {
		t.Errorf("got %+v", items)
	}
}

func TestOpenedRowStaysApart(t *testing.T) {
	f := newFixture(t)
	opened := f.add(Pantry, Values{Name: "Passata", Quantity: 1 * One, ExpiresOn: day(2027, 9, 30), OpenedDays: 4})
	if err := f.stock.Open(ctx, Pantry, opened, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	f.add(Pantry, Values{Name: "Passata", Quantity: 1 * One, ExpiresOn: day(2027, 9, 30)})
	items := f.list(Pantry)
	if len(items) != 2 || items[0].Quantity != 1*One || items[1].Quantity != 1*One {
		t.Fatalf("got %+v", items)
	}
	if f.get(Pantry, opened).Quantity != 1*One {
		t.Error("added to the opened row")
	}
}

func TestChangeTheLocation(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Passata", Quantity: 3 * One, Location: "Pantry"})
	f.now = f.now.Add(time.Hour)
	if err := f.stock.Update(ctx, Pantry, id, f.marco.ID, Values{Name: "Passata", Quantity: 3 * One, Location: "Cellar"}); err != nil {
		t.Fatal(err)
	}
	it := f.get(Pantry, id)
	if it.Location != "Cellar" || it.ChangedBy != "marco" || !it.ChangedAt.Equal(f.now) || it.CreatedBy != "anna" {
		t.Errorf("got %+v", it)
	}
	if err := f.stock.Update(ctx, Pantry, id, f.marco.ID, Values{Name: "", Quantity: 3 * One}); !errors.Is(err, ErrEmptyName) {
		t.Errorf("empty name: %v", err)
	}
}

func TestUseOne(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Passata", Quantity: 3 * One})
	removed, err := f.stock.Decrease(ctx, Pantry, id, f.anna.ID)
	if err != nil || removed != nil {
		t.Fatal(removed, err)
	}
	if q := f.get(Pantry, id).Quantity; q != 2*One {
		t.Errorf("quantity %v", q)
	}
	if err := f.stock.Increase(ctx, Pantry, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	if q := f.get(Pantry, id).Quantity; q != 3*One {
		t.Errorf("after increase: %v", q)
	}
}

func TestUseTheLastOne(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Passata", Quantity: 1 * One, Location: "Pantry"})
	removed, err := f.stock.Decrease(ctx, Pantry, id, f.anna.ID)
	if err != nil {
		t.Fatal(err)
	}
	if removed == nil || removed.ID != id || removed.Location != "Pantry" {
		t.Errorf("removed: %+v", removed)
	}
	if items := f.list(Pantry); len(items) != 0 {
		t.Errorf("still there: %v", items)
	}
}

func TestTwoUsersAtOnce(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Passata", Quantity: 3 * One})
	var wg sync.WaitGroup
	for _, u := range []int64{f.anna.ID, f.marco.ID} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.stock.Decrease(ctx, Pantry, id, u); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if q := f.get(Pantry, id).Quantity; q != 1*One {
		t.Errorf("quantity %v", q)
	}
}

func TestThrowSomethingAway(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Yogurt", Quantity: 2 * One, ExpiresOn: day(2026, 11, 1)})
	it, err := f.stock.Remove(ctx, Pantry, id)
	if err != nil || it.Name != "Yogurt" || it.Quantity != 2*One {
		t.Fatal(it, err)
	}
	if items := f.list(Pantry); len(items) != 0 {
		t.Errorf("still there: %v", items)
	}
	if _, err := f.stock.Remove(ctx, Pantry, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("second removal: %v", err)
	}
}

func TestUndo(t *testing.T) {
	f := newFixture(t)
	v := Values{Name: "Passata", Quantity: 2 * One, Unit: "bottles", Location: "Pantry", ExpiresOn: day(2027, 9, 30),
		OpenedOn: day(2026, 11, 5), OpenedDays: 4}
	id := f.add(Pantry, v)
	it, err := f.stock.Remove(ctx, Pantry, id)
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Minute)
	if _, err := f.stock.Restore(ctx, Pantry, f.marco.ID, it); err != nil {
		t.Fatal(err)
	}
	items := f.list(Pantry)
	if len(items) != 1 || items[0].Values != v || items[0].CreatedBy != "anna" || !items[0].CreatedAt.Equal(it.CreatedAt) {
		t.Errorf("got %+v", items)
	}
}

func TestOpenASingleItem(t *testing.T) {
	f := newFixture(t)
	id := f.add(Medicines, Values{Name: "Eye drops", Quantity: 1 * One, OpenedDays: 28})
	if err := f.stock.Open(ctx, Medicines, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	if it := f.get(Medicines, id); !it.OpenedOn.Equal(day(2026, 11, 6)) || it.Quantity != 1*One {
		t.Errorf("got %+v", it)
	}
	if items := f.list(Medicines); len(items) != 1 {
		t.Errorf("rows: %v", items)
	}
}

func TestOpenOneOfSeveral(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Passata", Quantity: 3 * One, ExpiresOn: day(2027, 9, 30), OpenedDays: 4})
	if err := f.stock.Open(ctx, Pantry, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	items := f.list(Pantry)
	if len(items) != 2 {
		t.Fatalf("rows: %+v", items)
	}
	// The opened bottle expires first, so it comes first.
	opened, rest := items[0], items[1]
	if rest.ID != id || rest.Quantity != 2*One || !rest.OpenedOn.IsZero() {
		t.Errorf("rest: %+v", rest)
	}
	if opened.Quantity != 1*One || !opened.OpenedOn.Equal(day(2026, 11, 6)) || !opened.ExpiresOn.Equal(day(2027, 9, 30)) ||
		opened.OpenedDays != 4 {
		t.Errorf("opened: %+v", opened)
	}
}

func TestEffectiveExpiry(t *testing.T) {
	for _, c := range []struct {
		name string
		v    Values
		want time.Time
	}{
		{"opening comes first", Values{ExpiresOn: day(2027, 9, 30), OpenedOn: day(2026, 11, 6), OpenedDays: 4}, day(2026, 11, 10)},
		{"printed date comes first", Values{ExpiresOn: day(2026, 11, 15), OpenedOn: day(2026, 11, 6), OpenedDays: 28}, day(2026, 11, 15)},
		{"no printed date", Values{OpenedOn: day(2026, 11, 6), OpenedDays: 5}, day(2026, 11, 11)},
		{"not opened", Values{ExpiresOn: day(2027, 9, 30), OpenedDays: 4}, day(2027, 9, 30)},
		{"opened, no shelf life", Values{ExpiresOn: day(2027, 9, 30), OpenedOn: day(2026, 11, 6)}, day(2027, 9, 30)},
		{"nothing", Values{}, time.Time{}},
	} {
		if got := c.v.Expiry(); !got.Equal(c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestStates(t *testing.T) {
	f := newFixture(t)
	for _, c := range []struct {
		name    string
		section Section
		v       Values
		want    State
	}{
		{"last good day", Pantry, Values{ExpiresOn: day(2026, 11, 6)}, Expiring},
		{"expired", Pantry, Values{ExpiresOn: day(2026, 11, 5)}, Expired},
		{"within the lead time", Pantry, Values{ExpiresOn: day(2026, 11, 13)}, Expiring},
		{"beyond the lead time", Pantry, Values{ExpiresOn: day(2026, 11, 14)}, Fine},
		{"nothing to expire", Pantry, Values{OpenedDays: 4}, Fine},
		{"default medicines lead time", Medicines, Values{ExpiresOn: day(2026, 11, 26)}, Expiring},
		{"default pantry lead time", Pantry, Values{ExpiresOn: day(2026, 11, 26)}, Fine},
		{"default supplies lead time", Supplies, Values{ExpiresOn: day(2026, 11, 13)}, Expiring},
		{"beyond the default supplies lead time", Supplies, Values{ExpiresOn: day(2026, 11, 14)}, Fine},
	} {
		c.v.Name, c.v.Quantity = c.name, One
		if got := f.get(c.section, f.add(c.section, c.v)).State; got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestConfiguredLeadTime(t *testing.T) {
	f := newFixture(t)
	f.stock.Lead[Pantry] = 3
	if it := f.get(Pantry, f.add(Pantry, Values{Name: "Milk", Quantity: 1 * One, ExpiresOn: day(2026, 11, 11)})); it.State != Fine {
		t.Errorf("state %v", it.State)
	}
}

func TestDaysFollowTheTimeZone(t *testing.T) {
	f := newFixture(t)
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatal(err)
	}
	defer func(l *time.Location) { time.Local = l }(time.Local)
	time.Local = rome
	id := f.add(Pantry, Values{Name: "Milk", Quantity: 1 * One, ExpiresOn: day(2026, 11, 6)})
	// 23:30 UTC on the 6th is already the 7th in Rome.
	f.now = time.Date(2026, 11, 6, 23, 30, 0, 0, time.UTC)
	if it := f.get(Pantry, id); it.State != Expired {
		t.Errorf("state %v", it.State)
	}
	f.now = time.Date(2026, 11, 6, 22, 30, 0, 0, time.UTC)
	if it := f.get(Pantry, id); it.State != Expiring {
		t.Errorf("before midnight in Rome: %v", it.State)
	}
}

func TestWarnings(t *testing.T) {
	f := newFixture(t)
	f.add(Medicines, Values{Name: "Eye drops", Quantity: 1 * One, ExpiresOn: day(2026, 11, 16)})
	f.add(Pantry, Values{Name: "Yogurt", Quantity: 1 * One, ExpiresOn: day(2026, 11, 5)})
	f.add(Pantry, Values{Name: "Milk", Quantity: 1 * One, ExpiresOn: day(2026, 11, 8)})
	f.add(Pantry, Values{Name: "Rice", Quantity: 1 * One, ExpiresOn: day(2027, 11, 5)})
	f.add(Pantry, Values{Name: "Salt", Quantity: 1 * One})
	items, err := f.stock.Warnings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(items); !slices.Equal(got, []string{"Yogurt", "Milk", "Eye drops"}) {
		t.Errorf("warnings: %v", got)
	}
}

func TestBuySomethingAgain(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Passata", Quantity: 1 * One, Location: "Pantry, top shelf", OpenedDays: 4})
	if _, err := f.stock.Remove(ctx, Pantry, id); err != nil {
		t.Fatal(err)
	}
	p, found, err := f.stock.Product(ctx, Pantry, "passata")
	if err != nil || !found || p != (Product{Name: "Passata", Location: "Pantry, top shelf", OpenedDays: 4}) {
		t.Errorf("product %+v %v %v", p, found, err)
	}
	if got, _ := f.stock.Names(ctx, Pantry); !slices.Equal(got, []string{"Passata"}) {
		t.Errorf("names %v", got)
	}
	if got, _ := f.stock.Locations(ctx, Pantry); !slices.Equal(got, []string{"Pantry, top shelf"}) {
		t.Errorf("locations %v", got)
	}
}

func TestProductKeepsTheLastValues(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Passata", Quantity: 1 * One, Location: "Pantry"})
	if err := f.stock.Update(ctx, Pantry, id, f.anna.ID, Values{Name: "Passata", Quantity: 1 * One, Location: "Cellar", OpenedDays: 4}); err != nil {
		t.Fatal(err)
	}
	if p, _, _ := f.stock.Product(ctx, Pantry, "Passata"); p.Location != "Cellar" || p.OpenedDays != 4 {
		t.Errorf("product %+v", p)
	}
}

func TestSuggestionsStayInTheirSection(t *testing.T) {
	f := newFixture(t)
	f.add(Medicines, Values{Name: "Paracetamol", Quantity: 1 * One, Location: "Bathroom"})
	if _, found, _ := f.stock.Product(ctx, Pantry, "Paracetamol"); found {
		t.Error("found in the pantry")
	}
	if got, _ := f.stock.Names(ctx, Pantry); len(got) != 0 {
		t.Errorf("names %v", got)
	}
	if got, _ := f.stock.Locations(ctx, Pantry); len(got) != 0 {
		t.Errorf("locations %v", got)
	}
}

func TestParseQuantity(t *testing.T) {
	for text, want := range map[string]Quantity{"1,5": 15, "1.5": 15, "2": 20, "2.0": 20, " 3 ": 30, "0.5": 5, "12,3": 123} {
		if got, err := ParseQuantity(text); err != nil || got != want {
			t.Errorf("%q: %d, %v, want %d", text, got, err, want)
		}
	}
	for _, text := range []string{"", "0", "0.0", "-1", "1.25", "1,5,3", "abc", "1.x", "1.-"} {
		if got, err := ParseQuantity(text); !errors.Is(err, ErrQuantity) {
			t.Errorf("%q: %d, %v, want refused", text, got, err)
		}
	}
}

func TestQuantityString(t *testing.T) {
	for q, want := range map[Quantity]string{20: "2", 15: "1.5", 5: "0.5", 10: "1", 123: "12.3"} {
		if got := q.String(); got != want {
			t.Errorf("%d: %q, want %q", int(q), got, want)
		}
	}
}

func TestQuantityWithADecimal(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Flour", Quantity: 15, Unit: "kg"})
	f.add(Pantry, Values{Name: "Flour", Quantity: 5, Unit: "kg"})
	if q := f.get(Pantry, id).Quantity; q != 2*One {
		t.Errorf("merged quantity %v", q)
	}
}

func TestUseOneOfADecimalQuantity(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Flour", Quantity: 25})
	if removed, err := f.stock.Decrease(ctx, Pantry, id, f.anna.ID); err != nil || removed != nil {
		t.Fatal(removed, err)
	}
	if q := f.get(Pantry, id).Quantity; q != 15 {
		t.Errorf("quantity %v", q)
	}
}

func TestUseTheLastPart(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Flour", Quantity: 5})
	removed, err := f.stock.Decrease(ctx, Pantry, id, f.anna.ID)
	if err != nil || removed == nil || removed.Quantity != 5 {
		t.Fatal(removed, err)
	}
	if items := f.list(Pantry); len(items) != 0 {
		t.Errorf("still there: %v", items)
	}
}

func TestOpenOneOfADecimalQuantity(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Milk", Quantity: 25})
	if err := f.stock.Open(ctx, Pantry, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	items := f.list(Pantry)
	if len(items) != 2 {
		t.Fatalf("rows: %+v", items)
	}
	for _, it := range items {
		if it.ID == id && (it.Quantity != 15 || !it.OpenedOn.IsZero()) ||
			it.ID != id && (it.Quantity != One || !it.OpenedOn.Equal(day(2026, 11, 6))) {
			t.Errorf("row %+v", it)
		}
	}
}

func TestOpenAPart(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Flour", Quantity: 5})
	if err := f.stock.Open(ctx, Pantry, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	if it := f.get(Pantry, id); it.Quantity != 5 || !it.OpenedOn.Equal(day(2026, 11, 6)) {
		t.Errorf("got %+v", it)
	}
	if items := f.list(Pantry); len(items) != 1 {
		t.Errorf("rows: %v", items)
	}
}
