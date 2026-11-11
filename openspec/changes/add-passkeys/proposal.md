# Proposal

## Why

hausbuch is about to be reachable from the internet. Passwords alone are then the weak point: they get reused and
guessed, and typing one on a phone is the slowest part of using the app. Passkeys sign in with the phone's own unlock,
can't be reused or phished, and need nothing to remember. Passwords stay as the fallback, because passkeys work only
over HTTPS and the image must keep working on a home server reached over plain HTTP.

## What Changes

- **Passkeys, when the installation has a public HTTPS address**: a new `HAUSBUCH_URL` setting gives the address users
  open. With an HTTPS address (or `localhost`), passkeys are offered; without one, nothing changes.
- **Signing in with a passkey** from the login page, with a button and the browser's own passkey suggestions in the
  username field; no username or password to type.
- **A passkey offered after a password sign-in**, to a user who has none: "Sign in faster next time", with "Not now"
  going on to where they were headed.
- **Own passkeys on the account page**: listed with when they were created and last used, another one can be added,
  any can be removed; the password keeps working.
- **A one-time sign-in link** printed by a new `user link` command, valid 15 minutes and once, to set up a passkey on a
  new phone or get back in without a password. It works over plain HTTP too.

### Out of scope

- Sign-in without a password for installations without HTTPS, or removing passwords altogether.
- Replacing the lock after 3 failed passwords with a slowdown per address, the request limit per address, and rules for
  new passwords: a separate change, planned next.
- Naming passkeys, or telling which device a passkey is on.
- Passkeys as a second factor after the password.
- Sending the sign-in link by e-mail or any other channel: whoever runs the command passes it on.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `user-accounts`: adds signing in with a passkey, managing one's own passkeys, the passkey offer after a password
  sign-in, and the one-time sign-in link with its `user link` command.

## Impact

- New dependency: `github.com/go-webauthn/webauthn`, for the WebAuthn ceremonies.
- Database: a table of passkeys, a table of sign-in links, and a WebAuthn user handle on each account.
- `internal/user`: storing passkeys and links. `internal/web`: the ceremonies' endpoints, the offer page, the account
  page's passkeys, the link page; a `passkeys.js` for the browser side. `cmd/hausbuch`: `HAUSBUCH_URL` and `user link`.
- README: `HAUSBUCH_URL`, and how to set up passkeys behind a proxy.
