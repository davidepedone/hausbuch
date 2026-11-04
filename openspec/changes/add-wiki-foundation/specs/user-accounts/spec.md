# Spec Delta

## Purpose

Gives every resident their own account and keeps everything behind a login. Accounts are managed from the command
line of the running container, so only whoever runs the installation decides who has access.

## ADDED Requirements

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
After 3 failed sign-in attempts in a row for the same username, the system SHALL refuse sign-ins for that username,
even with the right password, for 15 minutes. The message SHALL be the same as for wrong credentials. The lock SHALL
survive a restart and SHALL be liftable from the command line.

#### Scenario: Lock after failures
- **WHEN** 3 sign-ins in a row fail for a username and the right password is then submitted within 15 minutes
- **THEN** the user is not signed in

#### Scenario: Lock expires
- **WHEN** 15 minutes have passed since the last of those failures
- **THEN** the right password signs the user in

#### Scenario: Lock lifted from the command line
- **WHEN** a username is locked and the `user unlock` command is run for it
- **THEN** the right password signs that user in immediately

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
with their state, set an account's password, disable and enable an account, and lift a lock. They SHALL work while
the server is running. There SHALL be no self-registration and no account management in the web interface.

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
