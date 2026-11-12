package db

import (
	"context"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

// upTo returns the real migrations up to and including version.
func upTo(t *testing.T, version int64) fs.FS {
	t.Helper()
	real, err := fs.Sub(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	names, err := fs.Glob(real, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	for _, n := range names {
		if v, _ := strconv.ParseInt(strings.SplitN(n, "_", 2)[0], 10, 64); v <= version {
			b, err := fs.ReadFile(real, n)
			if err != nil {
				t.Fatal(err)
			}
			fsys[n] = &fstest.MapFile{Data: b}
		}
	}
	return fsys
}

func TestQuantitiesBecomeTenths(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "hausbuch.db")
	old, err := open(ctx, path, dir, upTo(t, 6))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"INSERT INTO app_user (id, username, username_key, display_name, password_hash) VALUES (1, 'anna', 'anna', 'Anna', 'x')",
		`INSERT INTO stock_item (id, section, name, name_key, quantity, unit, location, expires_on, opened_on, opened_days,
			created_by, created_at, changed_by, changed_at)
		VALUES (7, 'pantry', 'Passata', 'passata', 3, 'bottles', 'Cellar', '2027-09-30', '2026-11-06', 4, 1, 100, 1, 200)`,
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = old.Close()

	db, err := Open(ctx, path, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var got string
	err = db.QueryRow(`SELECT concat_ws('|', id, section, name, name_key, quantity_tenths, unit, location, expires_on,
		opened_on, opened_days, created_by, created_at, changed_by, changed_at) FROM stock_item`).Scan(&got)
	if want := "7|pantry|Passata|passata|30|bottles|Cellar|2027-09-30|2026-11-06|4|1|100|1|200"; err != nil || got != want {
		t.Errorf("row %q, %v, want %q", got, err, want)
	}
	var index int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = 'stock_item_section_name'").
		Scan(&index); err != nil || index != 1 {
		t.Errorf("index: %d, %v", index, err)
	}
}

func TestRowsGetNoCategory(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "hausbuch.db")
	old, err := open(ctx, path, dir, upTo(t, 7))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"INSERT INTO app_user (id, username, username_key, display_name, password_hash) VALUES (1, 'anna', 'anna', 'Anna', 'x')",
		`INSERT INTO stock_item (section, name, name_key, quantity_tenths, created_by, created_at, changed_by, changed_at)
		VALUES ('pantry', 'Passata', 'passata', 30, 1, 100, 1, 200)`,
		"INSERT INTO stock_product (section, name_key, name) VALUES ('pantry', 'passata', 'Passata')",
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = old.Close()

	db, err := Open(ctx, path, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var item, product string
	if err := db.QueryRow("SELECT (SELECT category FROM stock_item), (SELECT category FROM stock_product)").
		Scan(&item, &product); err != nil || item != "" || product != "" {
		t.Errorf("categories %q %q, %v", item, product, err)
	}
}

func TestProductsGetNoUnit(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "hausbuch.db")
	old, err := open(ctx, path, dir, upTo(t, 12))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec("INSERT INTO stock_product (section, name_key, name) VALUES ('pantry', 'passata', 'Passata')"); err != nil {
		t.Fatal(err)
	}
	_ = old.Close()

	db, err := Open(ctx, path, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var unit string
	if err := db.QueryRow("SELECT unit FROM stock_product").Scan(&unit); err != nil || unit != "" {
		t.Errorf("unit %q, %v", unit, err)
	}
}
