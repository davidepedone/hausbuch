# Spec Delta

## MODIFIED Requirements

### Requirement: Suggestions from products already used
The system SHALL remember, per section, every product name used and the category, location and shelf life after
opening it last had. While adding an item, the names and categories already used in the section and the locations used
in any section SHALL be suggested, and choosing a known name SHALL fill in its category, location and shelf life after
opening, which stay editable.

#### Scenario: Buy something again
- **WHEN** a "Passata" row in "Long-life", kept in "Pantry, top shelf" and lasting 4 days once opened was removed, and
  a user starts adding an item and chooses the suggested name "Passata"
- **THEN** the category "Long-life", the location "Pantry, top shelf" and the shelf life of 4 days are filled in

#### Scenario: Category suggestions
- **WHEN** the pantry has items in "Fresh" and "Long-life" and a user starts typing a category
- **THEN** "Fresh" and "Long-life" are suggested

#### Scenario: Suggestions stay in their section
- **WHEN** "Paracetamol" was used in Medicines, in the category "Pain"
- **THEN** neither "Paracetamol" nor "Pain" is suggested while adding an item to Pantry

#### Scenario: Locations from the whole home
- **WHEN** "Detergent" was kept in "Cellar" in Supplies and a user starts adding an item to Pantry
- **THEN** "Cellar" is suggested as a location

## ADDED Requirements

### Requirement: Location choices under the field
Wherever a location is entered, the locations used in any section SHALL be shown as choices under the field, which fill
it in when chosen. Locations differing only in case SHALL be one choice, as first spelt in alphabetical order. No
location SHALL be suggested before the household has used one.

#### Scenario: Choose a location under the field
- **WHEN** "Cellar" and "Fridge" have been used and a user adding an item to Medicines chooses "Cellar" under the
  location field
- **THEN** the location field holds "Cellar", and can still be changed by typing

#### Scenario: One choice per location
- **WHEN** "Cellar" was used in Supplies and "cellar" in Pantry
- **THEN** the location field offers one "Cellar" choice

#### Scenario: Nothing used yet
- **WHEN** no location has been used
- **THEN** the location field offers no choices

### Requirement: Stock by location
A "Locations" page, reached from the home page, SHALL list every location that some row of any section is kept in, in
alphabetical order ignoring case, each with the number of rows kept there. Locations differing only in case SHALL be
listed once, as first spelt in alphabetical order. Rows without a location SHALL not be counted.

#### Scenario: Locations across sections
- **WHEN** Pantry has two rows in "Cellar" and one in "Fridge", and Supplies one row in "cellar"
- **THEN** the Locations page lists "Cellar" with 3 rows and "Fridge" with 1

#### Scenario: No locations
- **WHEN** no row has a location
- **THEN** the Locations page says that no row has a location yet

### Requirement: A location's rows
Each location SHALL have a page listing the rows kept there, ignoring case, grouped by section in the sections' order,
each group in the order of its section, with each row's name, quantity, unit and expiry, linking to the row in its
section. Wherever a row shows its location, the location SHALL link to its page.

#### Scenario: What is in the cellar
- **WHEN** Pantry has "Passata" in "Cellar" and Supplies "Detergent" in "cellar", and a user opens the page of
  "Cellar"
- **THEN** it lists "Passata" under Pantry and "Detergent" under Supplies, each linking to its row

#### Scenario: From a row to its location
- **WHEN** a pantry row is kept in "Cellar" and a user chooses "Cellar" on that row
- **THEN** the page of "Cellar" opens

#### Scenario: A location left empty
- **WHEN** the last row kept in "Garage" is removed and a user opens the page of "Garage"
- **THEN** the page says that nothing is kept there
