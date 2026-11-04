package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenConfiguresConnections(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "hausbuch.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Two connections held open at once are two distinct connections, so pragmas applied only to the first one
	// would show on the second.
	for i := range 2 {
		c, err := db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		var journal string
		var busy, fk int
		if err := c.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&journal); err != nil {
			t.Fatal(err)
		}
		if err := c.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&busy); err != nil {
			t.Fatal(err)
		}
		if err := c.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&fk); err != nil {
			t.Fatal(err)
		}
		if journal != "wal" || busy != 5000 || fk != 1 {
			t.Errorf("connection %d: journal_mode=%s busy_timeout=%d foreign_keys=%d", i, journal, busy, fk)
		}
	}
}

func TestOpenAppliesMigrationsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hausbuch.db")

	versions := func() []int64 {
		t.Helper()
		db, err := Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		rows, err := db.Query("SELECT version_id FROM goose_db_version WHERE version_id > 0 ORDER BY id")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var vs []int64
		for rows.Next() {
			var v int64
			if err := rows.Scan(&v); err != nil {
				t.Fatal(err)
			}
			vs = append(vs, v)
		}
		return vs
	}

	first := versions()
	if len(first) == 0 {
		t.Fatal("fresh database has no applied migrations")
	}
	if second := versions(); len(second) != len(first) {
		t.Errorf("second open applied migrations again: %v, then %v", first, second)
	}
}

// A transaction that checks and then writes must not let another one check in between: transactions take the write
// lock when they begin, so a second one waits for the first to end.
func TestTransactionsTakeTheWriteLockWhenTheyBegin(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "hausbuch.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	began := make(chan error, 1)
	go func() {
		second, err := db.BeginTx(context.Background(), nil)
		if err == nil {
			err = second.Rollback()
		}
		began <- err
	}()
	select {
	case err := <-began:
		t.Errorf("a second transaction began while the first was open (%v)", err)
	case <-time.After(200 * time.Millisecond):
	}
	if err := first.Rollback(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-began:
		if err != nil {
			t.Errorf("second transaction: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("the second transaction didn't begin after the first ended")
	}
}
