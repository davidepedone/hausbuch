# Spec Delta

## Purpose

Gives every resident their own account, keeps everything behind a login, and lets an administrator manage who has
access.

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

### Requirement: Repeated failed sign-ins lock the username
After 5 failed sign-in attempts in a row for the same username, the system SHALL refuse sign-ins for that username,
even with the right password, for 5 minutes. The message SHALL be the same as for wrong credentials.

#### Scenario: Lock after failures
- **WHEN** 5 sign-ins in a row fail for a username and the right password is then submitted within 5 minutes
- **THEN** the user is not signed in

#### Scenario: Lock expires
- **WHEN** 5 minutes have passed since the last of those failures
- **THEN** the right password signs the user in

### Requirement: Long-lived sessions
A session SHALL stay valid for a configurable duration, 30 days by default, so residents do not have to sign in again
every time they open the wiki on their own devices.

#### Scenario: Returning within the session duration
- **WHEN** a signed-in user comes back to the wiki before the session duration has elapsed
- **THEN** they are still signed in

#### Scenario: Session expired
- **WHEN** a user comes back after the session duration has elapsed
- **THEN** they are asked to sign in again

### Requirement: Sign out
The system SHALL let a signed-in user sign out, ending their session on that device.

#### Scenario: User signs out
- **WHEN** a signed-in user signs out
- **THEN** their session ends and the next request asks them to sign in

### Requirement: First administrator from configuration
When the system starts with no accounts at all, it SHALL create an administrator account from the username and
password given in its configuration. When accounts already exist, it SHALL ignore those settings.

#### Scenario: First start
- **WHEN** the system starts with no accounts and the initial administrator username and password are configured
- **THEN** an administrator account with those credentials exists and can sign in

#### Scenario: Later starts
- **WHEN** the system starts and at least one account exists
- **THEN** no account is created or changed from the configuration

#### Scenario: First start without the settings
- **WHEN** the system starts with no accounts and the initial administrator settings are missing
- **THEN** the system refuses to start and reports which settings are missing

### Requirement: Administrator manages accounts
An administrator SHALL be able to create an account (username, display name, initial password, administrator or
not), reset its password, change whether it is an administrator, and disable or re-enable it. Usernames SHALL be
unique, ignoring case. There SHALL be no self-registration.

#### Scenario: Create an account
- **WHEN** an administrator creates an account with an unused username
- **THEN** the new user can sign in with the initial password

#### Scenario: Duplicate username
- **WHEN** an administrator creates an account whose username matches an existing one ignoring case
- **THEN** the system refuses and says the username is taken

#### Scenario: Non-administrator tries to manage accounts
- **WHEN** a user who is not an administrator opens account management
- **THEN** the system refuses access

#### Scenario: Disable an account
- **WHEN** an administrator disables an account
- **THEN** that user's existing sessions end and they can no longer sign in, and their past edits still show their
  name

### Requirement: At least one administrator remains
The system SHALL refuse any change that would leave no enabled administrator.

#### Scenario: Last administrator demotes or disables themselves
- **WHEN** the only enabled administrator tries to remove their own administrator role or disable their account
- **THEN** the system refuses and explains that at least one administrator is required

### Requirement: Users change their own password
A signed-in user SHALL be able to change their own password by giving the current one.

#### Scenario: Password change
- **WHEN** a user gives their correct current password and a new one
- **THEN** the new password works for the next sign-in and the old one does not

#### Scenario: Wrong current password
- **WHEN** a user gives a wrong current password
- **THEN** the password is not changed
