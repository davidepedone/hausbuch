# Spec Delta

## Purpose

Keeps copies of the database without anyone having to take them, so that an installation updated automatically can
always go back to its data as it was before an update, or a few days earlier.

## ADDED Requirements

### Requirement: Copy before a schema change
Before changing the schema of a database that already holds data, the system SHALL save a complete copy of it in the
backups directory. If the copy cannot be saved, the system SHALL leave the database unchanged and SHALL NOT start.

#### Scenario: Starting a newer release
- **WHEN** a release with schema changes starts on a database left by an older release
- **THEN** the backups directory holds a copy of the database as the older release left it, before the schema changes

#### Scenario: Copy impossible
- **WHEN** a release with schema changes starts and the backups directory can't be written
- **THEN** the database is left as it was and the system exits with an error saying why

#### Scenario: Start without schema changes
- **WHEN** the system starts on a database whose schema is already current
- **THEN** no copy is made for a schema change

#### Scenario: First start
- **WHEN** the system starts with no database yet
- **THEN** it creates the database without making a copy

#### Scenario: Repeated failing starts
- **WHEN** a schema change fails and the system is started again and again on the same database
- **THEN** the backups directory holds a single copy for that schema change, of the database as the older release left
  it, and the copies don't grow with each attempt

### Requirement: Daily copy
While running, the system SHALL save a complete copy of the database in the backups directory once a day, the first
when it starts unless that day's copy already exists. A failed daily copy SHALL be logged and SHALL NOT stop the system.

#### Scenario: Running for several days
- **WHEN** the system runs for three days
- **THEN** the backups directory holds one daily copy for each of those days

#### Scenario: Restart on the same day
- **WHEN** the system is restarted on a day for which a daily copy already exists
- **THEN** no second copy is made for that day

#### Scenario: Copy while residents are editing
- **WHEN** a page is saved while a daily copy is being made
- **THEN** both succeed, and the copy holds the page either as it was before the save or after it, never a mix

### Requirement: Old daily copies are removed
The system SHALL keep the configured number of most recent daily copies, 7 by default, and remove older ones. Copies
taken before a schema change SHALL be kept until removed by hand.

#### Scenario: More days than kept
- **WHEN** the system has run for 10 days with the default setting
- **THEN** the backups directory holds the daily copies of the last 7 days and no older ones

#### Scenario: Copies before schema changes are kept
- **WHEN** old daily copies are removed
- **THEN** every copy taken before a schema change is still there

### Requirement: Copies are usable as they are
Every copy SHALL be a complete database that the system can start on in place of the original. A copy interrupted
before it is complete SHALL NOT be left under a copy's name.

#### Scenario: Restoring a copy
- **WHEN** the system is stopped, a copy is put in place of the database, and the system is started
- **THEN** it shows the pages as they were when the copy was made, and residents can sign in with the passwords they
  had then

#### Scenario: Copy interrupted
- **WHEN** the system stops while a copy is being written
- **THEN** no incomplete file is left among the copies, and the next copy is made normally

### Requirement: Backups configuration
The backups directory and the number of daily copies to keep SHALL be configurable through environment variables. An
invalid value SHALL stop the system at start with an error naming the variable.

#### Scenario: Default location
- **WHEN** no backups directory is configured
- **THEN** copies are saved in a `backups` directory next to the database file

#### Scenario: Invalid number of copies
- **WHEN** the number of daily copies to keep is not a whole number of at least 1
- **THEN** the system does not start and reports the variable and its value

### Requirement: Running a previous release
A previous release SHALL start and work normally on a database whose schema was changed by a newer release, so that
an installation can be rolled back by running the previous image.

#### Scenario: Rolling back
- **WHEN** a newer release has changed the schema and the previous release is started on that database
- **THEN** residents can sign in, read, edit, create and search pages
