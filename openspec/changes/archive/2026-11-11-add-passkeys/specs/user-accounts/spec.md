# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

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
