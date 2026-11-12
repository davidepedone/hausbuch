# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. Today expired and expiring rows
are computed on every read: `stock.Store.Warnings` returns them for the home page's box, with each row's state from
its effective expiry date and its section's lead time. Nothing is stored about them. A session is a random token in a
cookie, stored as its SHA-256 in `session`; signing out, disabling an account and the daily clean-up delete session
rows. Passkeys are enabled by `web.PasskeyConfig` when `HAUSBUCH_URL` is HTTPS or `http://localhost`. Background work
runs as goroutines started by `serve` (the daily backup, the clean-up). The account page is `/account/password`.

Web Push: the browser's `PushManager.subscribe` returns an endpoint URL at its vendor's push service and two keys; the
server sends each message to the endpoint, encrypted for the browser (RFC 8291) and signed with the server's VAPID key
(RFC 8292). Messages need a service worker to be shown. On iPhone and iPad, `PushManager` exists only in a web app
opened from the Home Screen.

## Goals / Non-Goals

**Goals:**

- No new setting to configure: notifications work as soon as `HAUSBUCH_URL` is HTTPS.
- Nothing readable by the push services; nothing in the log that names what a household keeps.

**Non-Goals:**

- Delivery guarantees: a message the push service drops is not retried beyond its TTL.
- Testing the browser's subscription in automated tests: headless browsers can't reach a real push service.

## Decisions

### `SherClockHolmes/webpush-go` for encryption and VAPID

It implements RFC 8291 (`aes128gcm`) and VAPID, MIT-licensed, and depends only on `golang-jwt/jwt/v5` and
`golang.org/x/crypto`, both already in the module graph.

*Alternative considered*: writing the encryption on the standard library (`crypto/ecdh`, `crypto/hkdf`, AES-GCM, an
ES256 JWT). Around 150 lines, but cryptography where a mistake is silent; a small, used library is the safer bet.

### The VAPID key pair is generated once and kept in the database

At start, if the `push_key` table is empty, a P-256 key pair is generated and stored. The VAPID subject is
`HAUSBUCH_URL`. Keeping the key in the database means backups and restores carry it, and existing subscriptions keep
working after a restore or a move; subscriptions are bound to the public key, so a new key would silently break them.

*Alternative considered*: a `HAUSBUCH_VAPID_*` environment variable. One more secret for whoever deploys to generate
and keep, for no benefit to a single installation.

### Subscriptions belong to a session

`push_subscription (endpoint PRIMARY KEY, session_hash REFERENCES session ON DELETE CASCADE, user_id, p256dh, auth,
created_at)`. Turning notifications on stores the browser's subscription with the current session's hash; turning
them off deletes it. Because the session row is deleted by signing out, by disabling the account and by the clean-up
of sessions ended unused, the cascade gives the spec's "Notifications stop with the device's sign-in" without code in
any of those paths. A second subscription from the same endpoint replaces the first.

*Alternative considered*: subscriptions belonging to the user, deleted on sign-out. Then a device whose session ended
unused would keep receiving names of what a household keeps, and every path that ends a session would have to
remember the subscriptions.

### Choices and what was told, per user

`push_choice (user_id PRIMARY KEY, minutes, medicines, pantry, supplies, last_day)`: the summary's time as minutes
after midnight, one flag per section, and the last day a summary was due (`YYYY-MM-DD` in the installation's time
zone). A missing row means the defaults: 540 (09:00), all sections, never due. `push_told (user_id, item_id, state,
PRIMARY KEY (user_id, item_id))`: the state, expiring or expired, each row was last part of a summary in.

When a user's summary is due, the job takes `Warnings` restricted to the chosen sections; the new rows are those whose
id is not in `push_told` with the same state. Then `push_told` for that user is replaced by the current rows and
states, which also forgets the rows that stopped being expiring or expired, or whose section was unchecked, as the
spec's "A row told again after a change" asks. Both happen whether or not anything was new; a summary is sent only
when something was. `push_told` rows go with their stock row (`ON DELETE CASCADE`).

*Alternative considered*: comparing with the warnings at the previous summary's time, recomputed. Lead times and dates
change in between, so the past can't be recomputed reliably.

### One job, every minute

`serve` starts a goroutine that every minute, for each user with at least one subscription, checks whether today's
time has come and `last_day` is not today; if so it builds and sends the summary and sets `last_day` to today. A
server started after the time sends that day's summary at its first check; a whole day without the server is never
made up, and what was new is still new. Turning notifications on after today's time therefore sends a first summary
within a minute when something is already expiring, which also shows the user it works.

*Alternative considered*: one timer per user. Simpler to reason about the minute loop, and it costs one query a
minute.

### What a message carries

The payload is JSON with a title ("2 things to check") and a body naming rows as "Yogurt (Pantry, expired)", expired
first then by effective expiry date, as many as fit in 3,000 bytes, then "and 12 more". Push services accept around
4 KB after encryption. Messages are sent with a TTL of 24 hours, urgency `normal` and the topic `summary`, so that a
summary still waiting for an offline phone is replaced by the next one instead of piling up. A 404 or 410 from the
push service deletes that subscription; other failures are logged and left for the next day.

### The service worker at `/sw.js`

A service worker controls only the paths under its own URL, so it is served at `/sw.js`, from the embedded static
files, without a session like the rest of `/static/`: it is the same for everyone and holds no data. It does two
things: on `push` it shows the notification with the payload's title and body and the app's icon; on
`notificationclick` it focuses an open hausbuch window or opens `/`. No `fetch` handler: nothing is cached.

### The account page section

Rendered only when passkeys are enabled. It holds a form for the time (`<input type="time">`) and the three section
checkboxes, posted to `POST /account/notifications`, and a `notifications.js` that decides what to show: without
`serviceWorker`, `PushManager` or `Notification` it shows the Home Screen explanation; otherwise it registers
`/sw.js`, compares the browser's current subscription with the endpoints the server lists for this session (rendered
into the page), and shows "Turn on" or "Turn off". Turning on asks for permission, subscribes with the VAPID public key
and posts the subscription to `POST /account/notifications/subscribe`; turning off unsubscribes and posts the endpoint
to `POST /account/notifications/unsubscribe`. Both endpoints act only on the current session's subscriptions.

## Risks / Trade-offs

- [A device's notifications stop after the session timeout without use] → Choosing a notification opens hausbuch,
  which refreshes the session, so a household that reads its summaries stays subscribed. The README says so.
- [The server can't reach a push service, behind a firewall that blocks outgoing connections] → The failure is logged
  with the device count; the README says the server must reach the browsers' push services over HTTPS.
- [A row's names are in the stored subscriptions' messages at the push service until delivered] → They are encrypted
  for the browser; the push service holds ciphertext only.
- [Losing the database loses the VAPID key, and every device must turn notifications on again] → The key is in the
  backups like everything else.
- [Many users due at the same minute] → Summaries are sent one after another; a household has a handful of users.

## Migration Plan

One migration adds `push_key`, `push_subscription`, `push_choice` and `push_told`; nothing existing changes. Rolling
back to the previous release leaves the tables unused. The key pair is created at the first start of the new release.
