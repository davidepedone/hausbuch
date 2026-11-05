# Tasks

## 1. Project skeleton, build and image

- [x] 1.1 Install Go (Homebrew), create the Go module with `cmd/hausbuch` and the `internal/` packages, and an HTTP server answering `/healthz`; verify `go build ./...` succeeds and `curl localhost:8080/healthz` answers
- [x] 1.2 Add the `Makefile` with `make check` (`go vet`, `golangci-lint`, `go test ./...`, `go build`); verify `make check` is green
- [x] 1.3 Open the SQLite database from `HAUSBUCH_DB_PATH` (WAL, busy timeout, foreign keys) and apply embedded goose migrations at start; verify a test that a fresh file gets the schema and a second start applies nothing
- [x] 1.4 Wire the Tailwind standalone CLI into `make check`, embed the compiled CSS, HTMX and the templates, and add a base responsive layout (header with app name, content area); verify the binary serves the CSS with no files next to it
- [x] 1.5 Add the Dockerfile (static build, distroless non-root, `/data` volume) and the `hausbuch healthcheck` subcommand used by `HEALTHCHECK`; verify `docker build` succeeds and the container reports healthy
- [x] 1.6 Add the GitHub Actions workflow: `make check` on every push, image published to `ghcr.io/davidepedone/hausbuch` (`latest` and commit SHA) on `main`; verify the workflow passes on the change's branch
- [x] 1.7 Write the README: what hausbuch is, how to run the image (example compose with a volume), the environment variables table, managing accounts with `docker exec … /hausbuch user …`, safe backups of the SQLite file, and that HTTPS is the deployer's responsibility; verify the example compose starts a working instance

## 2. Accounts and sign-in (`user-accounts`)

- [x] 2.1 Add the `app_user` migration and user store with argon2id hashing; verify tests that usernames equal ignoring case (including non-ASCII letters) are rejected and that a hash verifies only its own password
- [x] 2.2 Add the subcommand dispatch (`serve` as default, `healthcheck`, `user`) and the `user add` and `user list` commands, reading passwords without echo or from piped standard input; verify tests for the "Add an account", "Duplicate username", "List accounts" and both "Passwords never on the command line" scenarios
- [x] 2.3 Add the `session` migration and session handling (random token, stored hashed, cookie flags, expiry from `HAUSBUCH_SESSION_TIMEOUT`, enabled check on every request); verify tests for the "Long-lived sessions" scenarios with a controllable clock and that a session survives a restart of the server
- [x] 2.4 Build the login page, sign out, the middleware requiring a session everywhere except login, static assets and `/healthz`, and `http.CrossOriginProtection`; verify `httptest` tests for the "Login required everywhere", "Sign in with username and password", "No accounts yet" and "Sign out" scenarios and that a cross-origin POST is refused
- [x] 2.5 Lock a username for 15 minutes after 3 failed sign-ins in a row, counted on the user's row, and add `user unlock`; verify tests for the three "Repeated failed sign-ins lock the username" scenarios with a controllable clock, and that a lock survives a restart of the server
- [x] 2.6 Add `user set-password`, `user disable` (ending that user's sessions) and `user enable`, run against a database the server is using; verify tests for the "Set a password", "Disable an account" and "Enable an account" scenarios with the server running
- [x] 2.7 Build "change my password"; verify tests for both scenarios
- [x] 2.8 Check the login page (with and without accounts) and password change in a browser with Playwright at phone, tablet and desktop widths

## 3. Pages (`wiki-pages`)

- [x] 3.1 Add the `page`, `page_revision`, `page_alias` and `page_link` migrations and the page store; verify the migrations apply in `make check`
- [x] 3.2 Render Markdown with goldmark (GFM tables and task lists) with raw HTML escaped as text; verify unit tests for the "Formatted text in Markdown" scenarios, including a script tag and a `javascript:` link
- [x] 3.3 Build create and view page with the title rules and the language choice preset from `HAUSBUCH_DEFAULT_LANGUAGE`; verify tests for the "Create a page" and "Page language" scenarios
- [x] 3.4 Resolve `[[Title]]` and `[[Title|text]]` with the wikilink extension through titles and aliases, missing links styled and leading to creation; verify tests for the "Links between pages" scenarios
- [x] 3.5 Build edit and rename (alias kept for the former title), and rewrite `page_link` on save; verify tests for the "Edit a page" and "Pages linking here" scenarios
- [x] 3.6 Build the editor with a Markdown text area and HTMX preview; verify a test that the preview renders the same HTML as the page view
- [x] 3.7 Build the home page list, newest change first, with type filter; verify tests for the "List pages" scenarios
- [x] 3.8 Build delete, the deleted pages list (who, when, removal date) and restore; verify tests for the "Delete and restore a page" scenarios and the "Deleted pages list" scenario
- [x] 3.9 Add the daily cleanup removing pages deleted more than 30 days ago, with their history; verify tests for the "Removal after 30 days" and "Restored before removal" scenarios with a controllable clock
- [x] 3.10 Check viewing, editing (on a phone, by touch), list and deleted pages in a browser with Playwright at phone, tablet and desktop widths

## 4. Templates (`page-templates`)

- [x] 4.1 Add the page types and the embedded Recipe, How-to and Where is it skeletons in English and Italian; verify a test that each non-blank type's skeleton has its required headings in both languages
- [x] 4.2 Add type choice (Blank preselected) to page creation, prefilling the body in the page's language, swapping the skeleton on type or language change while the body is untouched, and type change on edit; verify tests for all `page-templates` scenarios
- [x] 4.3 Check type and language choice and the prefilled editor in a browser with Playwright at phone width, including a switch before and after typing

## 5. History (`page-history`)

- [x] 5.1 Append a revision on every save that changes title, type, language or body, and none otherwise; verify tests for the "Every save is a revision" scenarios
- [x] 5.2 Build the history list and the read-only revision view; verify tests for the "View the history of a page" scenarios
- [x] 5.3 Build the line diff between a revision and the current version; verify the "Compare" scenario test
- [x] 5.4 Build restore of a revision as a new revision; verify the "Restore" scenario test
- [x] 5.5 Reject saves based on an outdated revision with the conflict screen (user's text kept, who changed it, diff), the check running in a `BEGIN IMMEDIATE` transaction; verify tests for the "Concurrent edits do not overwrite each other" scenarios, including two saves racing in parallel
- [x] 5.6 Track open editors in memory (30-second activity signal, removed on save or cancel, ignored after 2 minutes) and warn on opening the editor with "go back" and "edit anyway"; verify tests for every "Warning when someone else is editing" scenario with a controllable clock
- [x] 5.7 Check history, diff, the editing warning and the conflict screen in a browser with Playwright at phone and desktop widths, with two users in two browser contexts

## 6. Search (`page-search`)

- [x] 6.1 Confirm a Go Snowball library with English and Italian stemmers (`blevesearch/snowballstem` first) and add the word splitting and stemming; verify unit tests that "descaling"/"descale" and "irrigazione"/"irrigare" share a stem and that "l'irrigazione" splits into two words
- [x] 6.2 Add the `page_fts` migration (stemmed title and body, unicode61 without diacritics) kept current on save, delete and restore in the same transaction; verify a test that a saved page is findable, a deleted one is not, a restored one is again
- [x] 6.3 Build search with each word stemmed both ways and quoted, title matches first then `bm25()`, excerpts built from the original body and the "nothing found" offer; verify tests for every `page-search` scenario, plus input containing FTS5 operators, quotes and HTML
- [x] 6.4 Put the search field in the header of every signed-in screen; verify the "Search from a page on a phone" scenario in a browser with Playwright at phone width

## 7. Integration check

- [ ] 7.1 Run the image from the README's example compose from scratch: two accounts added with `docker exec`, a lock lifted with `user unlock`, a page from each template, a link, a rename, an editing warning, a conflicting edit, a restore, a search, a container restart keeping the session; verify every step behaves as the specs say

## Workflow follow-up

- Open the pull request for the change's branch and merge it once reviewed.
- Archive the change with `/opsx:archive`, which moves its spec deltas into `openspec/specs/`.
