# Proposal

## Why

The home page's box points out what has expired or is about to, but only to whoever opens the app: an item expiring
next week is found out too late by a household that doesn't look. A notification on the phone, the day something
starts expiring, brings the warning to them. Over HTTPS, browsers can receive notifications from a web app (Web Push),
on iPhone too once the app is added to the Home Screen, so hausbuch can send them without a native app.

## What Changes

- **A daily summary of what is new**: once a day, at the time each user chooses, a notification lists the rows that
  started expiring or expired since that user's previous summary, in the sections they chose. When nothing is new, no
  notification is sent. Tapping it opens the home page.
- **Notifications on the account page**: a user turns notifications on or off for the device they are using, and
  chooses the time of the summary (09:00 unless changed) and which sections it covers (Medicines, Pantry and Supplies
  unless changed). The page explains that on iPhone and iPad hausbuch must first be added to the Home Screen.
- **One subscription per device**: notifications are turned on separately on each browser and device, and the
  choices of time and sections apply to all of a user's devices. Signing out on a device stops its notifications;
  disabling an account stops all of its notifications.
- **Only over a secure address**: notifications are offered under the same condition as passkeys, an HTTPS address in
  `HAUSBUCH_URL` (or `localhost`). Without one, no page mentions them.
- **End-to-end encrypted payloads**: what a notification says is encrypted for the receiving browser; the push
  service that delivers it (Apple's, Google's or Mozilla's) sees only that a notification arrived.
- README: what notifications need (HTTPS, the server reaching the browsers' push services), and that their keys are
  kept in the database, so a restored backup keeps existing subscriptions working.

### Out of scope

- Buttons in the notification (Used, Thrown away, Still there): Safari on iPhone ignores them. A later change, for the
  browsers that show them.
- Reminders on pages: they don't exist yet. When they do, they can be added to the same summary.
- Offline reading and caching pages in the service worker: a separate change for the app on the phone.
- A Settings page: the choices live on the account page for now and move there when one exists.
- Notifications by e-mail, Telegram or any other channel, and notifications for anything other than expiry.
- Different times for different sections, or more than one summary a day.

## Capabilities

### New Capabilities

- `push-notifications`: notifications sent to users' devices, their subscriptions, the daily expiry summary and the
  choices that shape it.

### Modified Capabilities

None. The summary uses the expired and expiring rows exactly as `expiry-warnings` defines them.

## Impact

- New dependency: a Web Push library implementing RFC 8291 encryption and VAPID (RFC 8292) authentication.
- Database: subscriptions per device, each user's choices of time and sections, the rows already notified, and the
  installation's VAPID key pair.
- `internal/stock`: the expired and expiring rows of chosen sections. A new notifications package: subscriptions,
  choices, building and sending the summary. `internal/web`: the account page's notifications section, subscribe and
  unsubscribe endpoints, the service worker at the site's root. `cmd/hausbuch`: the job that sends due summaries.
- README.
