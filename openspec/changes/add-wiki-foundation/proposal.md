# Proposal

## Why

The knowledge a household runs on — recipes, how to descale the coffee machine, when to turn the irrigation on, where
things are kept — lives in people's heads or nowhere. hausbuch gives every resident one private place to write it down
and find it again. This first change builds the foundation every later feature rests on: accounts, pages, history and
search.

## What Changes

- Residents sign in with their own account; nothing but the login page is reachable without signing in. Accounts are
  created and managed with commands of the application binary, run inside the container; there is no
  self-registration and no administrator role.
- Residents create, read, edit and delete pages written as formatted text, with links between pages. Deleted pages
  can be restored.
- A new page starts from a template chosen by content type (Recipe, How-to, Where is it, or Blank). A template is a
  text skeleton defined in the application; the page keeps its type so pages can be listed by type.
- Every saved edit is kept as a revision recording who made it and when. Any resident can view an older revision,
  compare it with the current one and restore it. Two residents editing the same page at once cannot silently
  overwrite each other.
- Full-text search over page titles and content.
- A responsive UI in English that works on desktop, tablet and phone.
- The application ships as a generic Docker image configured through environment variables.

### Out of scope

- Attachments (photos and files) — a later change.
- Reminders and in-app notifications — a later change.
- Creating or editing templates from the UI — a possible later change.
- Structured fields for content types (e.g. searchable ingredient lists): pages stay free text.
- Access from outside the home network, HTTPS termination, email or push notifications.
- Languages other than English; per-page permissions (every resident can edit every page).
- Account management in the web interface, and deleting accounts (they can only be disabled).

## Capabilities

### New Capabilities

- `user-accounts`: residents' accounts, signing in and out, sessions, the sign-in lock, and account management from
  the command line.
- `wiki-pages`: creating, viewing, editing, listing, deleting and restoring pages; formatted text and links between
  pages.
- `page-templates`: the content types a page can be created from and the skeleton each one provides.
- `page-history`: revisions of a page, who changed what, comparing and restoring revisions, protection against
  concurrent edits overwriting each other.
- `page-search`: full-text search over pages.

### Modified Capabilities

None: this is the first change.

## Impact

- New application code, build and Docker image; the stack is proposed in `design.md`.
- A database holding accounts, pages and revisions; its location comes from an environment variable.
- README documents the image, its environment variables, how to run it and how to manage accounts.
