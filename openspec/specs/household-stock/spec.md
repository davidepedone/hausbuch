# household-stock Specification

## Purpose

Keeps track of the medicines, the food and the household supplies the household has at home: what, how much, where
it is kept and when it expires, so that anyone can check it and keep it up to date.

## Requirements

### Requirement: Stock sections
The system SHALL have three sections, Medicines, Pantry and Supplies, each listing its own items and each reached from
a tile of its own on the home page. Items SHALL be listed by name, and rows with the same name SHALL be shown together,
ordered by effective expiry date, rows without one last.

#### Scenario: Items listed by name
- **WHEN** the pantry holds "Rice", "Passata" with two expiry dates, and "Salt" without one
- **THEN** the Pantry section lists "Passata", "Rice" and "Salt" in this order, the two "Passata" rows next to each
  other with the earlier expiry date first

#### Scenario: Sections are separate
- **WHEN** a user adds an item to Medicines
- **THEN** it appears in Medicines and not in Pantry

#### Scenario: Supplies kept apart from food
- **WHEN** a user adds "Dishwasher tablets" to Supplies
- **THEN** it appears in Supplies and not in Pantry

#### Scenario: Sections on the home page
- **WHEN** a user opens the home page
- **THEN** it shows a tile for Medicines, one for Pantry and one for Supplies, each leading to its section

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

### Requirement: One row per expiry date
The same product with different expiry dates SHALL be kept as separate rows. Adding a product whose name, ignoring
case, and expiry date match an unopened row of the same section SHALL add to that row's quantity instead.

#### Scenario: Same product, different expiry
- **WHEN** "Paracetamol" expiring in March 2027 exists and a user adds "Paracetamol" expiring in November 2026
- **THEN** the Medicines section shows two "Paracetamol" rows, one per expiry date

#### Scenario: Same product, same expiry
- **WHEN** "Passata", 2, expiring 30 September 2027 exists and a user adds "passata", 1, expiring the same day
- **THEN** there is one "Passata" row with quantity 3

#### Scenario: Opened row stays apart
- **WHEN** an opened "Passata" expiring 30 September 2027 exists and a user adds "Passata" expiring the same day
- **THEN** the new bottle is a row of its own, not opened

### Requirement: Edit an item
Any signed-in user SHALL be able to change every field of a row, following the same rules as adding it. The row SHALL
record who changed it last and when.

#### Scenario: Change the location
- **WHEN** a user moves "Passata" from "Pantry" to "Cellar"
- **THEN** the row shows "Cellar" and the user as the one who changed it last

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

### Requirement: Remove an item
Any signed-in user SHALL be able to remove a row, whatever its quantity. A removed row SHALL be deleted for good, with
no list of removed items.

#### Scenario: Throw something away
- **WHEN** a user removes an expired "Yogurt" row with quantity 2
- **THEN** the row no longer appears in the section

### Requirement: Undo a removal
Right after a row is removed, by decreasing it to zero or by removing it, the system SHALL offer to undo it for a few
seconds. Undoing SHALL bring the row back with all its values.

#### Scenario: Undo
- **WHEN** a user removes "Passata" by mistake and chooses to undo
- **THEN** the "Passata" row is back with its quantity, unit, location, dates and shelf life after opening

#### Scenario: No undo later
- **WHEN** a user removes a row and does not undo it
- **THEN** after a few seconds the offer to undo disappears and the row stays removed

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

### Requirement: Effective expiry date
An opened row with a shelf life after opening SHALL expire on the earlier of its printed expiry date and its opening
date plus the shelf life, or on the latter when it has no printed expiry date. Any other row SHALL expire on its
printed expiry date, if any.

#### Scenario: Opening comes first
- **WHEN** "Passata" expiring 30 September 2027 and lasting 4 days once opened is opened on 6 November 2026
- **THEN** its effective expiry date is 10 November 2026

#### Scenario: Printed date comes first
- **WHEN** "Eye drops" expiring 15 November 2026 and lasting 28 days once opened is opened on 6 November 2026
- **THEN** its effective expiry date is 15 November 2026

#### Scenario: No printed date
- **WHEN** "Pesto" without an expiry date and lasting 5 days once opened is opened on 6 November 2026
- **THEN** its effective expiry date is 11 November 2026

### Requirement: Suggestions from products already used
The system SHALL remember, per section, every product name used and the location and shelf life after opening it last
had. While adding an item, names and locations already used SHALL be suggested, and choosing a known name SHALL fill
in its location and shelf life after opening, which stay editable.

#### Scenario: Buy something again
- **WHEN** a "Passata" row kept in "Pantry, top shelf" and lasting 4 days once opened was removed, and a user starts
  adding an item and chooses the suggested name "Passata"
- **THEN** the location "Pantry, top shelf" and the shelf life of 4 days are filled in

#### Scenario: Suggestions stay in their section
- **WHEN** "Paracetamol" was used in Medicines
- **THEN** it is not suggested while adding an item to Pantry
