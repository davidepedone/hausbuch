# page-history Specification

## Purpose

Keeps every version of every page with who wrote it, so any change can be understood and undone, and protects
residents from overwriting each other's edits.

## Requirements

### Requirement: Every save is a revision
Every save that creates a page or changes its title, type or body SHALL be kept as a revision recording them as saved,
the user who saved it and when. A save that changes nothing SHALL NOT create a revision.

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
with the old revision's content, and SHALL NOT remove any revision.

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

### Requirement: Warning when someone else is editing
Opening a page's editor SHALL warn the user when another user has the same page open in an editor, naming them and
since when, and SHALL let them go back or edit anyway. An editor SHALL count as open until it is saved or cancelled,
or until it has given no sign of activity for 2 minutes (a closed tab, a sleeping phone).

#### Scenario: Someone else is editing
- **WHEN** Anna has had a page's editor open for 5 minutes and Marco opens the editor of the same page
- **THEN** Marco sees that Anna has been editing it for 5 minutes, and can go back or edit anyway

#### Scenario: Editing anyway
- **WHEN** Marco chooses to edit anyway
- **THEN** the editor opens and saving is still subject to the concurrent edit check

#### Scenario: Editor saved or cancelled
- **WHEN** Anna saves or cancels, and Marco then opens the editor
- **THEN** Marco sees no warning

#### Scenario: Editor abandoned
- **WHEN** Anna closes her tab without saving, and Marco opens the editor 2 minutes later
- **THEN** Marco sees no warning

#### Scenario: Own editor in another tab
- **WHEN** a user has a page's editor open in one tab and opens it again in another
- **THEN** they see no warning about themselves
