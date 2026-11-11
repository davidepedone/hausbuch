# Spec Delta

## MODIFIED Requirements

### Requirement: Repeated failed sign-ins lock the username
After 3 failed sign-in attempts in a row for the same username from the same address, the system SHALL refuse
password sign-ins for that username from that address, even with the right password, for 15 minutes. Sign-ins for that
username from other addresses SHALL NOT be affected. The message SHALL be the same as for wrong credentials. The lock
SHALL survive a restart and SHALL be liftable from the command line, for every address at once. An address is an IPv4
address, or the first 64 bits of an IPv6 address.

#### Scenario: Lock after failures
- **WHEN** 3 sign-ins in a row fail for a username from one address and the right password is then submitted from
  that address within 15 minutes
- **THEN** the user is not signed in

#### Scenario: Other addresses unaffected
- **WHEN** a username is locked from one address and the right password is submitted from another address
- **THEN** the user is signed in

#### Scenario: Lock expires
- **WHEN** 15 minutes have passed since the last of those failures
- **THEN** the right password signs the user in from that address

#### Scenario: Lock lifted from the command line
- **WHEN** a username is locked from one or more addresses and the `user unlock` command is run for it
- **THEN** the right password signs that user in immediately from any of them

## ADDED Requirements

### Requirement: Request limit per address
The system SHALL refuse more than 20 requests a minute from one address to the login page, the login form, sign-in
links and passkey sign-in, saying to try again in a minute. Other requests SHALL NOT count towards the limit nor be
refused by it.

#### Scenario: Too many attempts
- **WHEN** an address has made 20 requests to the login form within a minute and makes another
- **THEN** it is refused with a message to try again in a minute, and no password is checked

#### Scenario: Limit over
- **WHEN** a minute has passed since an address was refused
- **THEN** its next sign-in is handled as usual

#### Scenario: Another address
- **WHEN** one address is over the limit and another address opens the login page
- **THEN** the login page is shown

#### Scenario: Same IPv6 block
- **WHEN** two IPv6 addresses sharing their first 64 bits make 20 requests to the login form within a minute between
  them, and one of them makes another
- **THEN** it is refused

### Requirement: Client address behind a proxy
The address of a request SHALL be the one its connection comes from. Only when the connection comes from a proxy listed
in the installation's configuration SHALL the address be the client's, as that proxy reports it in the
`X-Forwarded-For` header.

#### Scenario: Header from an unknown sender
- **WHEN** a request whose connection doesn't come from a listed proxy carries an `X-Forwarded-For` header
- **THEN** the header is ignored and the connection's address is used

#### Scenario: Behind a listed proxy
- **WHEN** a request comes through a listed proxy that reports the client's address in `X-Forwarded-For`
- **THEN** the client's address is used for the request limit, the sign-in lock and the log

### Requirement: Limited concurrent password checks
At most 2 password checks SHALL run at the same time. A sign-in or password change that has waited 5 seconds for its
turn SHALL be refused with a message to try again, and SHALL NOT count as a failed attempt.

#### Scenario: Too many at once
- **WHEN** 2 password checks are running and a sign-in has waited 5 seconds for one of them to end
- **THEN** that sign-in is refused with a message to try again, and the username's failed attempts are unchanged

### Requirement: Rules for new passwords
A new password, set by `user add`, `user set-password` or the account page, SHALL be 15 to 256 characters long and
SHALL NOT be a common password or the account's username or display name, ignoring case. No mix of letter cases, digits
or symbols SHALL be required, and passwords SHALL NOT expire. A refused password SHALL be told apart by the reason.
Existing passwords SHALL keep working.

#### Scenario: Too short
- **WHEN** a password of 14 characters is entered for `user add`
- **THEN** the command fails, says the password must be at least 15 characters, and creates nothing

#### Scenario: Too long
- **WHEN** a password of 257 characters is entered on the account page
- **THEN** the password is not changed and the page says it can be at most 256 characters

#### Scenario: Common password
- **WHEN** a password from the list of common passwords is entered for `user set-password`
- **THEN** the password is not changed and the command says it is too common

#### Scenario: Same as the username
- **WHEN** a user's username, in different letter case, is entered as their new password on the account page
- **THEN** the password is not changed and the page says it must differ from the username and display name

#### Scenario: Passphrase
- **WHEN** a password of 4 lowercase words separated by spaces, 28 characters in all, is entered on the account page
- **THEN** the password is changed

#### Scenario: Existing short password
- **WHEN** a user whose password was set before these rules, shorter than 15 characters, signs in with it
- **THEN** they are signed in

#### Scenario: Rules shown
- **WHEN** a user opens the account page
- **THEN** the password form says a new password needs at least 15 characters

### Requirement: Security events logged
The system SHALL write a line to its log for each sign-in and failed sign-in, with the method (password, passkey or
link) and the address; a lock starting; a request refused by the request limit or the password-check limit; a sign-in
link opened when not valid; a passkey added or removed; and a password changed. A line SHALL name the account when one
is concerned, and SHALL NOT contain passwords, link tokens, or usernames typed that match no account.

#### Scenario: Failed sign-in
- **WHEN** a sign-in with a wrong password fails for an existing username
- **THEN** the log has a line saying a password sign-in failed, with the username and the address, and without the
  password

#### Scenario: Unknown username
- **WHEN** a sign-in fails for a username that matches no account
- **THEN** the log has a line saying a password sign-in failed for an unknown username, with the address, and without
  what was typed

#### Scenario: Link used
- **WHEN** a user signs in with a sign-in link
- **THEN** the log has a line saying they signed in with a link, with the address, and without the link's token
