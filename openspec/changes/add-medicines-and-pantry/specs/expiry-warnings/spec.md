# Spec Delta

## Purpose

Points out the medicines and food that have expired or are about to, so that the household uses or replaces them in
time instead of finding out too late.

## ADDED Requirements

### Requirement: Expired and expiring items
A row SHALL be expired once the day after its effective expiry date has begun, and expiring when it is not expired and
its effective expiry date is within its section's lead time from today. Days SHALL follow the installation's time
zone. A row without an effective expiry date SHALL never be expired or expiring.

#### Scenario: Last good day
- **WHEN** a row's effective expiry date is today
- **THEN** it is expiring, not expired

#### Scenario: Expired
- **WHEN** a row's effective expiry date was yesterday
- **THEN** it is expired

#### Scenario: Within the lead time
- **WHEN** the pantry's lead time is 7 days and a pantry row expires in 7 days
- **THEN** it is expiring

#### Scenario: Beyond the lead time
- **WHEN** the pantry's lead time is 7 days and a pantry row expires in 8 days
- **THEN** it is neither expired nor expiring

#### Scenario: Nothing to expire
- **WHEN** a row has no expiry date and is not opened with a shelf life after opening
- **THEN** it is neither expired nor expiring

### Requirement: Lead time per section
Each section SHALL have a lead time in days: 30 for Medicines and 7 for Pantry, unless the installation configures
others. A configured lead time SHALL be a whole number of at least 0; otherwise the system SHALL NOT start.

#### Scenario: Defaults
- **WHEN** no lead times are configured and a medicine and a pantry item both expire in 20 days
- **THEN** the medicine is expiring and the pantry item is not

#### Scenario: Configured lead time
- **WHEN** the installation sets the pantry's lead time to 3 days and a pantry item expires in 5 days
- **THEN** it is not expiring

#### Scenario: Invalid lead time
- **WHEN** a configured lead time is negative or not a whole number
- **THEN** the system does not start and says which setting is wrong

### Requirement: Warnings in the sections
In each section, expired and expiring rows SHALL be marked as such, distinguishably from each other, with their
effective expiry date.

#### Scenario: Marked rows
- **WHEN** the Pantry section holds an expired row, an expiring row and a row expiring next year
- **THEN** the first is marked expired, the second expiring, and the third is not marked

### Requirement: Expiring box on the home page
The home page SHALL start with a box listing every expired and expiring row of both sections, expired first, then by
effective expiry date, each with its section and leading to it. With no such rows, the box SHALL NOT be shown.

#### Scenario: Something to check
- **WHEN** "Yogurt" in Pantry expired yesterday and "Eye drops" in Medicines expire in 10 days
- **THEN** the home page's box lists "Yogurt" first, marked expired, then "Eye drops", each with its section

#### Scenario: Nothing to check
- **WHEN** no row is expired or expiring
- **THEN** the home page shows no box
