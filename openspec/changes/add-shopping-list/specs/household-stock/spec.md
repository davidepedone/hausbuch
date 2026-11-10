# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

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
