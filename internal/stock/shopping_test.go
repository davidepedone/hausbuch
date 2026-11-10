package stock

import (
	"errors"
	"slices"
	"testing"
)

func (f *fixture) addEntry(e Entry) {
	f.t.Helper()
	if err := f.stock.AddEntry(ctx, f.anna.ID, e); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) entries() []Entry {
	f.t.Helper()
	list, err := f.stock.Entries(ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	return list
}

// entry returns the only entry named name that is bought, or not, as given.
func (f *fixture) entry(name string, bought bool) Entry {
	f.t.Helper()
	var found []Entry
	for _, e := range f.entries() {
		if nameKey(e.Name) == nameKey(name) && e.Bought == bought {
			found = append(found, e)
		}
	}
	if len(found) != 1 {
		f.t.Fatalf("%d %q entries with bought %v, want 1: %+v", len(found), name, bought, f.entries())
	}
	return found[0]
}

func (f *fixture) tick(name string, bought bool) {
	f.t.Helper()
	if err := f.stock.Tick(ctx, f.entry(name, !bought).ID, bought); err != nil {
		f.t.Fatal(err)
	}
}

func entryNames(list []Entry) []string {
	var names []string
	for _, e := range list {
		names = append(names, e.Name)
	}
	return names
}

func TestGroupedList(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Bread"})
	f.addEntry(Entry{Name: "Rice", Section: Pantry})
	f.addEntry(Entry{Name: "Passata", Section: Pantry})
	f.addEntry(Entry{Name: "Paracetamol", Section: Medicines})
	toBuy, bought := GroupEntries(f.entries())
	var got [][]string
	var labels []string
	for _, g := range toBuy {
		labels = append(labels, g.Label())
		got = append(got, entryNames(g.Entries))
	}
	if !slices.Equal(labels, []string{"Medicines", "Pantry", "Other"}) {
		t.Errorf("groups %v", labels)
	}
	want := [][]string{{"Paracetamol"}, {"Passata", "Rice"}, {"Bread"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("entries %v, want %v", got, want)
	}
	if len(bought) != 0 {
		t.Errorf("bought %v", bought)
	}
}

func TestBoughtEntriesApart(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Rice", Section: Pantry})
	f.addEntry(Entry{Name: "Passata", Section: Pantry})
	f.tick("Rice", true)
	toBuy, bought := GroupEntries(f.entries())
	if len(toBuy) != 1 || !slices.Equal(entryNames(toBuy[0].Entries), []string{"Passata"}) {
		t.Errorf("to buy %+v", toBuy)
	}
	if !slices.Equal(entryNames(bought), []string{"Rice"}) {
		t.Errorf("bought %+v", bought)
	}
}

func TestEntryForTheStock(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: " Passata ", Quantity: 30, Unit: "bottles", Section: Pantry})
	got := f.entry("Passata", false)
	want := Entry{ID: got.ID, Name: "Passata", Quantity: 30, Unit: "bottles", Section: Pantry}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestEntryWithoutASection(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Bread"})
	if got := f.entry("Bread", false); got.Section != "" || got.Quantity != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestRefusedEntry(t *testing.T) {
	f := newFixture(t)
	for _, c := range []struct {
		e    Entry
		want error
	}{
		{Entry{Name: "  "}, ErrEmptyName},
		{Entry{Name: "Rice", Quantity: -10}, ErrQuantity},
		{Entry{Name: "Rice", Section: "garage"}, ErrSection},
	} {
		if err := f.stock.AddEntry(ctx, f.anna.ID, c.e); !errors.Is(err, c.want) {
			t.Errorf("%+v: got %v, want %v", c.e, err, c.want)
		}
	}
	if list := f.entries(); len(list) != 0 {
		t.Errorf("list %+v", list)
	}
}

func TestAddedTwice(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Rice", Quantity: 10, Section: Pantry})
	f.addEntry(Entry{Name: "rice", Quantity: 20, Section: Pantry})
	if got := f.entry("Rice", false); got.Quantity != 30 || got.Name != "Rice" {
		t.Errorf("got %+v", got)
	}
}

func TestAddedAgainWithoutAQuantity(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Rice", Quantity: 10, Unit: "kg", Section: Pantry})
	f.addEntry(Entry{Name: "Rice", Section: Pantry})
	if got := f.entry("Rice", false); got.Quantity != 10 || got.Unit != "kg" {
		t.Errorf("got %+v", got)
	}
}

func TestQuantityGivenTheSecondTime(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Rice", Section: Pantry})
	f.addEntry(Entry{Name: "Rice", Quantity: 20, Unit: "kg", Section: Pantry})
	if got := f.entry("Rice", false); got.Quantity != 20 || got.Unit != "kg" {
		t.Errorf("got %+v", got)
	}
}

func TestAlreadyBought(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Rice", Section: Pantry})
	f.tick("Rice", true)
	f.addEntry(Entry{Name: "Rice", Section: Pantry})
	f.entry("Rice", false)
	f.entry("Rice", true)
}

func TestDifferentSection(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Soap", Section: Supplies})
	f.addEntry(Entry{Name: "Soap"})
	toBuy, _ := GroupEntries(f.entries())
	if len(toBuy) != 2 || toBuy[0].Section != Supplies || toBuy[1].Section != "" {
		t.Errorf("groups %+v", toBuy)
	}
}

func TestNoLongerNeeded(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Rice"})
	if err := f.stock.RemoveEntry(ctx, f.entry("Rice", false).ID); err != nil {
		t.Fatal(err)
	}
	if list := f.entries(); len(list) != 0 {
		t.Errorf("list %+v", list)
	}
	if err := f.stock.RemoveEntry(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown entry: %v", err)
	}
}

func TestClear(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Bread"})
	f.addEntry(Entry{Name: "Passata", Section: Pantry})
	f.addEntry(Entry{Name: "Milk"})
	f.tick("Bread", true)
	f.tick("Passata", true)
	if err := f.stock.ClearBought(ctx); err != nil {
		t.Fatal(err)
	}
	_, bought := GroupEntries(f.entries())
	if !slices.Equal(entryNames(bought), []string{"Passata"}) {
		t.Errorf("bought %+v", bought)
	}
	f.entry("Milk", false)
}

func TestBoughtThereAgain(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Butter", Quantity: One})
	f.addEntry(Entry{Name: "Butter", Section: Pantry, Shop: "Migros"})
	if err := f.stock.RemoveEntry(ctx, f.entry("Butter", false).ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.stock.AddRowToList(ctx, Pantry, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	if got := f.entry("butter", false); got.Shop != "Migros" {
		t.Errorf("got %+v", got)
	}
}

func TestShopChanged(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Butter", Shop: "Migros"})
	f.addEntry(Entry{Name: "Butter", Shop: "Coop"})
	if got := f.entry("Butter", false); got.Shop != "Coop" {
		t.Errorf("got %+v", got)
	}
	if _, shop, err := f.stock.ListProduct(ctx, "butter"); err != nil || shop != "Coop" {
		t.Errorf("remembered %q, %v", shop, err)
	}
}

func TestTick(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Rice", Quantity: 10, Section: Pantry})
	f.tick("Rice", true)
	if got := f.entry("Rice", true); got.Quantity != 10 {
		t.Errorf("got %+v", got)
	}
	f.tick("Rice", false)
	f.entry("Rice", false)
	if err := f.stock.Tick(ctx, 99, true); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown entry: %v", err)
	}
}

func TestTickedTwice(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Rice", Quantity: 20, Section: Pantry})
	f.tick("Rice", true)
	f.addEntry(Entry{Name: "Rice", Quantity: 10, Section: Pantry})
	f.tick("Rice", true)
	if list := f.entries(); len(list) != 1 || !list[0].Bought || list[0].Quantity != 30 {
		t.Errorf("list %+v", list)
	}
}

func TestPutAwaySaved(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Passata", Quantity: 30, Section: Pantry})
	f.tick("Passata", true)
	v := Values{Name: "Passata", Quantity: 30, ExpiresOn: day(2027, 9, 30)}
	id, err := f.stock.PutAway(ctx, f.entry("Passata", true).ID, f.marco.ID, v)
	if err != nil {
		t.Fatal(err)
	}
	if it := f.get(Pantry, id); it.Quantity != 30 || !it.ExpiresOn.Equal(day(2027, 9, 30)) || it.CreatedBy != "marco" {
		t.Errorf("row %+v", it)
	}
	if list := f.entries(); len(list) != 0 {
		t.Errorf("list %+v", list)
	}
}

func TestPutAwayAddedToAnExistingRow(t *testing.T) {
	f := newFixture(t)
	v := Values{Name: "Passata", Quantity: 20, Location: "Pantry, top shelf", ExpiresOn: day(2027, 9, 30)}
	row := f.add(Pantry, v)
	f.addEntry(Entry{Name: "Passata", Section: Pantry})
	f.tick("Passata", true)
	v.Quantity = 30
	id, err := f.stock.PutAway(ctx, f.entry("Passata", true).ID, f.anna.ID, v)
	if err != nil {
		t.Fatal(err)
	}
	if id != row || f.get(Pantry, row).Quantity != 50 || len(f.list(Pantry)) != 1 {
		t.Errorf("rows %+v", f.list(Pantry))
	}
}

func TestPutAwayRefused(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Passata", Section: Pantry})
	f.addEntry(Entry{Name: "Rice", Section: Pantry})
	f.addEntry(Entry{Name: "Bread"})
	f.tick("Passata", true)
	f.tick("Bread", true)
	for _, c := range []struct {
		name   string
		bought bool
		v      Values
		want   error
	}{
		{"Passata", true, Values{Name: " ", Quantity: One}, ErrEmptyName},
		{"Rice", false, Values{Name: "Rice", Quantity: One}, ErrNotFound},
		{"Bread", true, Values{Name: "Bread", Quantity: One}, ErrNotFound},
	} {
		if _, err := f.stock.PutAway(ctx, f.entry(c.name, c.bought).ID, f.anna.ID, c.v); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	if _, err := f.stock.PutAway(ctx, 99, f.anna.ID, Values{Name: "X", Quantity: One}); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown entry: %v", err)
	}
	if rows := f.list(Pantry); len(rows) != 0 {
		t.Errorf("rows %+v", rows)
	}
	if len(f.entries()) != 3 {
		t.Errorf("list %+v", f.entries())
	}
}

func TestRunningLow(t *testing.T) {
	f := newFixture(t)
	id := f.add(Medicines, Values{Name: "Paracetamol", Quantity: One, Unit: "packs"})
	it, err := f.stock.AddRowToList(ctx, Medicines, id, f.anna.ID)
	if err != nil || it.Name != "Paracetamol" {
		t.Fatal(it, err)
	}
	got := f.entry("Paracetamol", false)
	if got.Section != Medicines || got.Unit != "packs" || got.Quantity != 0 {
		t.Errorf("got %+v", got)
	}
	if f.get(Medicines, id).Quantity != One {
		t.Error("the row changed")
	}
	if _, err := f.stock.AddRowToList(ctx, Pantry, id, f.anna.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("row of another section: %v", err)
	}
}

func TestAlreadyOnTheList(t *testing.T) {
	f := newFixture(t)
	id := f.add(Pantry, Values{Name: "Rice", Quantity: One})
	f.addEntry(Entry{Name: "Rice", Quantity: 20, Section: Pantry})
	if _, err := f.stock.AddRowToList(ctx, Pantry, id, f.anna.ID); err != nil {
		t.Fatal(err)
	}
	if got := f.entry("Rice", false); got.Quantity != 20 {
		t.Errorf("got %+v", got)
	}
}

func TestKnownProduct(t *testing.T) {
	f := newFixture(t)
	f.add(Pantry, Values{Name: "Passata", Quantity: One})
	if section, _, err := f.stock.ListProduct(ctx, "passata"); err != nil || section != Pantry {
		t.Errorf("got %q, %v", section, err)
	}
}

func TestNameUsedInTwoSections(t *testing.T) {
	f := newFixture(t)
	f.add(Pantry, Values{Name: "Soap", Quantity: One})
	f.add(Supplies, Values{Name: "Soap", Quantity: One})
	if section, _, err := f.stock.ListProduct(ctx, "Soap"); err != nil || section != "" {
		t.Errorf("got %q, %v", section, err)
	}
	names, err := f.stock.ListNames(ctx)
	if err != nil || !slices.Equal(names, []string{"Soap"}) {
		t.Errorf("names %v, %v", names, err)
	}
}

func TestListNames(t *testing.T) {
	f := newFixture(t)
	f.add(Pantry, Values{Name: "Rice", Quantity: One})
	f.addEntry(Entry{Name: "Bread", Shop: "Bakery"})
	names, err := f.stock.ListNames(ctx)
	if err != nil || !slices.Equal(names, []string{"Bread", "Rice"}) {
		t.Errorf("names %v, %v", names, err)
	}
}

func TestShopSuggestions(t *testing.T) {
	f := newFixture(t)
	f.addEntry(Entry{Name: "Butter", Shop: "Migros"})
	f.addEntry(Entry{Name: "Paracetamol", Shop: "Pharmacy"})
	f.addEntry(Entry{Name: "Milk", Shop: "migros"})
	shops, err := f.stock.Shops(ctx)
	if err != nil || !slices.Equal(shops, []string{"Migros", "Pharmacy"}) {
		t.Errorf("shops %v, %v", shops, err)
	}
}

func TestShopChips(t *testing.T) {
	chips := ShopChips([]Entry{{Shop: "Pharmacy"}, {Shop: "Migros"}, {Shop: "migros"}, {}})
	want := []Chip{{"migros", "Migros"}, {"pharmacy", "Pharmacy"}}
	if !slices.Equal(chips, want) {
		t.Errorf("got %+v, want %+v", chips, want)
	}
}
