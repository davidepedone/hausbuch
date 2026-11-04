# hausbuch

A private household wiki for the people living in a home: recipes, how-to procedures, seasonal chores, where things
are kept. Pages start from templates, can carry photos and files, and can have reminders that show up in the app when
they are due.

Work in progress: the specification lives in [`openspec/`](openspec/).

## Running it

hausbuch is a single Docker image, `ghcr.io/davidepedone/hausbuch`, for `linux/amd64` and `linux/arm64`. Everything
it stores is one SQLite database in `/data`, so that directory must be on a volume.

```yaml
services:
  hausbuch:
    image: ghcr.io/davidepedone/hausbuch:stable
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - hausbuch-data:/data

volumes:
  hausbuch-data:
```

Releases are versioned by date, `YYYY.MM.N` (the `N`th release of that month), and published under three kinds of tag:

- `stable`: the newest release that has been tried; what an installation should normally run.
- `candidate`: the newest release, before it has been tried. Run it on one installation to try releases first.
- `YYYY.MM.N`: one release, to pin an installation to it.

Each release also has `YYYY.MM.N-amd64` and `YYYY.MM.N-arm64` tags, the image of one platform: use the tags above
instead, which pick the right one.

The container runs as the unprivileged user `65532`. With a named volume, as above, Docker gives it the right owner.
With a bind mount (`./data:/data`), the host directory must be writable by that user:
`chown 65532:65532 ./data`.

The image reports its health to Docker by itself (`/hausbuch healthcheck`, calling `/healthz`).

### Configuration

| Variable | Purpose | Default |
| --- | --- | --- |
| `HAUSBUCH_DB_PATH` | SQLite database file | `/data/hausbuch.db` |
| `HAUSBUCH_LISTEN` | listen address | `:8080` |
| `HAUSBUCH_SESSION_TIMEOUT` | how long a session lasts without being used, as a Go duration | `720h` (30 days) |
| `HAUSBUCH_DEFAULT_LANGUAGE` | language new pages start in: `en` (English) or `it` (Italian) | `en` |
| `TZ` | time zone dates are shown in, such as `Europe/Rome` | `UTC` |

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

Don't copy the database file while hausbuch is writing to it: the copy can be inconsistent. Either stop the container
and copy the whole `/data` directory (the database may come with `-wal` and `-shm` files that belong to it), or, with
the container running, use SQLite's online backup from the host:

```sh
sqlite3 /path/to/data/hausbuch.db ".backup /path/to/backup/hausbuch.db"
```

## Development

Requirements: Go (the version in `go.mod`), [golangci-lint](https://golangci-lint.run/) v2, `make` and `curl`. The
Tailwind CSS CLI is downloaded by `make` into `bin/`.

```sh
make check   # lint, tests and build, as in CI
HAUSBUCH_DB_PATH=./hausbuch.db HAUSBUCH_LISTEN=127.0.0.1:8080 ./hausbuch
```

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
