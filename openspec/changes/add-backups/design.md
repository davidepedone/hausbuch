# Design

## Context

See `proposal.md` for the motivation and `specs/database-backups/spec.md` for the required behaviour. The current state
that shapes the approach:

- Everything hausbuch stores is one SQLite database in WAL mode (`HAUSBUCH_DB_PATH`, `/data/hausbuch.db` in the
  image); `/data` is the container's only volume and is owned by the unprivileged user the container runs as.
- `db.Open` applies pending migrations with goose at every open, both for the server and for the `user` commands,
  which run as a second process while the server runs.
- Migrations are embedded SQL files run by goose, each in a transaction.
- The server already has a daily loop (`cleanUp` in `cmd/hausbuch`) for expired sessions and deleted pages.
- Checked against goose v3.29: a database recording a migration version the binary doesn't know opens without error,
  and nothing is applied. An older binary therefore starts on a newer schema; whether it then works depends only on
  what the newer migrations did.

## Goals / Non-Goals

**Goals:**

- Copies that are consistent without stopping the server, and that are usable as they are.
- No new dependency, no new process, nothing to set up: the defaults work in the image as it is.

**Non-Goals:**

- Incremental or compressed copies: the database of a household is a few megabytes at most.
- Checking a copy's integrity after making it, beyond SQLite writing it.

## Decisions

### Copies with `VACUUM INTO`

`VACUUM INTO '<file>'` writes a new, compact database from a single read transaction. In WAL mode readers don't block
writers, so residents can keep saving while it runs, and the copy holds the database exactly as it was at one moment.
The copy has no `-wal` file of its own: it is one self-contained file.

Each copy is written to `<name>.tmp` in the backups directory and renamed to `<name>` once complete; the rename is
atomic on the same filesystem, so a file with a copy's name is always complete. Leftover `*.tmp` files, from a process
stopped half-way, are removed before each copy. `VACUUM INTO` refuses an existing target, so the temporary file is
removed first if present.

Alternatives considered:
- *SQLite's online backup API*: equally consistent, but the pure-Go driver exposes it only through driver-specific
  calls; `VACUUM INTO` is plain SQL.
- *Copying the file*: not consistent while the server writes, and the WAL file would have to be copied with it.

### Before migrations, inside `db.Open`

`db.Open` already knows when migrations are pending, so the copy goes there: goose's provider reports the current
version and whether migrations are pending. When the current version is 0 the database is new (nothing to protect);
otherwise, with migrations pending, `db.Open` makes the copy before calling `Up`. If the copy fails, `db.Open`
returns the error without migrating, and the server or command exits with it.

The copy is named after the version it was taken at, `before-schema-<version>.db`, and is rewritten on every attempt.
A failed migration rolls back in its transaction, so each attempt copies the same data again: a container restarting
in a loop after a failed update keeps a single, current copy instead of filling the disk. Since `db.Open` is shared,
a `user` command run right after an update, before the server starts, makes the copy too.

These copies are never removed automatically: there is one per release with schema changes, a few a year.

`db.Open` takes the backups directory as a parameter; callers get it from configuration.

### Daily copies, checked hourly

A goroutine of the server checks once an hour whether today's copy, `daily-YYYY-MM-DD.db` in the configured time zone
(`TZ`), exists; if not, it makes it and then removes the oldest `daily-*.db` beyond the number to keep. The first check
runs at start. Checking hourly rather than sleeping 24 hours means every calendar day the server runs gets its copy,
whatever time it started, and restarts on the same day make no second copy. Names sort by date, so pruning sorts them
and removes all but the last N. A failure is logged and retried at the next check.

It is a separate loop from the daily cleanup, which keeps its 24-hour period and its job.

### Configuration

| Variable | Purpose | Default |
| --- | --- | --- |
| `HAUSBUCH_BACKUP_DIR` | directory the copies are saved in | `backups` next to the database file |
| `HAUSBUCH_BACKUP_KEEP` | number of daily copies to keep, at least 1 | `7` |

Both are read and validated in `main` before the database is opened, for the server and for the `user` commands
(which only need the directory). The directory is created if missing. With the image's defaults it is
`/data/backups`, inside the volume, so it needs no extra mount and is owned by the container's user.

### Rolling back: additive migrations only

Running the previous image works if the previous code still works on the newer schema. From this change on,
migrations only add: new tables, new indexes, new columns that are nullable or have a default (so that inserts by
older code, which don't name them, still succeed). They never rename or drop tables or columns, never change a
column's meaning, and never add constraints existing code could violate. A change that really needs one of these
must be designed in two releases: first stop using the old shape, then remove it.

The rule is written in the README's development section. A test applies a stand-in future migration (a new table and
a new column with a default, plus a goose version the code doesn't know) to a current database and checks that the
current code still opens it, signs in, saves, lists and searches pages: it pins goose's behaviour and shows what an
additive migration means.

Alternative considered: *down migrations run by the newer release on rollback*. The older image doesn't know them
and the newer one isn't running, so nothing would run them.

## Risks / Trade-offs

- [The copies are on the same disk as the database] → Stated in the README, with the advice to copy
  `/data/backups` elsewhere. Out of the application's reach.
- [Restoring by hand while leaving the old `-wal` file next to the restored database would mix the two] → The README's
  restore steps say to stop the container and remove `hausbuch.db-wal` and `hausbuch.db-shm` before starting it again.
- [The copy before a schema change is rewritten on each attempt] → Each attempt copies the database the failed
  migration left unchanged, so the content is the same; rewriting it is what keeps it single and current.
- [A schema change made by mistake in a non-additive way breaks rollback] → The README rule and the stand-in migration
  test; review of each migration.
- [The disk fills up] → Copies are small, at most `HAUSBUCH_BACKUP_KEEP` daily copies plus one per schema change. If
  a daily copy fails it is logged and the server keeps running; if the copy before a schema change fails the server
  doesn't start, which leaves the data untouched.
- [The server is down for a whole day] → No copy for that day, and the retention counts copies, not days, so the last
  N copies are kept regardless of gaps.

## Migration Plan

This change adds no migration. The first release with it starts making daily copies at once; the first copy before a
schema change is made by the first later release that has migrations. Rolling back from this release to the previous
one needs nothing: the previous one ignores the backups directory.
