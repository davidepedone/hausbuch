# Proposal

## Why

Nothing in the app says which release is running, so after an update nobody can tell whether it arrived or what
changed, and the people living in the home have no obvious way to report a problem or suggest an idea (issue #34).

## What Changes

- **A footer on every page of a signed-in user**: the running version, a "What's new" link and a "Report a problem"
  link.
- **The version** is the release's own, `YYYY.MM.N`, built into the binary by the release script; a build made any
  other way shows "dev".
- **"What's new"** opens that version's release on GitHub, with the notes the release script already generates; for
  "dev" it opens the list of releases.
- **"Report a problem"** opens the repository's choice of issue forms (bug or idea).
- Both are ordinary links, not new tabs: in the app added to a phone's home screen they open in a browser view over
  the app, and closing it returns to where the user was.

### Out of scope

- Release notes inside the app, or a page listing past releases.
- Telling users that a newer release exists.
- Reporting a problem without a GitHub account: the link leads to GitHub's own forms.
- The footer on the sign-in page, which shows nothing to whoever isn't signed in.

## Capabilities

### New Capabilities

- `app-version`: which release is running, and where to read what changed in it and to report a problem.

### Modified Capabilities

None.

## Impact

- `Dockerfile` (a `VERSION` build argument passed to the linker), `scripts/release.sh` (passing it),
  `cmd/hausbuch` (the version variable handed to the web handler), `internal/web` (the footer in the layout).
- README: where the version shows and how a build gets it.
