package db

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestDailyCopies(t *testing.T) {
	db, dir := openTemp(t)
	day := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	for i := range 3 {
		if err := DailyBackup(context.Background(), db, dir, 7, day.AddDate(0, 0, i)); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"daily-2026-03-01.db", "daily-2026-03-02.db", "daily-2026-03-03.db"}
	if got := files(t, dir); !slices.Equal(got, want) {
		t.Errorf("backups directory holds %v, want %v", got, want)
	}
}

func TestNoSecondCopyOnTheSameDay(t *testing.T) {
	db, dir := openTemp(t)
	if _, err := db.Exec("CREATE TABLE note (body TEXT)"); err != nil {
		t.Fatal(err)
	}
	morning := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	if err := DailyBackup(context.Background(), db, dir, 7, morning); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO note VALUES ('a')"); err != nil {
		t.Fatal(err)
	}
	if err := DailyBackup(context.Background(), db, dir, 7, morning.Add(10*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := files(t, dir); !slices.Equal(got, []string{"daily-2026-03-01.db"}) {
		t.Errorf("backups directory holds %v, want one copy", got)
	}
	if n := count(t, filepath.Join(dir, "daily-2026-03-01.db"), "note"); n != 0 {
		t.Error("the day's copy was made again")
	}
}

func TestDailyCopyIsNamedInTheTimeZone(t *testing.T) {
	db, dir := openTemp(t)
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatal(err)
	}
	// 23:30 in UTC is already the next day in Rome.
	if err := DailyBackup(context.Background(), db, dir, 7, time.Date(2026, 3, 1, 23, 30, 0, 0, time.UTC).In(rome)); err != nil {
		t.Fatal(err)
	}
	if got := files(t, dir); !slices.Equal(got, []string{"daily-2026-03-02.db"}) {
		t.Errorf("backups directory holds %v, want the copy of 2 March", got)
	}
}

func TestOldDailyCopiesAreRemoved(t *testing.T) {
	db, dir := openTemp(t)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, beforeSchema(4)), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	for i := range 10 {
		if err := DailyBackup(context.Background(), db, dir, 7, day.AddDate(0, 0, i)); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{beforeSchema(4)}
	for i := 3; i < 10; i++ {
		want = append(want, "daily-"+day.AddDate(0, 0, i).Format(time.DateOnly)+".db")
	}
	slices.Sort(want)
	if got := files(t, dir); !slices.Equal(got, want) {
		t.Errorf("backups directory holds %v, want %v", got, want)
	}
}

func TestDailyCopyAfterAnInterruptedOne(t *testing.T) {
	db, dir := openTemp(t)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "daily-2026-03-01.db.tmp"), []byte("half"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := DailyBackup(context.Background(), db, dir, 7, time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if got := files(t, dir); !slices.Equal(got, []string{"daily-2026-03-01.db"}) {
		t.Errorf("backups directory holds %v, want only the day's copy", got)
	}
	if n := count(t, filepath.Join(dir, "daily-2026-03-01.db"), "goose_db_version"); n == 0 {
		t.Error("the day's copy is empty")
	}
}
