# Proposal

## Why

hausbuch is about to be reachable from the internet, and its sign-in was built for a home network. Today 3 wrong
passwords lock a username for everyone, so a stranger can keep a resident locked out just by typing their username;
nothing limits how often an address may try; a burst of sign-ins, each checking a password with 19 MiB of argon2id,
can exhaust a small server's memory; any non-empty password is accepted; and nothing is logged when someone signs in or
fails to, so an attack would go unnoticed.

## What Changes

- **A request limit per address** on what can be reached without signing in: the login page and form, sign-in links and
  the passkey sign-in endpoints. Over the limit the server answers "Too many attempts" for a minute. IPv6 addresses
  count per `/64`, the block a single home or server usually gets.
- **The lock after 3 wrong passwords applies to a username from one address**, not to the username everywhere: the
  resident still signs in from their own devices while someone elsewhere is locked out. **BREAKING** for `user list`,
  which shows a username as locked when it is locked from any address, and `user unlock`, which lifts all of them.
- **At most 2 password checks at a time**: further sign-ins and password changes wait their turn for a few seconds, then
  are told to try again, instead of the server running out of memory.
- **The client's address behind a proxy**: a new `HAUSBUCH_TRUSTED_PROXIES` setting lists the proxies whose
  `X-Forwarded-For` header is believed. Without it the address is the connection's, so nobody can pick their own.
- **Rules for new passwords**, after NIST SP 800-63B: 15 to 256 characters, no required mix of letters and symbols, no
  expiry; common passwords and the account's own username or display name are refused. They apply when a password is
  set, from the `user` commands or the account page; existing passwords keep working.
- **Security events in the log**: signing in or failing to, with the method and the address; a lock starting; the
  request limit or the password-check limit being hit; a sign-in link used, expired or already used; a passkey added or
  removed; a password changed. Never a password, a token or a link.
- README: the new setting, the password rules, and the log: what it records, and keeping its size limited, as the
  example `docker-compose.yml` already does.

### Out of scope

- CAPTCHAs, fail2ban-style blocking at the proxy, IP reputation lists, geoblocking.
- A command or page summarising security events; alerts to whoever runs the installation (they come with web push).
- Removing passwords, or requiring a passkey.
- Asking residents with an existing short password to change it.
- Request limits on pages that need a session.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `user-accounts`: the sign-in lock moves to username and address; adds the request limit per address, the limit on
  concurrent password checks, the client address behind a trusted proxy, the rules for new passwords, and the logging
  of security events.

## Impact

- Database: a table of failed sign-ins per username and address replaces `app_user.failed_sign_ins` and
  `locked_until`.
- `internal/user`: the lock, the password rules with a bundled list of common passwords, and the limit on concurrent
  password checks. `internal/web`: the client address, the request limit, the log lines, the password hints on the
  account page. `cmd/hausbuch`: `HAUSBUCH_TRUSTED_PROXIES`, the rules' messages in `user add` and `user set-password`.
- README.
- No new dependency.
