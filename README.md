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
    image: ghcr.io/davidepedone/hausbuch:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - hausbuch-data:/data

volumes:
  hausbuch-data:
```

The container runs as the unprivileged user `65532`. With a named volume, as above, Docker gives it the right owner.
With a bind mount (`./data:/data`), the host directory must be writable by that user:
`chown 65532:65532 ./data`.

The image reports its health to Docker by itself (`/hausbuch healthcheck`, calling `/healthz`).

### Configuration

| Variable | Purpose | Default |
| --- | --- | --- |
| `HAUSBUCH_DB_PATH` | SQLite database file | `/data/hausbuch.db` |
| `HAUSBUCH_LISTEN` | listen address | `:8080` |

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
