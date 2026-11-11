# Tasks

## 1. Accounts

- [x] 1.1 Add the migration (`sign_in_failure`, dropping `app_user.failed_sign_ins` and `locked_until`) and verify its
  Down with a throwaway test
- [x] 1.2 Move the lock to username and address in `Authenticate`, `Unlock`, `List` and the daily clean-up; verify
  tests for "Lock after failures", "Other addresses unaffected", "Lock expires", "Lock lifted from the command line",
  unknown usernames leaving no row, and `user list` showing a lock from any address
- [x] 1.3 Guard argon2id with the limit of 2 and `ErrBusy`; verify a test for "Too many at once", holding both slots
- [x] 1.4 Add the password rules with the embedded common list to `Add`, `SetPassword` and `ChangePassword`, and their
  messages to `user add` and `user set-password`; verify tests for "Too short", "Too long", "Common password", "Same as
  the username", "Passphrase" and "Existing short password", and the README's Accounts section on the rules

## 2. Web interface

- [x] 2.1 Add the request address with `HAUSBUCH_TRUSTED_PROXIES`; verify tests for "Header from an unknown sender",
  "Behind a listed proxy", an invalid setting stopping the server, and IPv6 addresses keyed by `/64`, and the setting
  in the README's configuration table
- [x] 2.2 Add the request limit on the open routes, with the login page's and the passkey endpoints' answers; verify
  web tests for "Too many attempts", "Limit over", "Another address" and "Same IPv6 block", and that a signed-in page
  is never refused
- [x] 2.3 Show the rules and the refusals on the account page, and "busy" on the login and account pages; verify web
  tests for "Rules shown", the rules' messages and a busy sign-in
- [x] 2.4 Write the security events; verify tests capturing the log for "Failed sign-in", "Unknown username" and "Link
  used", and that no line holds a password or a token; describe the log in the README

## 3. Integration

- [x] 3.1 With `make check` green, in Chrome at 390px and at desktop width, go through a refused password on the
  account page, the login page over the limit and the passkey sign-in over the limit; verify the messages show and
  nothing overflows

## Workflow follow-up

- Archive the change once merged.
