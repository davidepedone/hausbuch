# Spec Delta

## MODIFIED Requirements

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
