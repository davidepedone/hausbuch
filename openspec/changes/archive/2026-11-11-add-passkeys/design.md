# Design

## Context

See `proposal.md` for the motivation and the delta spec for the required behaviour. Today a session is a random token
in an `HttpOnly`, `SameSite=Lax` cookie, `Secure` when the request came over HTTPS directly or through a proxy
(`X-Forwarded-Proto`); the database keeps only its SHA-256. Passwords are argon2id hashes in `app_user`, with a lock
after 3 failures. The `user` commands run in the container against the same database. Configuration comes from
`HAUSBUCH_*` environment variables.

WebAuthn binds a passkey to a relying party ID, the site's host name, and the browser accepts a ceremony only in a
secure context: HTTPS, or `localhost`.

## Goals / Non-Goals

**Goals:**

- Passkeys as the everyday sign-in where the installation allows them; nothing changes where it doesn't.
- No secret in the database that signs anyone in: passkeys store public keys, links store hashes.

**Non-Goals:**

- Attestation (proving the authenticator's make): a household has no reason to restrict it.

## Decisions

### `HAUSBUCH_URL` gives the origin and the relying party

`HAUSBUCH_URL` is the address users open, such as `https://home.example.org`. Its host is the relying party ID and the
URL, without a path, the only accepted origin. Passkeys are enabled when it is `https://…`, or `http://localhost…`
(for development and tests). The setting can't be guessed from requests: behind a proxy the host header and scheme are
the proxy's word, and a relying party ID must not change once passkeys exist.

*Alternative considered*: deriving the origin from each request. A misconfigured proxy would then make passkeys created
under one name useless under another, silently.

### `go-webauthn/webauthn` for the ceremonies

The library verifies registration and assertion responses, including the counter and the origin. Registration asks for
a discoverable credential (resident key required) with user verification preferred, so that "Sign in with a passkey"
needs no username; login uses its discoverable flow, mapping the returned user handle to the account.

*Alternative considered*: implementing the verification by hand. CBOR, COSE keys and the signature formats are exactly
where a small mistake is a security hole.

### Storage

A migration adds:

- `app_user.webauthn_id`: 32 random bytes, the user handle, created with the first passkey. It is not the account's id,
  which would leak the order of accounts to every authenticator.
- `passkey`: credential id (primary key), user id (cascading delete), the library's credential record as JSON,
  `created_at`, `last_used_at`. The record holds the public key, flags and sign counter; the counter and
  `last_used_at` are updated on every sign-in.
- `sign_in_link`: SHA-256 of the token (primary key), user id, `expires_at`. Printing a link deletes the account's
  earlier links; using one deletes it.

### Ceremony state in memory

Between the "begin" and "finish" request of a ceremony the server keeps the challenge. It is held in memory, keyed by a
random id in a short-lived cookie, and expires after 5 minutes. A restart in the middle only makes the user try again;
there is one process.

### Endpoints and pages

- `POST /login/passkey/begin` and `/finish`: open, like the login page; the finish answers with the URL to go to, and
  checks the account is enabled but not the password lock.
- `GET /account/passkeys/offer`: the offer page, shown by redirect after a password or link sign-in for a user without
  passkeys, carrying the original `next`.
- `POST /account/passkeys/begin` and `/finish`, `POST /account/passkeys/{id}/remove`: signed in.
- `GET /link/{token}` shows the account's display name and a form; `POST /link/{token}` signs in. The GET doesn't
  touch the link, so previews don't use it up.

The account page (`/account/password` today) gains a "Passkeys" section. All passkey markup is left out when passkeys
are disabled.

### Browser side in `passkeys.js`

A small script converts the options' base64url fields to `ArrayBuffer`s and back, calls `navigator.credentials.create`
and `.get`, and posts the result. On the login page it also starts a conditional request (`mediation: "conditional"`)
when the browser supports it, so passkeys appear among the username field's suggestions; the field gets
`autocomplete="username webauthn"`. Without JavaScript the password form works as before.

### Tests

The `user` package and the link and offer handlers are tested as usual. The ceremonies are tested in Chrome with the
DevTools virtual authenticator (`WebAuthn.addVirtualAuthenticator`), which creates and uses real passkeys: the test
server is reached as `localhost`, a secure context.

## Risks / Trade-offs

- [`HAUSBUCH_URL` set wrong, or changed later] → passkeys stop working while passwords and links keep working; the
  README says to set it once, to the address users open.
- [A new dependency in the sign-in path] → it is the most used Go WebAuthn library, kept up to date by Dependabot like
  the others.
- [A lost phone whose passkey wasn't synced] → the password, or a sign-in link from the command line, gets the user
  back in; they remove the old passkey from the account page.
