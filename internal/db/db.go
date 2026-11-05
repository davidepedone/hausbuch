// Package db opens the SQLite database and keeps its schema up to date.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

//go:embed migrations/*.sql
var migrations embed.FS

// Pragmas go in the DSN rather than in a PRAGMA statement after opening, so that every connection of the pool gets
// them, not only the first one. Transactions start with BEGIN IMMEDIATE, taking the write lock at once: a
// transaction that checks and then writes (is the title free?) can't have its check go stale before it writes.
const pragmas = "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_txlock=immediate"

// Open opens the SQLite database at path, creating it if needed, and applies pending migrations. Before migrating a
// database that already has a schema, it saves a copy of it in backupDir, and doesn't migrate if that fails.
func Open(ctx context.Context, path, backupDir string) (*sql.DB, error) {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, err
	}
	return open(ctx, path, backupDir, fsys)
}

func open(ctx context.Context, path, backupDir string, fsys fs.FS) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+pragmas)
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, db, backupDir, fsys); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB, backupDir string, fsys fs.FS) error {
	p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	if err != nil {
		return err
	}
	pending, err := p.HasPending(ctx)
	if err != nil || !pending {
		return err
	}
	version, err := p.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	// Version 0 is a new database: nothing to protect. A failed migration rolls back its own transaction, so a start
	// retried after one copies the same data under the same name again: retries don't pile up copies. Migrations
	// applied before the failed one stay, so the retry's copy has a new name and the first copy is kept.
	if version > 0 {
		if err := Backup(ctx, db, backupDir, beforeSchema(version)); err != nil {
			return fmt.Errorf("back up before migrating: %w", err)
		}
	}
	_, err = p.Up(ctx)
	return err
}

// beforeSchema is the name of the copy taken before migrating a database from version.
func beforeSchema(version int64) string { return fmt.Sprintf("before-schema-%d.db", version) }
