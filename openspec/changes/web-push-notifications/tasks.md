# Tasks

## 1. Storage and summaries

- [x] 1.1 Add the migration (`push_key`, `push_subscription` cascading from `session`, `push_choice`, `push_told`
  cascading from `stock_item`) and verify its Down with a throwaway test
- [x] 1.2 Add `webpush-go` and a notifications package: the key pair created once at start, subscriptions (add
  replacing the same endpoint, remove, list per session and per user), choices with their defaults; verify tests for
  the key surviving a restart, "Defaults", and subscriptions gone after signing out, after a session ends unused and
  after disabling the account ("Sign out", "Session ended unused", "Disabled account")
- [x] 1.3 Add the summary: new rows from `Warnings` in the chosen sections against `push_told`, the told rows replaced,
  the payload's title and body with the size limit; verify tests for "Something started expiring", "Nothing new",
  "Expiring, then expired", "Section not chosen", "First summary", "No section chosen", "Date pushed back, then due
  again", "Two rows" and "Many rows"
- [x] 1.4 Add sending: VAPID-signed, encrypted requests with TTL, urgency and topic, a 404 or 410 deleting the
  subscription, and the log lines; verify tests against a fake push service for "Sent payload" (no row or section
  name in the request body, and the body decrypts to the payload with the subscription's keys), "Subscription gone"
  and "Summary sent"
- [x] 1.5 Add the job in `cmd/hausbuch`, run every minute with a replaceable clock; verify tests for a summary at the
  chosen time, "Changed choices", "Started late" and "Day missed"

## 2. Web interface

- [x] 2.1 Serve `/sw.js` without a session, showing the payload on `push` and opening or focusing `/` on
  `notificationclick`; verify a web test that it is served without a session with a JavaScript content type, and a
  Chrome test that the worker registers with scope `/`
- [x] 2.2 Add the account page's "Notifications" section, `POST /account/notifications` for the choices, the subscribe
  and unsubscribe endpoints acting only on the current session, and `notifications.js`; verify web tests for "Plain
  HTTP installation", "HTTPS installation", saving the choices, and unsubscribing another session's endpoint being
  refused, and Chrome tests for the Home Screen explanation when `PushManager` is missing ("Safari tab on iPhone") and
  for "Permission refused" with notifications denied
- [x] 2.3 Document notifications in the README: they need an HTTPS `HAUSBUCH_URL`, the server must reach the browsers'
  push services over HTTPS, the key is in the database and travels with backups, a device's notifications stop when
  its session ends, and on iPhone and iPad hausbuch must be opened from the Home Screen

## 3. Integration

- [x] 3.1 With `make check` green, in Chrome at 390px and at desktop width, go through the account page's section on
  and off with `localhost`, and verify nothing overflows
- [ ] 3.2 On a real iPhone added to the Home Screen and on a desktop browser, against an HTTPS installation running a
  candidate release: turn notifications on, make a row expiring, receive the summary at the chosen time, open it on the
  home page, sign out and verify nothing more arrives. Note in the PR what was tried on which device

## Workflow follow-up

- Archive the change after the PR is merged.
