# user-accounts Specification

## Purpose

Gives every resident their own account and keeps everything behind a login. Accounts are managed from the command
line of the running container, so only whoever runs the installation decides who has access.

## Requirements

### Requirement: Login required everywhere
The system SHALL require a signed-in user for every page and endpoint except the login page, the static assets it
needs, and the health check. Being able to reach the server SHALL grant no access to content.

#### Scenario: Anonymous visitor opens a page
- **WHEN** a visitor who is not signed in requests any wiki page
- **THEN** the system redirects them to the login page and shows no page content

#### Scenario: Anonymous request to a content endpoint
- **WHEN** a request without a valid session reaches an endpoint that returns or changes content
- **THEN** the system refuses it without returning or changing any content

### Requirement: Sign in with username and password
The system SHALL let a user sign in with their username and password. A failed attempt SHALL NOT reveal whether the
username exists.

#### Scenario: Correct credentials
- **WHEN** a user submits a valid username and password
- **THEN** the system signs them in and shows the page they originally asked for, or the home page

#### Scenario: Wrong credentials
- **WHEN** a user submits a username that does not exist, or an existing username with the wrong password
- **THEN** the system shows the same "invalid username or password" message in both cases and does not sign them in

#### Scenario: Disabled account
- **WHEN** the user of a disabled account submits its correct credentials
- **THEN** the system does not sign them in

### Requirement: No accounts yet
While no account exists, the login page SHALL say so and show the command that creates the first one.

#### Scenario: Fresh installation
- **WHEN** someone opens the login page of an installation with no accounts
- **THEN** the page explains that accounts are created from the command line and shows the `user add` command

### Requirement: Repeated failed sign-ins lock the username
After 3 failed sign-ins in a row for a username from one address, an IPv4 address or the first 64 bits of an IPv6
address, the system SHALL refuse password sign-ins for it from that address for 15 minutes, even with the right
password, with the same message as for wrong credentials. Other addresses SHALL NOT be affected. The lock SHALL
survive a restart and SHALL be liftable from the command line, for every address at once.

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

### Requirement: Long-lived sessions
A session SHALL stay valid until a configurable duration, 30 days by default, passes without any request from it, so
residents who use the wiki on their own devices do not have to sign in again.

#### Scenario: Returning within the session duration
- **WHEN** a signed-in user comes back to the wiki before the session duration has elapsed since their last request
- **THEN** they are still signed in

#### Scenario: Regular use keeps the session
- **WHEN** a user makes a request every 20 days with a 30-day session duration
- **THEN** they stay signed in beyond 30 days after signing in

#### Scenario: Session expired
- **WHEN** a user comes back after the session duration has elapsed since their last request
- **THEN** they are asked to sign in again

### Requirement: Sign out
The system SHALL let a signed-in user sign out, ending their session on that device.

#### Scenario: User signs out
- **WHEN** a signed-in user signs out
- **THEN** their session ends and the next request asks them to sign in

### Requirement: Accounts managed from the command line
The application binary SHALL provide `user` commands to add an account (username and display name), list accounts
with their state, set an account's password, disable and enable an account, lift a lock, and print a one-time sign-in
link for an account. They SHALL work while the server is running. There SHALL be no self-registration and no account
management in the web interface beyond a user's own password and passkeys.

#### Scenario: Add an account
- **WHEN** `user add` is run with an unused username and a display name, and a password is entered
- **THEN** the new user can sign in with that password

#### Scenario: Duplicate username
- **WHEN** `user add` is run with a username that matches an existing one ignoring case
- **THEN** the command fails, says the username is taken, and creates nothing

#### Scenario: List accounts
- **WHEN** `user list` is run
- **THEN** it shows every account's username, display name, whether it is disabled and whether it is locked

#### Scenario: Set a password
- **WHEN** `user set-password` is run for an account and a new password is entered
- **THEN** the new password works for the next sign-in and the old one does not

#### Scenario: Disable an account
- **WHEN** `user disable` is run for an account
- **THEN** that user's existing sessions end and they can no longer sign in, and their past edits still show their
  name

#### Scenario: Enable an account
- **WHEN** `user enable` is run for a disabled account
- **THEN** that user can sign in again

#### Scenario: Print a sign-in link
- **WHEN** `user link` is run for an enabled account
- **THEN** it prints a one-time sign-in link for that account

### Requirement: Passwords never on the command line
The `user` commands SHALL read passwords from the terminal without echoing them, or from standard input when it is
not a terminal, and SHALL NOT accept them as arguments, so they don't end up in shell history or process lists.

#### Scenario: Interactive password
- **WHEN** `user add` is run in a terminal
- **THEN** it asks for the password twice without showing it, and refuses if the two don't match

#### Scenario: Password from standard input
- **WHEN** `user add` is run with the password piped to its standard input
- **THEN** it uses that password without asking

### Requirement: Users change their own password
A signed-in user SHALL be able to change their own password by giving the current one.

#### Scenario: Password change
- **WHEN** a user gives their correct current password and a new one
- **THEN** the new password works for the next sign-in and the old one does not

#### Scenario: Wrong current password
- **WHEN** a user gives a wrong current password
- **THEN** the password is not changed

### Requirement: Passkeys need a secure address
Passkeys SHALL be offered only when the installation is configured with the address its users open and that address
is HTTPS, or HTTP on `localhost`. Otherwise no page SHALL mention passkeys, and signing in SHALL work with passwords
and sign-in links as before.

#### Scenario: Plain HTTP installation
- **WHEN** the installation has no address configured, or an HTTP address other than `localhost`
- **THEN** the login page and the account page show nothing about passkeys

### Requirement: Sign in with a passkey
Where passkeys are offered, the login page SHALL let a user sign in with a passkey, with a button and through the
browser's passkey suggestions in the username field, without typing a username or a password, and SHALL then show
the page originally asked for, or the home page. A passkey sign-in SHALL work while the username is locked by failed
password attempts and SHALL NOT work for a disabled account. A cancelled one SHALL leave the password form in place.

#### Scenario: Passkey sign-in
- **WHEN** a user with a passkey chooses "Sign in with a passkey" and confirms on their device
- **THEN** they are signed in and shown the home page

#### Scenario: Locked username
- **WHEN** 3 password attempts for a username have failed and its user signs in with a passkey
- **THEN** they are signed in

#### Scenario: Disabled account
- **WHEN** the user of a disabled account signs in with a passkey
- **THEN** they are not signed in

#### Scenario: Passkey cancelled
- **WHEN** a user starts a passkey sign-in and cancels it on their device
- **THEN** the login page says the passkey sign-in didn't complete and the password form is still there

### Requirement: Passkey offered after a password sign-in
Where passkeys are offered, a user without a passkey who signs in with their password or a sign-in link SHALL be shown
a page offering to create one, before the page they were headed to. Creating it, or choosing "Not now", SHALL go on to
that page. A user with a passkey SHALL NOT be shown the offer.

#### Scenario: Offer after the password
- **WHEN** a user without a passkey signs in with their password
- **THEN** the passkey offer is shown, and creating a passkey or choosing "Not now" opens the page they asked for

#### Scenario: No offer with a passkey
- **WHEN** a user who has a passkey signs in with their password
- **THEN** the page they asked for opens directly

### Requirement: Own passkeys on the account page
Where passkeys are offered, the account page SHALL list the signed-in user's passkeys, each with the day it was created
and the day it was last used, and SHALL let the user add a passkey and remove any of theirs. Removing every passkey
SHALL leave the password working. A user SHALL see and remove only their own passkeys.

#### Scenario: Add a passkey
- **WHEN** a user adds a passkey on the account page and confirms on their device
- **THEN** the passkey is listed and signs them in from then on

#### Scenario: Remove a passkey
- **WHEN** a user removes one of their passkeys
- **THEN** it is no longer listed and no longer signs them in, and their password still works

### Requirement: One-time sign-in link
A sign-in link SHALL be valid for 15 minutes and for one sign-in, and printing a new link for an account SHALL make its
earlier links invalid. Opening a link SHALL show whose account it signs into and a button to sign in, so that a link
preview fetching the page doesn't use it up. Signing in through a link SHALL work over plain HTTP too, and SHALL NOT
work for a disabled account. The link SHALL be printed with the configured address, or as a path when none is
configured.

#### Scenario: Sign in with a link
- **WHEN** a user opens a link printed 5 minutes earlier and chooses to sign in
- **THEN** they are signed in

#### Scenario: Link used twice
- **WHEN** a user opens a link that has already signed someone in
- **THEN** the page says the link is no longer valid and nobody is signed in

#### Scenario: Link expired
- **WHEN** a user opens a link printed 16 minutes earlier
- **THEN** the page says the link is no longer valid and nobody is signed in

#### Scenario: Newer link printed
- **WHEN** a link is printed for an account and then another one for the same account
- **THEN** the first link no longer signs in

#### Scenario: Opening is not signing in
- **WHEN** a link is opened but its button is not chosen
- **THEN** nobody is signed in and the link still works

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
