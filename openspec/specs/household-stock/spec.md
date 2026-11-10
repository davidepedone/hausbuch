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
and optionally a unit, a category, a location, an expiry date and a number of days it lasts once opened. The name SHALL
NOT be empty. The decimal separator SHALL be either a dot or a comma, and a whole quantity SHALL be shown without
decimals.

#### Scenario: New item
- **WHEN** a user adds "Passata", quantity 3, unit "bottles", location "Pantry, top shelf", expiry date 30 September
  2027
- **THEN** the Pantry section shows the row with these values and records the user as the one who added it

#### Scenario: Item with a category
- **WHEN** a user adds "Milk" with the category "Fresh"
- **THEN** the row is in the section's "Fresh" group

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
Right after a row is removed, by decreasing it to zero or by removing it, the system SHALL offer for a few seconds to
undo it or to add the product to the shopping list. Undoing SHALL bring the row back with all its values. Adding to the
list SHALL add an entry with the row's name, unit and section, as adding a row to the shopping list does, and leave the
row removed.

#### Scenario: Undo
- **WHEN** a user removes "Passata" by mistake and chooses to undo
- **THEN** the "Passata" row is back with its quantity, unit, category, location, dates and shelf life after opening

#### Scenario: No undo later
- **WHEN** a user removes a row and does not undo it
- **THEN** after a few seconds the offer to undo disappears and the row stays removed

#### Scenario: Used the last one
- **WHEN** a user decreases "Passata", in bottles, from 1 and chooses to add it to the shopping list
- **THEN** the list has "Passata", in bottles, under Pantry, to buy, and the row stays removed

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
The system SHALL remember, per section, every product name used and the category, location and shelf life after
opening it last had. While adding an item, names, categories and locations already used SHALL be suggested, and
choosing a known name SHALL fill in its category, location and shelf life after opening, which stay editable.

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

### Requirement: Group by category
A section where some item has a category SHALL list its items in groups, one per category in alphabetical order and an
"Other" group last for the items without one, each group ordered as the section is. Categories differing only in case
SHALL be one group. The section's chips SHALL be its categories, and "Other" when it has items without one.

#### Scenario: Grouped list
- **WHEN** the pantry holds "Milk" in "Fresh", "Rice" and "Passata" in "Long-life", and "Salt" without a category
- **THEN** the Pantry section shows a "Fresh" group with "Milk", a "Long-life" group with "Passata" and "Rice", and an
  "Other" group with "Salt", in this order

#### Scenario: Same category, different case
- **WHEN** "Milk" has the category "Fresh" and "Yogurt" the category "fresh"
- **THEN** both are in one group

#### Scenario: Chips of a section
- **WHEN** the pantry holds items in "Fresh" and "Long-life" and one without a category
- **THEN** its chips are "Fresh", "Long-life" and "Other"

#### Scenario: Choose a category
- **WHEN** a user chooses the "Fresh" chip
- **THEN** only the "Fresh" group is shown

#### Scenario: No categories
- **WHEN** no item of the section has a category
- **THEN** the section lists its items without groups and without chips

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

### Requirement: Add a row to the shopping list
Each row SHALL offer to add its product to the shopping list, leaving the row as it is. The entry SHALL have the row's
name, unit and section and no quantity, following the list's rule of one entry per product.

#### Scenario: Running low
- **WHEN** "Paracetamol" in Medicines has quantity 1 and a user adds it to the shopping list
- **THEN** the list has "Paracetamol" under Medicines, to buy, and the row still has quantity 1

#### Scenario: Already on the list
- **WHEN** "Rice", 2, in Pantry is to buy and a user adds the "Rice" row to the shopping list
- **THEN** there is one "Rice" entry to buy, with quantity 2

#### Scenario: Confirmed on the row
- **WHEN** a user adds a row to the shopping list
- **THEN** the section says the product was added to the list
