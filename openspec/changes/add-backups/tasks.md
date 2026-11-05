# Tasks

## 1. Making a copy

- [ ] 1.1 Add to `internal/db` a function copying the open database with `VACUUM INTO` to `<name>.tmp` in the backups directory (created if missing, leftover `*.tmp` removed first) and renaming it to `<name>`; verify tests that the copy opens as a database with the same rows, that a page saved while copying leaves the copy consistent, and that a leftover `.tmp` file is removed and never counted as a copy

## 2. Copy before a schema change

- [ ] 2.1 Make `db.Open` take the backups directory and, when the database has a schema version and migrations are pending, write `before-schema-<version>.db` before migrating, returning the error without migrating if the copy fails; verify tests (with an extra test migration) for every "Copy before a schema change" scenario: newer release, copy impossible (backups directory not writable), no pending migrations, first start, and repeated failing starts leaving one copy
- [ ] 2.2 Read `HAUSBUCH_BACKUP_DIR` in `main` (default `backups` next to the database file) and pass it to `db.Open` for the server and the `user` commands; verify a test of the "Default location" scenario

## 3. Daily copies

- [ ] 3.1 Add the server's hourly check making `daily-YYYY-MM-DD.db` in the configured time zone when missing, then removing the oldest daily copies beyond `HAUSBUCH_BACKUP_KEEP` and never `before-schema-*` copies, logging failures without stopping; verify tests with a controllable clock for every "Daily copy" and "Old daily copies are removed" scenario
- [ ] 3.2 Read and validate `HAUSBUCH_BACKUP_KEEP` (default 7, a whole number of at least 1) at start; verify a test of the "Invalid number of copies" scenario
- [ ] 3.3 Document the two variables in the README's configuration table, rewrite its "Backups" section (where the copies are, that they share the disk, copying them elsewhere, restoring with the container stopped and the `-wal`/`-shm` files removed); verify the restore steps against a running container in task 5.1

## 4. Rolling back

- [ ] 4.1 Add a test applying a stand-in future migration (a new table, a new column with a default, and a goose version the code doesn't know) to a current database, then checking that the current code opens it, signs in, saves, lists and searches pages; verify it passes for the "Rolling back" scenario
- [ ] 4.2 Write the additive-migrations rule in the README's development section; verify it matches the design's rule

## 5. Integration

- [ ] 5.1 Run the image from the README's example compose: let it make its daily copy, restart it the same day (no second copy), stop it, restore the copy following the README, start it and find the pages as they were; then start an image built with an extra test migration and check the `before-schema-*` copy, then the previous image on the migrated database; verify every step behaves as the specs say

## Workflow follow-up

- Archive the change once merged.
