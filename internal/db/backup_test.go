package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func openTemp(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(context.Background(), filepath.Join(dir, "hausbuch.db"), filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, filepath.Join(dir, "backups")
}

func count(t *testing.T, path, table string) int {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return n
}

func files(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestBackupIsACompleteDatabase(t *testing.T) {
	db, dir := openTemp(t)
	if _, err := db.Exec("CREATE TABLE note (body TEXT); INSERT INTO note VALUES ('a'), ('b')"); err != nil {
		t.Fatal(err)
	}
	if err := Backup(context.Background(), db, dir, "copy.db"); err != nil {
		t.Fatal(err)
	}
	if n := count(t, filepath.Join(dir, "copy.db"), "note"); n != 2 {
		t.Errorf("the copy has %d notes, want 2", n)
	}
	if got := files(t, dir); len(got) != 1 || got[0] != "copy.db" {
		t.Errorf("backups directory holds %v, want only copy.db", got)
	}
}

// Notes are written two at a time, in one transaction: a copy holding an odd number would hold half a write.
func TestBackupWhileWriting(t *testing.T) {
	db, dir := openTemp(t)
	if _, err := db.Exec("CREATE TABLE note (body TEXT)"); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if _, err := db.Exec("BEGIN; INSERT INTO note VALUES ('a'); INSERT INTO note VALUES ('b'); COMMIT"); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := range 5 {
		name := "copy" + string(rune('0'+i)) + ".db"
		if err := Backup(context.Background(), db, dir, name); err != nil {
			t.Fatal(err)
		}
		if n := count(t, filepath.Join(dir, name), "note"); n%2 != 0 {
			t.Errorf("%s holds %d notes: half a write", name, n)
		}
	}
	close(stop)
	wg.Wait()
	if n := count(t, filepath.Join(dir, "copy4.db"), "note"); n == 0 {
		t.Error("no write happened during the copies; the test proves nothing")
	}
}

func TestBackupRemovesLeftovers(t *testing.T) {
	db, dir := openTemp(t)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "daily-2026-01-01.db.tmp"), []byte("half"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Backup(context.Background(), db, dir, "copy.db"); err != nil {
		t.Fatal(err)
	}
	if got := files(t, dir); len(got) != 1 || got[0] != "copy.db" {
		t.Errorf("backups directory holds %v, want only copy.db", got)
	}
}

func TestBackupReplacesACopyWithTheSameName(t *testing.T) {
	db, dir := openTemp(t)
	if _, err := db.Exec("CREATE TABLE note (body TEXT)"); err != nil {
		t.Fatal(err)
	}
	if err := Backup(context.Background(), db, dir, "copy.db"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO note VALUES ('a')"); err != nil {
		t.Fatal(err)
	}
	if err := Backup(context.Background(), db, dir, "copy.db"); err != nil {
		t.Fatal(err)
	}
	if n := count(t, filepath.Join(dir, "copy.db"), "note"); n != 1 {
		t.Errorf("the copy has %d notes, want 1", n)
	}
}
