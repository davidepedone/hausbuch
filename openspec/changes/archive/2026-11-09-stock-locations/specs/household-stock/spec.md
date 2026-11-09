# Spec Delta

## MODIFIED Requirements

### Requirement: One row per expiry date
The same product with different expiry dates, or kept in different locations, SHALL be kept as separate rows. Adding
a product whose name and location, ignoring case, and expiry date match an unopened row of the same section SHALL add
to that row's quantity instead.

#### Scenario: Same product, different expiry
- **WHEN** "Paracetamol" expiring in March 2027 exists and a user adds "Paracetamol" expiring in November 2026
- **THEN** the Medicines section shows two "Paracetamol" rows, one per expiry date

#### Scenario: Same product, same expiry
- **WHEN** "Passata", 2, expiring 30 September 2027 exists and a user adds "passata", 1, expiring the same day
- **THEN** there is one "Passata" row with quantity 3

#### Scenario: Same product, different location
- **WHEN** "Pasta", 2, kept in "Cellar" and expiring 31 May 2027 exists and a user adds "Pasta", 1, kept in "Kitchen"
  and expiring the same day
- **THEN** the Pantry section shows two "Pasta" rows, next to each other: 2 in "Cellar" and 1 in "Kitchen"

#### Scenario: Same location, different case
- **WHEN** "Pasta", 2, kept in "Cellar" exists and a user adds "Pasta", 1, kept in "cellar", with the same expiry date
- **THEN** there is one "Pasta" row with quantity 3, kept in "Cellar"

#### Scenario: Opened row stays apart
- **WHEN** an opened "Passata" expiring 30 September 2027 exists and a user adds "Passata" expiring the same day
- **THEN** the new bottle is a row of its own, not opened

### Requirement: Move an item to another section
While editing a row, any signed-in user SHALL be able to choose another section for it. Saving SHALL move the row
there with its values, as edited, and show it in its new section. When the new section has an unopened row with the
same name and location, ignoring case, and expiry date, and the moved row isn't opened, the quantity SHALL be added to
that row instead.

#### Scenario: Move to the right section
- **WHEN** a user edits "Dishwasher tablets" in Pantry, chooses Supplies and saves
- **THEN** the row is in Supplies with its quantity, unit, category, location and dates, and no longer in Pantry

#### Scenario: Move onto the same product
- **WHEN** Supplies holds 1 "Descaler" expiring 14 November 2026 and a user moves 2 "descaler" with the same expiry
  date and location from Pantry to Supplies
- **THEN** Supplies holds one "Descaler" row with quantity 3, and Pantry has no "Descaler" row

#### Scenario: Different location moves apart
- **WHEN** Supplies holds "Descaler" kept in "Laundry" and a user moves "Descaler" kept in "Pantry" with the same
  expiry date from Pantry to Supplies
- **THEN** Supplies holds both rows, apart

#### Scenario: Opened row moves apart
- **WHEN** Supplies holds an unopened "Descaler" and a user moves an opened "Descaler" with the same expiry date
- **THEN** Supplies holds both rows, apart

#### Scenario: Edit and move together
- **WHEN** a user changes the quantity of "Detergent" to 2 and moves it to Supplies in the same save
- **THEN** the row is in Supplies with quantity 2

## ADDED Requirements

### Requirement: Move part of an item to another location
Each row SHALL offer to move some of it to another location of the same section, asking how much, 1 by default or the
whole quantity when it is less, and where to, suggesting the section's locations. The quantity follows the rules of
adding and SHALL NOT exceed the row's; the location SHALL differ from the row's, ignoring case.

#### Scenario: Move some
- **WHEN** "Pasta", 3, is kept in "Cellar" and a user moves 2 of it to "Kitchen"
- **THEN** the Pantry section shows "Pasta" with quantity 1 in "Cellar" and "Pasta" with quantity 2 in "Kitchen",
  both with the same values otherwise

#### Scenario: Move onto the same product
- **WHEN** "Pasta", 3, is kept in "Cellar", "Pasta", 1, with the same expiry date is kept in "Kitchen", and a user
  moves 2 from "Cellar" to "kitchen"
- **THEN** the Pantry section shows "Pasta" with quantity 1 in "Cellar" and "Pasta" with quantity 3 in "Kitchen"

#### Scenario: Move all of it
- **WHEN** "Pasta", 2, is kept in "Cellar" and a user moves 2 of it to "Kitchen"
- **THEN** the row is kept in "Kitchen", with quantity 2, and no "Pasta" row is left in "Cellar"

#### Scenario: Move all of it onto the same product
- **WHEN** "Pasta", 2, is kept in "Cellar", "Pasta", 1, with the same expiry date is kept in "Kitchen", and a user
  moves 2 from "Cellar" to "Kitchen"
- **THEN** the Pantry section shows one "Pasta" row, with quantity 3, in "Kitchen"

#### Scenario: Move a decimal quantity
- **WHEN** "Flour", 1.5 kg, is kept in "Cellar" and a user moves "0,5" of it to "Kitchen"
- **THEN** "Flour" shows quantity 1 in "Cellar" and 0.5 in "Kitchen"

#### Scenario: Opened part moves apart
- **WHEN** an opened "Passata", 1, is kept in "Kitchen", an unopened "Passata" with the same expiry date is kept in
  "Fridge", and a user moves the opened one to "Fridge"
- **THEN** the "Fridge" holds both rows, apart, the moved one still opened on the same day

#### Scenario: Default quantity
- **WHEN** a user starts moving "Pasta", 3, and then "Flour", 0.5
- **THEN** the quantity proposed is 1 for "Pasta" and 0.5 for "Flour"

#### Scenario: Locations suggested
- **WHEN** the pantry's products are kept in "Cellar" and "Kitchen" and a user starts moving a row
- **THEN** "Cellar" and "Kitchen" are suggested as where to move it

#### Scenario: Remembered where it went
- **WHEN** a user moves "Pasta" to "Kitchen" and later chooses the suggested name "Pasta" while adding an item
- **THEN** the location "Kitchen" is filled in

#### Scenario: Too much, or nothing
- **WHEN** a user moves 4 of a row with quantity 3, or a quantity that is 0 or less, has more than one decimal or is
  not a number
- **THEN** the system refuses, says the quantity is wrong, keeps what was entered, and the row is unchanged

#### Scenario: Same location
- **WHEN** a user moves "Pasta" kept in "Cellar" to "cellar"
- **THEN** the system refuses, says it is already kept there, keeps what was entered, and the row is unchanged
