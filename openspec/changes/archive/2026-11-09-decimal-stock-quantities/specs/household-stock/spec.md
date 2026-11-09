# Spec Delta

## MODIFIED Requirements

### Requirement: Add an item
Any signed-in user SHALL be able to add an item to a section with a name, a quantity above 0 with at most one decimal,
and optionally a unit, a location, an expiry date and a number of days it lasts once opened. The name SHALL NOT be
empty. The decimal separator SHALL be either a dot or a comma, and a whole quantity SHALL be shown without decimals.

#### Scenario: New item
- **WHEN** a user adds "Passata", quantity 3, unit "bottles", location "Pantry, top shelf", expiry date 30 September
  2027
- **THEN** the Pantry section shows the row with these values and records the user as the one who added it

#### Scenario: Item without expiry date
- **WHEN** a user adds "Salt" with no expiry date
- **THEN** the row is shown without an expiry date

#### Scenario: Quantity with a decimal
- **WHEN** a user adds "Flour" with quantity "1,5" and unit "kg"
- **THEN** the row shows quantity 1.5

#### Scenario: Whole quantity
- **WHEN** a user adds "Rice" with quantity "2.0"
- **THEN** the row shows quantity 2

#### Scenario: Missing name or quantity
- **WHEN** a user adds an item with an empty name, or a quantity that is 0 or less, has more than one decimal or is
  not a number
- **THEN** the system refuses, says which field is wrong, and keeps what was entered

### Requirement: Change the quantity
Each row SHALL offer a way to decrease and increase its quantity by one. Decreasing a row with a quantity of 1 or less
SHALL remove it.

#### Scenario: Use one
- **WHEN** a user decreases "Passata" from 3
- **THEN** the row shows quantity 2

#### Scenario: Use one of a decimal quantity
- **WHEN** a user decreases "Flour" from 2.5
- **THEN** the row shows quantity 1.5

#### Scenario: Use the last one
- **WHEN** a user decreases a row with quantity 1
- **THEN** the row is removed

#### Scenario: Use the last part
- **WHEN** a user decreases a row with quantity 0.5
- **THEN** the row is removed

#### Scenario: Two users at once
- **WHEN** two users each decrease "Passata" from 3 at the same moment
- **THEN** the row shows quantity 1

### Requirement: Shelf life after opening
A row MAY say how many days the item lasts once opened. A row SHALL be markable as opened today, and an opened row
SHALL show its opening date. The opening date SHALL be editable like the other fields. Marking a row with a quantity
above 1 SHALL split 1 off into a row of its own, opened; a row with a quantity of 1 or less SHALL be opened as a whole.

#### Scenario: Open a single item
- **WHEN** a user marks "Eye drops", quantity 1, lasting 28 days once opened, as opened today
- **THEN** the row shows it was opened today

#### Scenario: Open one of several
- **WHEN** a user marks "Passata", quantity 3, lasting 4 days once opened, as opened today
- **THEN** the section shows "Passata" with quantity 2, not opened, and a new "Passata" row with quantity 1, opened
  today, both with the same printed expiry date

#### Scenario: Open one of a decimal quantity
- **WHEN** a user marks "Milk", quantity 2.5, as opened today
- **THEN** the section shows "Milk" with quantity 1.5, not opened, and a new "Milk" row with quantity 1, opened today

#### Scenario: Open a part
- **WHEN** a user marks "Flour", quantity 0.5, as opened today
- **THEN** the row shows quantity 0.5 and that it was opened today
