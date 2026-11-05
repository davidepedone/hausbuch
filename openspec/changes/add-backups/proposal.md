# Proposal

## Why

hausbuch is meant to run unattended for long periods, with its image updated automatically by a tool that pulls the
newest release and restarts the container. Nobody is there to take a backup before an update, and today the only
backup is a manual `sqlite3 .backup` described in the README: a release with a broken migration, or a bug that damages
pages, could lose a household's data with no copy to go back to. The application should keep its own copies, and
going back to the previous release should keep working.

## What Changes

- Before applying pending schema migrations to a database that already has data, hausbuch copies the database into a
  backups directory. If the copy fails, it does not migrate and does not start.
- Once a day, at start and then every 24 hours, hausbuch copies the database into the same directory, keeping the most
  recent copies (7 by default) and removing older ones.
- Copies are consistent while the server runs (SQLite `VACUUM INTO`), and a copy interrupted half-way never looks
  like a complete one.
- Two new environment variables: the backups directory (default: `backups` next to the database, so `/data/backups`
  in the image) and how many daily copies to keep.
- A database migrated by a newer release keeps working with the previous release, so an installation can be rolled
  back by running the previous image. From this change on, schema migrations are additive only.
- The README explains where the copies are, how to restore one, and that they share the disk with the database.

### Out of scope

- Protection against losing the disk or the machine: the copies sit on the same volume as the database. Copying them
  elsewhere is up to whoever deploys hausbuch, and the README says so.
- Sending backups anywhere (another host, cloud storage, email).
- Restoring from the web interface or with a command: restoring is a manual step done with the container stopped.
- Showing or managing backups in the web interface.
- Backing up attachments, which don't exist yet; the change that adds them extends the backups.
- Automatic rollback of a failed update.

## Capabilities

### New Capabilities

- `database-backups`: the copies of the database hausbuch keeps by itself (before migrations and daily), how many are
  kept, and running a previous release on a database migrated by a newer one.

### Modified Capabilities

None.

## Impact

- `internal/db`: a copy before pending migrations; a function making a consistent copy and pruning old ones.
- `cmd/hausbuch`: the daily copy alongside the existing daily cleanup; two new environment variables, validated at
  start.
- README: configuration table, backups and restore, rollback.
- Development rule for future changes: migrations only add tables, columns (nullable or with a default) and indexes;
  they never rename or drop.
- No new dependencies.
