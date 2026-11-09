# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: Medicines and Pantry sections`
- TO: `### Requirement: Stock sections`

## MODIFIED Requirements

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
