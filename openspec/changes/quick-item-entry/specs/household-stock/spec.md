# Spec Delta

## ADDED Requirements

### Requirement: Expiry as a month and a year
Wherever an item's expiry date is entered, the field SHALL ask for a month and a year, and the row SHALL expire on
that month's last day. An "Exact day" choice SHALL switch the field to a full date. A form showing an expiry date
that is not a month's last day SHALL show it as a full date.

#### Scenario: Month and year
- **WHEN** a user adds "Aspirin" to Medicines with the expiry month September 2027
- **THEN** the row expires on 30 September 2027

#### Scenario: February
- **WHEN** a user adds "Crackers" with the expiry month February 2028
- **THEN** the row expires on 29 February 2028

#### Scenario: Exact day
- **WHEN** a user adding "Yogurt" chooses "Exact day" and enters 18 November 2026
- **THEN** the row expires on 18 November 2026

#### Scenario: Editing a month's last day
- **WHEN** a user edits a row expiring on 30 September 2027
- **THEN** the form shows the month September 2027

#### Scenario: Editing another day
- **WHEN** a user edits a row expiring on 18 November 2026
- **THEN** the form shows the full date 18 November 2026

#### Scenario: Month without a year
- **WHEN** a user adds an item with an expiry month but no year, or a year but no month
- **THEN** the system refuses, says the expiry date isn't complete, and keeps what was entered

#### Scenario: No expiry
- **WHEN** a user adds "Salt" leaving the month and the year empty
- **THEN** the row is shown without an expiry date

### Requirement: Adding several items in a row
After an item is added to a section, the "Add item" form SHALL stay open and empty, ready for the next item's name,
and SHALL say which item was added. Its location SHALL hold the location of the item just added, which choosing a
known product name SHALL NOT replace.

#### Scenario: Next item
- **WHEN** a user adds "Passata" to Pantry
- **THEN** the Pantry section shows the "Add item" form open, with an empty name ready to type in, and says that
  "Passata" was added

#### Scenario: Location kept
- **WHEN** a user adds "Passata" kept in "Cellar"
- **THEN** the next item's location is "Cellar"

#### Scenario: Known product after a kept location
- **WHEN** the location "Cellar" was kept from the item just added, and the user chooses the name "Rice", last kept in
  "Pantry, top shelf" in the category "Long-life"
- **THEN** the location stays "Cellar" and the category becomes "Long-life"

#### Scenario: Known product with an empty location
- **WHEN** the item just added had no location, and the user chooses the name "Rice", last kept in "Pantry, top shelf"
- **THEN** the location becomes "Pantry, top shelf"

## MODIFIED Requirements

### Requirement: Suggestions from products already used
The system SHALL remember, per section, every product name used and the unit, category, location and shelf life after
opening it last had. While adding an item, the names and categories already used in the section and the locations used
in any section SHALL be suggested, and choosing a known name SHALL fill in its unit, category, location and shelf life
after opening, which stay editable.

#### Scenario: Buy something again
- **WHEN** a "Passata" row in "Long-life", measured in "bottles", kept in "Pantry, top shelf" and lasting 4 days once
  opened was removed, and a user starts adding an item and chooses the suggested name "Passata"
- **THEN** the unit "bottles", the category "Long-life", the location "Pantry, top shelf" and the shelf life of 4 days
  are filled in

#### Scenario: Category suggestions
- **WHEN** the pantry has items in "Fresh" and "Long-life" and a user starts typing a category
- **THEN** "Fresh" and "Long-life" are suggested

#### Scenario: Suggestions stay in their section
- **WHEN** "Paracetamol" was used in Medicines, in the category "Pain"
- **THEN** neither "Paracetamol" nor "Pain" is suggested while adding an item to Pantry

#### Scenario: Locations from the whole home
- **WHEN** "Detergent" was kept in "Cellar" in Supplies and a user starts adding an item to Pantry
- **THEN** "Cellar" is suggested as a location
