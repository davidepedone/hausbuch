package db

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Backup saves a copy of the database in dir under name, consistent even while others write to it. The copy is
// written to a temporary file and renamed when complete, so a file under a copy's name is always a whole database;
// temporary files left by a copy interrupted earlier are removed first. An existing copy with the same name is
// replaced.
func Backup(ctx context.Context, db *sql.DB, dir, name string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if err != nil {
		return err
	}
	for _, f := range leftovers {
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	tmp := filepath.Join(dir, name+".tmp")
	// VACUUM INTO reads the database in one transaction, which in WAL mode doesn't block writers.
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, name))
}

// DailyBackup saves the copy of the day of now, in now's time zone, unless it already exists, then removes the
// oldest daily copies beyond keep. Copies taken before schema changes are left alone.
func DailyBackup(ctx context.Context, db *sql.DB, dir string, keep int, now time.Time) error {
	name := "daily-" + now.Format(time.DateOnly) + ".db"
	if _, err := os.Stat(filepath.Join(dir, name)); errors.Is(err, fs.ErrNotExist) {
		if err := Backup(ctx, db, dir, name); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	// The names sort by date.
	daily, err := filepath.Glob(filepath.Join(dir, "daily-*.db"))
	if err != nil {
		return err
	}
	slices.Sort(daily)
	for _, f := range daily[:max(0, len(daily)-keep)] {
		if err := os.Remove(f); err != nil {
			return err
		}
	}
	return nil
}
