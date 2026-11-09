# Spec Delta

## Purpose

Lets the people using an installation see which release it runs, read what changed in it, and report a problem or
suggest an idea.

## ADDED Requirements

### Requirement: Running version
Every page of a signed-in user SHALL end with a footer showing the running version: the release's version for a
released build, "dev" for any other build.

#### Scenario: Released build
- **WHEN** the running build is release 2026.11.3 and a signed-in user opens any page
- **THEN** its footer shows "2026.11.3"

#### Scenario: Other build
- **WHEN** the running build wasn't made by the release script
- **THEN** the footer shows "dev"

#### Scenario: Not signed in
- **WHEN** someone opens the sign-in page
- **THEN** no version is shown

### Requirement: What's new
The footer SHALL have a "What's new" link to the running version's release notes on GitHub, or to the list of
releases for a "dev" build. It SHALL open in the same window.

#### Scenario: Release notes
- **WHEN** the running build is release 2026.11.3 and a user follows "What's new"
- **THEN** the GitHub release v2026.11.3 of the repository opens

#### Scenario: Dev build
- **WHEN** the running build is "dev" and a user follows "What's new"
- **THEN** the repository's list of releases opens

### Requirement: Report a problem
The footer SHALL have a "Report a problem" link to the repository's choice of issue forms. It SHALL open in the same
window.

#### Scenario: Report
- **WHEN** a user follows "Report a problem"
- **THEN** GitHub's page for choosing between the bug and the idea form opens
