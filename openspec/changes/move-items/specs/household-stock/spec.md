# Spec Delta

## ADDED Requirements

### Requirement: Move an item to another section
While editing a row, any signed-in user SHALL be able to choose another section for it. Saving SHALL move the row
there with its values, as edited, and show it in its new section. When the new section has an unopened row with the
same name, ignoring case, and expiry date, and the moved row isn't opened, the quantity SHALL be added to that row
instead.

#### Scenario: Move to the right section
- **WHEN** a user edits "Dishwasher tablets" in Pantry, chooses Supplies and saves
- **THEN** the row is in Supplies with its quantity, unit, category, location and dates, and no longer in Pantry

#### Scenario: Move onto the same product
- **WHEN** Supplies holds 1 "Descaler" expiring 14 November 2026 and a user moves 2 "descaler" with the same expiry
  date from Pantry to Supplies
- **THEN** Supplies holds one "Descaler" row with quantity 3, and Pantry has no "Descaler" row

#### Scenario: Opened row moves apart
- **WHEN** Supplies holds an unopened "Descaler" and a user moves an opened "Descaler" with the same expiry date
- **THEN** Supplies holds both rows, apart

#### Scenario: Edit and move together
- **WHEN** a user changes the quantity of "Detergent" to 2 and moves it to Supplies in the same save
- **THEN** the row is in Supplies with quantity 2

### Requirement: Suggestions after a move
After a row is moved, the new section SHALL remember its product as after adding it there, and the section it left
SHALL forget the product, unless rows with the same name, ignoring case, remain there.

#### Scenario: Suggested in the new section
- **WHEN** a user moves "Dishwasher tablets" in the category "Cleaning" from Pantry to Supplies
- **THEN** choosing "Dishwasher tablets" while adding to Supplies fills in the category "Cleaning"

#### Scenario: Forgotten where it was
- **WHEN** a user moves the only "Dish soap" row from Pantry to Supplies
- **THEN** "Dish soap" is no longer suggested while adding an item to Pantry

#### Scenario: Still remembered where rows remain
- **WHEN** Pantry holds two "Passata" rows with different expiry dates and a user moves one of them to Supplies
- **THEN** "Passata" is still suggested while adding an item to Pantry
