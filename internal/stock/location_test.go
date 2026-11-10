package stock

import (
	"slices"
	"testing"
)

func TestLocationsAcrossSections(t *testing.T) {
	f := newFixture(t)
	f.add(Pantry, Values{Name: "Passata", Quantity: One, Location: "Cellar"})
	f.add(Pantry, Values{Name: "Beans", Quantity: One, Location: "cellar", ExpiresOn: day(2027, 1, 31)})
	f.add(Pantry, Values{Name: "Milk", Quantity: One, Location: "Fridge"})
	f.add(Pantry, Values{Name: "Rice", Quantity: One})
	f.add(Supplies, Values{Name: "Detergent", Quantity: One, Location: "CELLAR"})
	places, err := f.stock.Places(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := []Place{{"CELLAR", 3}, {"Fridge", 1}}; !slices.Equal(places, want) {
		t.Errorf("places %v, want %v", places, want)
	}

	rows, err := f.stock.InLocation(ctx, "Cellar")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, it := range rows {
		names = append(names, string(it.Section)+"/"+it.Name)
	}
	if want := []string{"pantry/Beans", "pantry/Passata", "supplies/Detergent"}; !slices.Equal(names, want) {
		t.Errorf("in the cellar %v, want %v", names, want)
	}
}

func TestNoLocations(t *testing.T) {
	f := newFixture(t)
	f.add(Pantry, Values{Name: "Rice", Quantity: One})
	if places, err := f.stock.Places(ctx); err != nil || len(places) != 0 {
		t.Errorf("places %v, %v", places, err)
	}
}

func TestALocationLeftEmpty(t *testing.T) {
	f := newFixture(t)
	id := f.add(Supplies, Values{Name: "Bike pump", Quantity: One, Location: "Garage"})
	if _, err := f.stock.Remove(ctx, Supplies, id); err != nil {
		t.Fatal(err)
	}
	places, _ := f.stock.Places(ctx)
	rows, err := f.stock.InLocation(ctx, "Garage")
	if len(places) != 0 || len(rows) != 0 || err != nil {
		t.Errorf("places %v, rows %v, %v", places, rows, err)
	}
}

func TestLocationsFromTheWholeHome(t *testing.T) {
	f := newFixture(t)
	if got, err := f.stock.Locations(ctx); err != nil || len(got) != 0 {
		t.Errorf("nothing used yet: %v, %v", got, err)
	}
	f.add(Supplies, Values{Name: "Detergent", Quantity: One, Location: "cellar"})
	f.add(Pantry, Values{Name: "Passata", Quantity: One, Location: "Cellar"})
	f.add(Medicines, Values{Name: "Paracetamol", Quantity: One, Location: "Bathroom"})
	if got, _ := f.stock.Locations(ctx); !slices.Equal(got, []string{"Bathroom", "Cellar"}) {
		t.Errorf("locations %v", got)
	}
}
