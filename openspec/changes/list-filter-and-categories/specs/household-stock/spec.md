# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

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

### Requirement: Undo a removal
Right after a row is removed, by decreasing it to zero or by removing it, the system SHALL offer to undo it for a few
seconds. Undoing SHALL bring the row back with all its values.

#### Scenario: Undo
- **WHEN** a user removes "Passata" by mistake and chooses to undo
- **THEN** the "Passata" row is back with its quantity, unit, category, location, dates and shelf life after opening

#### Scenario: No undo later
- **WHEN** a user removes a row and does not undo it
- **THEN** after a few seconds the offer to undo disappears and the row stays removed

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
