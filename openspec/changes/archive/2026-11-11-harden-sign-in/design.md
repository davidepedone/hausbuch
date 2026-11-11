# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. Today `user.Store.Authenticate`
checks the password with argon2id (19 MiB, 2 passes) and counts failures in `app_user.failed_sign_ins` and
`locked_until`; an unknown username is hashed against a dummy hash so it takes as long. The server never looks at the
client's address: `RemoteAddr` and `X-Forwarded-For` are unused, only `X-Forwarded-Proto` is read, to mark cookies
`Secure`. The open routes are `GET`/`POST /login`, `GET`/`POST /link/{token}` and `POST /login/passkey/begin` and
`/finish`. The log is the default `slog` handler and has only errors and start-up. Passwords are checked to be
non-empty, nothing else. There is one server process.

## Goals / Non-Goals

**Goals:**

- Every limit is in the application, so it holds behind any proxy or none.
- No limit can be used to keep a resident out: they can always sign in from their own address, with a passkey, or with
  a link.

**Non-Goals:**

- Stopping a distributed guessing attack by counting: with many addresses that only the password rules defeat.
- Sharing limits between processes: there is one.

## Decisions

### The address of a request

A function in `internal/web` returns the request's address: the host of `RemoteAddr`, or, when that is inside one of
the prefixes of `HAUSBUCH_TRUSTED_PROXIES` (comma-separated addresses or CIDR ranges, parsed at start-up; an invalid
entry stops the server, like the other settings), the rightmost `X-Forwarded-For` entry that is not itself a trusted
proxy. A malformed header falls back to `RemoteAddr`. IPv4-mapped IPv6 addresses are unmapped. The address's key,
used by the limit, the lock and the log, is the IPv4 address or the IPv6 `/64` prefix.

*Alternative considered*: always believing `X-Forwarded-For`. Anyone could then pick an address per request, and both
the limit and the lock would count nothing.

### The request limit: fixed one-minute windows in memory

A small limiter counts requests per address key in the current minute, with a mutex and a map that is emptied when the
minute changes, so it can't grow beyond one minute's addresses. It wraps the open routes only. A refused request gets
429: JSON with an `error` message for the passkey endpoints, which `passkeys.js` already shows, and the login page with
the message otherwise. The passkey suggestions the login page asks for on its own give up silently when they can't
start, so a login page over the limit says so once, not again under the passkey button. The clock is a field, replaceable in tests. 20 a minute leaves room for a household behind one
address and a user who mistypes, and is small enough to make guessing from one address pointless.

*Alternative considered*: a token bucket (`golang.org/x/time/rate`). It smooths bursts at the window's edge, which
doesn't matter at this size, and adds a dependency.

### The lock per username and address, in the database

A migration creates `sign_in_failure (username_key, address, failures, locked_until, last_failure_at)`, keyed by the
first two, and drops `app_user.failed_sign_ins` and `locked_until`; locks running at the upgrade are lost, at most 15
minutes of them. `Authenticate` takes the address key: a wrong password for an existing account increments its row and
starts the lock at the third; a right one deletes the row. Failures older than 15 minutes, or from before a lock that
has ended, start the count again, and the row is read and written in one transaction so that concurrent attempts
can't share a count. Unknown usernames are not recorded, as their refusal can't be told apart anyway, so the table
can't be filled with made-up names. `Unlock` deletes the username's rows. `List` shows the latest lock of each
account, so `user list` reads as before. The daily clean-up deletes the rows that no longer count.

*Alternative considered*: keeping the lock in memory. Simpler, but the spec requires it to survive a restart, and a
restart would otherwise give an attacker fresh attempts.

### At most 2 argon2id computations at a time

A buffered channel of 2 in `internal/user` guards every argon2id computation, hashing or checking, including the dummy
one for unknown usernames. Waiting takes the request's context with a 5-second timeout; on timeout the caller gets
`ErrBusy`, and `Authenticate` returns before touching the failure count. The web answers 503 with "Too many sign-ins
at once. Try again in a moment." on the login and account pages. Two computations are under 40 MiB and use two cores;
on one core they share it.

*Alternative considered*: lowering argon2's memory. It weakens every stored password to protect against a burst that
the queue handles.

### Password rules in `internal/user`

`checkNewPassword(password, username, displayName)` is called by `Add`, `SetPassword` and `ChangePassword` before
hashing, returning one of `ErrPasswordTooShort`, `ErrPasswordTooLong`, `ErrPasswordCommon` or `ErrPasswordPersonal`;
the `user` commands print them and the account page shows them. Length counts Unicode characters, not bytes, and
nothing is trimmed. The common list is embedded: the most common million passwords of SecLists (MIT licence), kept
only for entries of 15 to 256 characters, lowercased and deduplicated, since a shorter one is refused anyway; the file
says where it comes from. The comparison, with the list and with the username and display name, ignores case. The
account page's form says "At least 15 characters" and sets `minlength` and `maxlength`.

*Alternative considered*: checking Have I Been Pwned's range API. It sends part of a hash to a third party on every
password change and needs the network, against the product's privacy and offline nature.

### Security events with `slog`

Events are `slog` lines with a fixed message and `user`, `method` and `address` attributes, written where the outcome
is decided: `internal/user` for password sign-ins and locks, `Authenticate` getting the address for that; `internal/web`
for passkey and link sign-ins, refused requests, link pages that aren't valid, passkeys added or removed and password
changes. Messages: `signed in`, `sign-in failed` (with a `reason`: wrong password, locked, disabled, unknown user,
passkey refused), `sign-in lock started`, `request limit hit`, `password checks busy`, `sign-in link not valid`,
`passkey added`, `passkey removed`, `password changed`. A failed sign-in for an unknown username logs `user=unknown`,
since a username field often receives a password typed in the wrong place. Successes and the account's own changes are
`Info`, failures and refusals `Warn`. The handler and its destination stay as they are.

*Alternative considered*: a table of events in the database, queryable later. It would need a retention policy and a
way to read it, both out of scope; the container's log is already rotated and read with `docker logs`.

## Risks / Trade-offs

- [Many addresses guessing one password, 3 tries each per 15 minutes] → not stopped by counting; the 15-character
  minimum and the common list make guessing hopeless, and residents can move to passkeys.
- [A household behind one address hitting 20 requests a minute] → unlikely by hand; the message says to wait a minute.
- [`HAUSBUCH_TRUSTED_PROXIES` left unset behind a proxy] → every request has the proxy's address, so one person's
  mistakes lock the username for everyone through the proxy and the limit is shared: the README says to set it.
- [Residents with short passwords set before the rules] → they keep working; the rules only apply to new ones.
- [Addresses in the log are personal data] → the log stays on the installation and is rotated; the README says so.

## Migration Plan

The migration runs at start-up as usual. Its Down recreates the two `app_user` columns, empty, and drops the new table:
rolling back to the previous release loses only running locks.
