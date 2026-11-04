# Spec Delta

## Purpose

Keeps every version of every page with who wrote it, so any change can be understood and undone, and protects
residents from overwriting each other's edits.

## ADDED Requirements

### Requirement: Every save is a revision
Every save that creates a page or changes its title, type or body SHALL be kept as a revision recording the title,
type and body as saved, the user who saved it and when. A save that changes nothing SHALL NOT create a revision.

#### Scenario: Edit creates a revision
- **WHEN** a user saves a change to a page
- **THEN** a new revision exists with the saved content, their name and the time

#### Scenario: Save without changes
- **WHEN** a user saves a page without changing its title, type or body
- **THEN** no revision is added

### Requirement: View the history of a page
Any signed-in user SHALL be able to list a page's revisions, newest first, with author and time, and open any of them
read-only.

#### Scenario: History list
- **WHEN** a user opens the history of a page edited three times after its creation
- **THEN** they see four revisions, newest first, each with its author and time

#### Scenario: Open an old revision
- **WHEN** a user opens an old revision
- **THEN** they see that revision's title, type and formatted body, clearly marked as not the current version

### Requirement: Compare revisions
Any signed-in user SHALL be able to compare a revision with the current version, seeing which lines were added and
which removed.

#### Scenario: Compare
- **WHEN** a user compares an old revision with the current version
- **THEN** lines present only in the current version are shown as added and lines present only in the old one as
  removed

### Requirement: Restore a revision
Any signed-in user SHALL be able to restore an old revision. Restoring SHALL add a new revision, authored by them,
with the old revision's content; no revision SHALL ever be removed.

#### Scenario: Restore
- **WHEN** a user restores the second revision of a page that has five
- **THEN** the page shows the second revision's content and the history has six revisions, the newest by them

### Requirement: Concurrent edits do not overwrite each other
A save SHALL be accepted only if the page has not changed since the user began editing. Otherwise the system SHALL
keep the user's text, tell them who changed the page meanwhile, and let them review the other change before saving
again.

#### Scenario: Conflicting save
- **WHEN** two users open the same page for editing, the first saves, and then the second saves
- **THEN** the second save is not applied, the second user's text is still in their editor, and they are told the
  first user changed the page

#### Scenario: No conflict
- **WHEN** a user saves a page nobody else changed since they began editing
- **THEN** the save is applied
