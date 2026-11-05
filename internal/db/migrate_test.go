package db

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

// withMigration returns the real migrations plus one more, after them, with the given SQL.
func withMigration(t *testing.T, sql string) fs.FS {
	t.Helper()
	real, err := fs.Sub(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	entries, err := fs.ReadDir(real, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := fs.ReadFile(real, e.Name())
		if err != nil {
			t.Fatal(err)
		}
		fsys[e.Name()] = &fstest.MapFile{Data: b}
	}
	fsys["90000_test.sql"] = &fstest.MapFile{Data: []byte("-- +goose Up\n" + sql + "\n-- +goose Down\n")}
	return fsys
}

// current opens path with the real migrations, adds a row so it holds data, and closes it.
func current(t *testing.T, path string) {
	t.Helper()
	db, err := Open(context.Background(), path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO app_user (username, username_key, display_name, password_hash) VALUES ('anna', 'anna', 'Anna', 'x')"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func lastVersion(t *testing.T) int64 {
	t.Helper()
	real, err := fs.Sub(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	names, err := fs.Glob(real, "*.sql")
	if err != nil || len(names) == 0 {
		t.Fatal(names, err)
	}
	var v int64
	for _, n := range names {
		x, err := strconv.ParseInt(strings.SplitN(n, "_", 2)[0], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		v = max(v, x)
	}
	return v
}

func TestCopyBeforeASchemaChange(t *testing.T) {
	dir := t.TempDir()
	path, backups := filepath.Join(dir, "hausbuch.db"), filepath.Join(dir, "backups")
	current(t, path)

	db, err := open(context.Background(), path, backups, withMigration(t, "ALTER TABLE app_user ADD COLUMN note TEXT;"))
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	copyPath := filepath.Join(backups, beforeSchema(lastVersion(t)))
	if got := files(t, backups); len(got) != 1 {
		t.Errorf("backups directory holds %v, want one copy", got)
	}
	if n := count(t, copyPath, "app_user"); n != 1 {
		t.Errorf("the copy holds %d users, want 1", n)
	}
	if hasColumn(t, copyPath, "app_user", "note") {
		t.Error("the copy has the new column: it was taken after the schema change")
	}
}

func TestNoCopyWhenTheBackupsDirectoryCantBeWritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hausbuch.db")
	current(t, path)
	notADir := filepath.Join(dir, "file")
	if err := os.WriteFile(notADir, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := open(context.Background(), path, filepath.Join(notADir, "backups"),
		withMigration(t, "ALTER TABLE app_user ADD COLUMN note TEXT;"))
	if err == nil {
		t.Fatal("opened and migrated without a copy")
	}
	t.Logf("error: %v", err)
	if hasColumn(t, path, "app_user", "note") {
		t.Error("the database was migrated although the copy failed")
	}
}

func TestNoCopyWithoutSchemaChanges(t *testing.T) {
	dir := t.TempDir()
	path, backups := filepath.Join(dir, "hausbuch.db"), filepath.Join(dir, "backups")
	current(t, path)
	db, err := Open(context.Background(), path, backups)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if _, err := os.Stat(backups); !os.IsNotExist(err) {
		t.Errorf("backups directory exists (%v): a copy was made without schema changes", err)
	}
}

func TestNoCopyOnFirstStart(t *testing.T) {
	dir := t.TempDir()
	backups := filepath.Join(dir, "backups")
	db, err := Open(context.Background(), filepath.Join(dir, "hausbuch.db"), backups)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if _, err := os.Stat(backups); !os.IsNotExist(err) {
		t.Errorf("backups directory exists (%v): a new database was copied", err)
	}
}

func TestRepeatedFailingStartsKeepOneCopy(t *testing.T) {
	dir := t.TempDir()
	path, backups := filepath.Join(dir, "hausbuch.db"), filepath.Join(dir, "backups")
	current(t, path)
	failing := withMigration(t, "ALTER TABLE app_user ADD COLUMN note TEXT; SELECT * FROM no_such_table;")
	for range 3 {
		if _, err := open(context.Background(), path, backups, failing); err == nil {
			t.Fatal("the failing migration succeeded")
		}
	}
	got := files(t, backups)
	if len(got) != 1 || got[0] != beforeSchema(lastVersion(t)) {
		t.Errorf("backups directory holds %v, want only %s", got, beforeSchema(lastVersion(t)))
	}
	if n := count(t, filepath.Join(backups, got[0]), "app_user"); n != 1 {
		t.Errorf("the copy holds %d users, want 1", n)
	}
	if hasColumn(t, path, "app_user", "note") {
		t.Error("the failed migration left its column behind")
	}
}

func hasColumn(t *testing.T, path, table, column string) bool {
	t.Helper()
	return count(t, path, "pragma_table_info('"+table+"') WHERE name = '"+column+"'") > 0
}
