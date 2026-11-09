# Spec Delta

## MODIFIED Requirements

### Requirement: Lead time per section
Each section SHALL have a lead time in days: 30 for Medicines, 7 for Pantry and 7 for Supplies, unless the
installation configures others. A configured lead time SHALL be a whole number of at least 0; otherwise the system
SHALL NOT start.

#### Scenario: Defaults
- **WHEN** no lead times are configured and a medicine and a pantry item both expire in 20 days
- **THEN** the medicine is expiring and the pantry item is not

#### Scenario: Supplies default
- **WHEN** no lead times are configured and a supplies item expires in 7 days and another in 8 days
- **THEN** the first is expiring and the second is not

#### Scenario: Configured lead time
- **WHEN** the installation sets the pantry's lead time to 3 days and a pantry item expires in 5 days
- **THEN** it is not expiring

#### Scenario: Configured supplies lead time
- **WHEN** the installation sets the supplies' lead time to 0 days and a supplies item expires tomorrow
- **THEN** it is not expiring

#### Scenario: Invalid lead time
- **WHEN** a configured lead time is negative or not a whole number
- **THEN** the system does not start and says which setting is wrong

### Requirement: Expiring box on the home page
The home page SHALL start with a box listing every expired and expiring row of every section, expired first, then by
effective expiry date, each with its section and leading to it. With no such rows, the box SHALL NOT be shown.

#### Scenario: Something to check
- **WHEN** "Yogurt" in Pantry expired yesterday and "Eye drops" in Medicines expire in 10 days
- **THEN** the home page's box lists "Yogurt" first, marked expired, then "Eye drops", each with its section

#### Scenario: Supplies in the box
- **WHEN** "Descaler" in Supplies expires in 3 days
- **THEN** the home page's box lists "Descaler" with its section, Supplies

#### Scenario: Nothing to check
- **WHEN** no row is expired or expiring
- **THEN** the home page shows no box
