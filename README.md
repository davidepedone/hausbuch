# hausbuch

A private household wiki for the people living in a home: recipes, how-to procedures, seasonal chores, where things
are kept. Pages start from templates, can carry photos and files, and can have reminders that show up in the app when
they are due. Three sections, Medicines, Pantry and Supplies, keep track of what the household has at home, and the
home page points out what has expired or is about to. A shared shopping list collects what runs out, and what was bought
is put away into its section.

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

The footer of every page shows the running release, with links to its release notes and to report a problem; an image
built other than by `scripts/release.sh` (which passes `--build-arg VERSION=…`) shows `dev`.

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
| `HAUSBUCH_BACKUP_DIR` | directory the copies of the database are saved in (see [Backups](#backups)) | `backups` next to the database file |
| `HAUSBUCH_BACKUP_KEEP` | how many daily copies to keep, at least 1 | `7` |
| `HAUSBUCH_MEDICINES_LEAD_DAYS` | how many days before their expiry date medicines are pointed out as expiring, at least 0 | `30` |
| `HAUSBUCH_PANTRY_LEAD_DAYS` | the same for the pantry | `7` |
| `HAUSBUCH_SUPPLIES_LEAD_DAYS` | the same for household supplies | `7` |
| `HAUSBUCH_URL` | the address users open, such as `https://home.example.org`; with an HTTPS address users can sign in with passkeys (see [Passkeys](#passkeys)), and `user link` prints whole links | none: passwords only |
| `HAUSBUCH_TRUSTED_PROXIES` | the reverse proxies in front of hausbuch, as addresses or ranges separated by commas, such as `172.16.0.0/12` for one in a Docker network; their `X-Forwarded-For` header gives the client's address, which the request limit, the sign-in lock and the log go by. Set it behind a proxy, or every request counts as coming from the proxy | none: the connection's address |
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
docker exec -it hausbuch /hausbuch user unlock anna              # after 3 wrong passwords in a row from one address
docker exec -it hausbuch /hausbuch user link anna                # a one-time sign-in link
```

Replace `hausbuch` after `-it` with the container's name. Passwords are never given as arguments; without `-t` (no
terminal) they are read from standard input instead, e.g. `echo "$PASSWORD" | docker exec -i …`.

A new password needs 15 to 256 characters, with no required mix of capitals, digits or symbols: a few words with
spaces between them do well. Common passwords and the account's own username or display name are refused. Passwords
don't expire, and the rules apply only when a password is set, so existing ones keep working.

After 3 wrong passwords in a row from one address, a username is locked from that address for 15 minutes, even with
the right password; its user can still sign in from elsewhere. `user unlock` lifts the locks from every address.

`user link` prints a link that signs the account in once, within 15 minutes; printing another one makes the earlier
one invalid. Pass it on to set up a passkey on a new phone, or to get someone back in without resetting their
password. Opening the link only shows whose account it is, with a button to sign in, so that a messenger's link
preview doesn't use it up. Without `HAUSBUCH_URL` only its path is printed, to be appended to the usual address.

### Passkeys

With `HAUSBUCH_URL` set to an HTTPS address, users can sign in with a passkey, their device's fingerprint, face or
screen lock, instead of the password. After signing in with the password, a user without a passkey is offered to
create one; the account page lists their passkeys and adds or removes them. The password keeps working.

Browsers allow passkeys only over HTTPS, or on `localhost`, so an installation reached over plain HTTP keeps signing
in with passwords alone. A passkey belongs to the host name it was created under: set `HAUSBUCH_URL` once, to the
address users actually open through the proxy, and don't change it, or the passkeys already created stop working
(passwords and sign-in links don't).

### HTTPS

hausbuch serves plain HTTP. Passwords and session cookies travel unencrypted unless a reverse proxy in front of it
terminates HTTPS; setting that up is up to whoever deploys it. Behind a proxy, set `HAUSBUCH_TRUSTED_PROXIES` too.

### Sign-in protection and the log

An address can make at most 20 requests a minute to the login page, sign-in links and passkey sign-in; over that it is
told to try again in a minute. At most 2 passwords are checked at a time, so that a burst of sign-ins can't exhaust the
server's memory; one that waits more than 5 seconds is told to try again. An IPv6 address counts as its `/64`.

The container's log (`docker logs hausbuch`) has a line for every sign-in and failed sign-in, with the method, the
account and the address; for a lock starting; for requests refused by either limit; for a sign-in link that wasn't
valid; and for passkeys added or removed and passwords changed. It never holds passwords or link tokens, nor usernames
typed that match no account. Addresses are personal data: keep the log's size limited, as the `logging` options of
[`docker-compose.yml`](docker-compose.yml) do.

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

Requirements: Go (the version in `go.mod`), [golangci-lint](https://golangci-lint.run/) v2, `make`, `curl`, and
Chrome or Chromium, which the tests of the pages' scripts drive. The Tailwind CSS CLI is downloaded by `make` into
`bin/`.

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

## License

hausbuch is licensed under the GNU Affero General Public License v3.0, see [LICENSE](LICENSE). If you run a
modified version for others to use over a network, you must offer them its source code.
