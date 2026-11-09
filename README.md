# hausbuch

A private household wiki for the people living in a home: recipes, how-to procedures, seasonal chores, where things
are kept. Pages start from templates, can carry photos and files, and can have reminders that show up in the app when
they are due. Three sections, Medicines, Pantry and Supplies, keep track of what the household has at home, and the
home page points out what has expired or is about to.

Work in progress: the specification lives in [`openspec/`](openspec/).

## Running it

hausbuch is a single Docker image, `ghcr.io/davidepedone/hausbuch`, for `linux/amd64` and `linux/arm64`. Everything
it stores is one SQLite database in `/data`, so that directory must be on a volume.

[`docker-compose.yml`](docker-compose.yml) runs it on port 8090 with its data in `./data`, next to the file. Its
`com.centurylinklabs.watchtower.enable` label lets a [watchtower](https://github.com/nicholas-fedor/watchtower) already
running on the host, with label-enable, install every release moved to `:stable`. The labels starting with
`dev.dozzle.` give the container a name and an icon in [Dozzle](https://dozzle.dev/), for a host that runs it.

Releases are versioned by date, `YYYY.MM.N` (the `N`th release of that month), and published under three kinds of tag:

- `stable`: the newest release that has been tried; what an installation should normally run.
- `candidate`: the newest release, before it has been tried. Run it on one installation to try releases first.
- `YYYY.MM.N`: one release, to pin an installation to it.

Each release also has `YYYY.MM.N-amd64` and `YYYY.MM.N-arm64` tags, the image of one platform: use the tags above
instead, which pick the right one.

The container runs as the unprivileged user `65532`. With a host directory, as in `docker-compose.yml`, that
directory must be writable by that user: `chown 65532:65532 ./data`. With a named volume instead, Docker gives it the
right owner.

The image reports its health to Docker by itself (`/hausbuch healthcheck`, calling `/healthz`).

### Configuration

| Variable | Purpose | Default |
| --- | --- | --- |
| `HAUSBUCH_DB_PATH` | SQLite database file | `/data/hausbuch.db` |
| `HAUSBUCH_LISTEN` | listen address | `:8080` |
| `HAUSBUCH_SESSION_TIMEOUT` | how long a session lasts without being used, as a Go duration | `720h` (30 days) |
| `HAUSBUCH_DEFAULT_LANGUAGE` | language new pages start in: `en` (English) or `it` (Italian) | `en` |
| `HAUSBUCH_BACKUP_DIR` | directory the copies of the database are saved in (see [Backups](#backups)) | `backups` next to the database file |
| `HAUSBUCH_BACKUP_KEEP` | how many daily copies to keep, at least 1 | `7` |
| `HAUSBUCH_MEDICINES_LEAD_DAYS` | how many days before their expiry date medicines are pointed out as expiring, at least 0 | `30` |
| `HAUSBUCH_PANTRY_LEAD_DAYS` | the same for the pantry | `7` |
| `HAUSBUCH_SUPPLIES_LEAD_DAYS` | the same for household supplies | `7` |
| `TZ` | time zone dates are shown in, such as `Europe/Rome`; it also decides when a day's copy is due and when an item has expired | `UTC` |

### Accounts

Every page needs signing in, and there is no sign-up: accounts are managed with the `user` commands of the binary,
run inside the container while it is running. The image has no shell, so give the binary as the command:

```sh
docker exec -it hausbuch /hausbuch user add anna "Anna Rossi"   # asks for the password twice
docker exec -it hausbuch /hausbuch user list
docker exec -it hausbuch /hausbuch user set-password anna
docker exec -it hausbuch /hausbuch user disable anna             # also signs them out everywhere
docker exec -it hausbuch /hausbuch user enable anna
docker exec -it hausbuch /hausbuch user unlock anna              # after 3 wrong passwords in a row
```

Replace `hausbuch` after `-it` with the container's name. Passwords are never given as arguments; without `-t` (no
terminal) they are read from standard input instead, e.g. `echo "$PASSWORD" | docker exec -i …`.

After 3 wrong passwords in a row a username is locked for 15 minutes, even with the right password.

### HTTPS

hausbuch serves plain HTTP. Passwords and session cookies travel unencrypted unless a reverse proxy in front of it
terminates HTTPS; setting that up is up to whoever deploys it.

### Backups

hausbuch copies its database by itself into `HAUSBUCH_BACKUP_DIR`, `/data/backups` in the image:

- `daily-YYYY-MM-DD.db`: one copy for each day it runs, made at start and then checked every hour. The newest
  `HAUSBUCH_BACKUP_KEEP` are kept, older ones are removed.
- `before-schema-N.db`: made before a new release changes the database's structure, if the database already holds
  data. If this copy can't be made, hausbuch doesn't start and leaves the database as it was. These copies are never
  removed automatically.

Each copy is a complete SQLite database, consistent even when made while people are editing. A copy interrupted
half-way is never left under one of these names.

The copies are on the same volume as the database: they protect against a bad update or a mistake, not against
losing the disk. To keep copies elsewhere too, copy the files of the backups directory to another machine or disk;
they can be copied at any time.

To restore a copy:

1. Stop the container.
2. In the data directory, replace `hausbuch.db` with the copy, and delete `hausbuch.db-wal` and `hausbuch.db-shm` if
   they are there: they belong to the database being replaced, and would otherwise be applied to the restored one.
3. Start the container.

With a named volume, a throwaway container reaches its files. For example, to restore the copy of 3 November 2026
when the volume is called `hausbuch-data` (Compose puts the project's name in front of it: `docker volume ls` shows
the full name):

```sh
docker stop hausbuch
docker run --rm -v hausbuch-data:/data busybox sh -c \
  'cp /data/backups/daily-2026-11-03.db /data/hausbuch.db && rm -f /data/hausbuch.db-wal /data/hausbuch.db-shm && chown 65532:65532 /data/hausbuch.db'
docker start hausbuch
```

### Going back to a previous release

A release never changes the database in a way the previous release can't work with, so going back is running the
previous image: pin its version (`image: ghcr.io/davidepedone/hausbuch:YYYY.MM.N`), or move `:stable` back onto it
when releasing (see [Releasing](#releasing)).

## Development

Requirements: Go (the version in `go.mod`), [golangci-lint](https://golangci-lint.run/) v2, `make` and `curl`. The
Tailwind CSS CLI is downloaded by `make` into `bin/`.

```sh
make check   # lint, tests and build, as in CI
HAUSBUCH_DB_PATH=./hausbuch.db HAUSBUCH_LISTEN=127.0.0.1:8080 ./hausbuch
```

Migrations, in `internal/db/migrations`, are only ever added, never edited, and they only add: tables, indexes, and
columns that are nullable or have a default. They never rename or drop a table or a column, change what a column
means, or add a constraint existing rows or code could break. That keeps the previous release working on the database
a newer one migrated, which is what makes going back to it possible; a change that needs more is split across two
releases (first stop using the old shape, then remove it).

`make check` also generates `internal/web/static/app.css`; run `make css` after changing classes in the templates when
not going through `make check`.

### Releasing

Releases are cut from an up-to-date, clean `main` on a machine with Docker (Buildx) and the GitHub CLI logged in with
the `write:packages` scope:

```sh
scripts/release.sh             # make check, then the next YYYY.MM.N pushed as :candidate, git tag and GitHub release
scripts/promote.sh             # once it has been tried: moves :stable onto the newest release
scripts/promote.sh 2026.11.1   # or onto a given one, e.g. to roll back
```
