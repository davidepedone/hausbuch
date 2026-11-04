# Design

## Context

The repository is empty apart from the OpenSpec setup: there is no existing code to fit. See `proposal.md` for the
scope and `specs/` for the required behaviour. Constraints that shape the approach:

- A handful of users and at most a few thousand pages: performance is not a driver, simplicity of operation is.
- The product is one Docker image configured through environment variables, updated by pulling a new image and
  restarting the container. Sessions must survive that restart (they last 30 days by default).
- Every resident can write every page, so page content is untrusted input for every other resident's browser.

## Goals / Non-Goals

**Goals:**

- One small, self-contained container: no database server, no runtime besides the binary.
- Server-rendered pages that work well on a phone without a JavaScript application.
- Few dependencies, each doing something the standard library doesn't.
- A data model that later changes (attachments, reminders) can extend without reshaping pages and revisions.

**Non-Goals:**

- A rich-text (WYSIWYG) editor: Markdown in a text area with a preview.
- Real-time collaborative editing: concurrent edits are detected on save, not merged live.
- Horizontal scaling or several application instances: SQLite and the in-memory sign-in lock assume one process.

## Decisions

### Build it, rather than deploying an existing wiki

Self-hosted wikis such as BookStack or Wiki.js already offer accounts, revisions, search and attachments. They don't
fit the parts of hausbuch that matter most: content types with skeletons are weak or absent, and reminders shown in
the app are not there at all. Bending one of them to that, and keeping it updated, costs more than a small app built
for exactly this.

### Go, server-rendered with `html/template` + HTMX, Tailwind CSS

Go's standard library covers most of what this change needs: HTTP routing with method and path patterns
(`net/http.ServeMux`), HTML templates that escape by context (`html/template`), cross-origin request protection
(`http.CrossOriginProtection`), and embedding templates and assets into the binary (`embed`). The result is a single
static binary in an image of a few megabytes, with low memory use and instant start.

Pages are rendered on the server; HTMX handles the few dynamic bits (editor preview, search as you type). Tailwind,
compiled by its standalone CLI before `go build`, keeps the responsive layout simple. Both HTMX and the compiled CSS
are embedded, so the app needs nothing from the internet.

Alternatives considered:
- *Kotlin + Spring Boot*: everything ready-made, but a JVM using several hundred megabytes and a slow start for a
  household wiki, plus a database server next to it.
- *Node.js (TypeScript) + Fastify*: light enough, but it needs a runtime and `node_modules` in the image, and
  sessions, CSRF and the like come from several plugins.

### SQLite, embedded, with `modernc.org/sqlite`

SQLite is a single file in the container's data volume: no server, no second container, transactions for concurrent
edits, and full-text search through FTS5. `modernc.org/sqlite` is a pure-Go driver, so the binary builds with
`CGO_ENABLED=0` and runs on a minimal image.

The database runs in WAL mode with a busy timeout and foreign keys on. Writes that check-then-change (saving a page,
the last-administrator rule) run in `BEGIN IMMEDIATE` transactions, so two requests can't both pass the check.

Alternatives considered:
- *PostgreSQL*: better tooling at scale, but a second container to run and back up, for no gain at this size.
- *Plain Markdown files on disk, with git for history*: readable without the app, but users, sessions, aliases,
  backlinks, deleted pages and atomic saves would all have to be rebuilt by hand.

Schema changes are numbered SQL files, embedded and applied at start with `pressly/goose`; from the first release on
they are only ever added, never edited.

### Data model

- `app_user`: username, `username_key` (lower-cased, unique: SQLite's `NOCASE` only folds ASCII), display name,
  password hash, administrator flag, enabled flag.
- `session`: hash of the session token, user, created at, expires at. Only the hash is stored, so a copy of the
  database does not let anyone sign in.
- `page`: current title, `title_key` (title trimmed and lower-cased, unique), type, current revision number,
  `deleted_at`.
- `page_revision`: page, revision number (1, 2, …), title, type, body, author, created at. Append-only.
- `page_alias`: former `title_key`s of a page, so `[[Old title]]` keeps resolving after a rename. A title that is an
  alias of another page counts as taken; renaming a page back to one of its own aliases removes the alias.
- `page_link`: from page → linked `title_key`, rewritten on every save of the linking page; it powers "pages linking
  here" and nothing else.
- `page_fts`: FTS5 table with the current title and body of pages that are not deleted, updated in the same
  transaction as every save, delete and restore.

Deleting sets `deleted_at` and removes the page from `page_fts`; restoring does the opposite. Revisions are never
deleted, which is what makes restore and history trivially correct.

### Authentication and sessions, written in the app

Sessions are a random 256-bit token in an `HttpOnly`, `SameSite=Lax` cookie (`Secure` when the request arrives over
HTTPS), looked up by hash in `session`, with the user's enabled flag checked on every request. Keeping them in our
own table makes "end all sessions of a disabled user" one `DELETE`, and they survive restarts. Expired sessions are
removed at sign-in and once a day.

Passwords are hashed with argon2id (`golang.org/x/crypto/argon2`), parameters stored with each hash. Cross-site
request forgery is blocked by `http.CrossOriginProtection`, which rejects state-changing requests from other origins
using the browser's `Sec-Fetch-Site`/`Origin` headers, with no tokens in forms. Failed sign-ins are counted per
username in memory.

Alternative considered: *`alexedwards/scs` for sessions*. Mature, but ending every session of one user needs
iterating the whole store; our table is shorter than the glue around it.

### Markdown rendering

`yuin/goldmark` with its GFM extension (tables, task lists). Goldmark already refuses dangerous link schemes such as
`javascript:`; raw HTML is rendered by a small custom renderer that escapes it, so it shows as the text the user
typed instead of being dropped. `[[Title]]` and `[[Title|text]]` come from `go.abhg.dev/goldmark/wikilink`, with a
resolver backed by `page` and `page_alias` and a renderer that marks missing pages, so links follow renames,
deletions and restores without rewriting other pages.

### Templates

Page types are a fixed set (Blank, Recipe, How-to, Where is it); each non-blank type has an embedded Markdown skeleton
(`pagetypes/<type>.md`). Kept in code on purpose: creating templates from the UI is a later change and will introduce
its own storage.

### Concurrent edits

The edit form carries the revision number it started from. Saving, inside a `BEGIN IMMEDIATE` transaction, compares
it with the current one and either appends revision n+1 or answers with a conflict screen: the user's text stays in
the editor, the screen names who saved meanwhile and shows the diff between the version they started from and the
current one. No three-way merge: with a few users editing rarely, conflicts are rare and resolving them by hand is
enough.

### Search

FTS5 with the `porter unicode61 remove_diacritics 2` tokenizer gives English stemming and ignores case and accents.
The user's words are each quoted before being passed to `MATCH`, so FTS5 operators in the input are taken literally
and all words are required. Ordering puts pages whose title matches first, then `bm25()` rank; `snippet()` gives the
excerpt.

### History diff

Line-based diff of the Markdown source between two revisions (`sergi/go-diff` in line mode), shown as added/removed
lines. A diff of the source, not the rendered HTML, is both simpler and what the editor shows.

### Code layout

One module, `cmd/hausbuch` for `main`, and packages by domain under `internal/` (`user`, `page`, `search`,
`markdown`, `web` for handlers and templates, `db` for connection and migrations). Handlers depend on small
interfaces of the domain packages, so they are tested through `httptest` against a real SQLite file.

### Configuration

| Variable | Purpose | Default |
| --- | --- | --- |
| `HAUSBUCH_DB_PATH` | SQLite database file | `/data/hausbuch.db` |
| `HAUSBUCH_LISTEN` | listen address | `:8080` |
| `HAUSBUCH_ADMIN_USERNAME`, `HAUSBUCH_ADMIN_PASSWORD` | first administrator, used only when no account exists | none |
| `HAUSBUCH_SESSION_TIMEOUT` | session duration, as a Go duration | `720h` (30 days) |

### Build, image and CI

A `Makefile` with one `make check` target runs the whole build: Tailwind, `go vet`, `golangci-lint`, `go test
./...`, `go build`. Browser checks use Playwright against a locally running binary. The Dockerfile builds the static
binary in a Go image and copies it into `gcr.io/distroless/static` running as non-root, with `/data` as the volume.
The image has no shell or `wget`, so the health check is the binary itself (`hausbuch healthcheck`, calling
`/healthz`). GitHub Actions runs `make check` on every push; pushes to `main` also publish the image to
`ghcr.io/davidepedone/hausbuch` tagged `latest` and with the commit SHA.

## Risks / Trade-offs

- [Passwords and session cookies travel unencrypted when the app is served over plain HTTP] → Out of the app's
  control; the README states that HTTPS is the deployer's responsibility.
- [Every resident can write HTML-looking content seen by everyone] → HTML is escaped at render time and dangerous link
  schemes are refused; covered by tests on the `wiki-pages` HTML scenario.
- [Brute-forcing a password from the local network] → argon2id hashes, and the 5-failure lock from `user-accounts`.
  The counters live in memory and reset on restart: kept deliberately simple.
- [Copying the SQLite file while the app writes can produce a broken backup] → The README documents a safe backup
  (`sqlite3 … ".backup"` or stopping the container first).
- [Hand-written session handling is security-sensitive code] → Kept small, and every `user-accounts` scenario plus
  token handling (expiry, disabled users, cookie flags) is covered by tests.

## Migration Plan

First release: nothing to migrate. Migrations create the schema on first start and the first administrator is
created from the environment. Rolling back means running the previous image; from this change on, migrations are only
ever added, never edited.
