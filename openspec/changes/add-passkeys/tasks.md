# Tasks

## 1. Storage and configuration

- [x] 1.1 Add the migration (`app_user.webauthn_id`, `passkey`, `sign_in_link`) and verify its Down with a throwaway
  test
- [x] 1.2 Add sign-in links to `internal/user` and the `user link` command; verify tests for "Print a sign-in link",
  "Link used twice", "Link expired", "Newer link printed", a disabled account, and the link printed with and without
  `HAUSBUCH_URL`
- [x] 1.3 Add passkey storage to `internal/user` (add, list, remove, find by credential id and by user handle, update the
  counter and last use) and `HAUSBUCH_URL` parsing, with passkeys enabled only for HTTPS or `localhost`; verify tests,
  including "Plain HTTP installation" for the parsing

## 2. Web interface

- [x] 2.1 Add the link pages; verify web tests for "Sign in with a link", "Opening is not signing in", the expired and
  used link pages, and a disabled account
- [x] 2.2 Add `go-webauthn`, the ceremony endpoints, `passkeys.js`, the login page's button and suggestions, the offer
  page and the account page's passkeys; verify web tests for "Plain HTTP installation", "No offer with a passkey",
  removing another user's passkey being refused, and Chrome tests with a virtual authenticator for "Offer after the
  password", "Passkey sign-in", "Locked username", "Disabled account", "Passkey cancelled", "Add a passkey" and "Remove
  a passkey"

## 3. Documentation and integration

- [x] 3.1 Document `HAUSBUCH_URL`, passkeys and `user link` in the README
- [x] 3.2 With `make check` green, in Chrome at 390px and at desktop width with a virtual authenticator, go through the
  login page, the offer, the account page and a link page; verify nothing overflows. Note that a real phone can only be
  tried once the installation has HTTPS

## Workflow follow-up

- Archive the change once merged.
