# Spec Delta

## MODIFIED Requirements

### Requirement: Suggestions from products already used
The system SHALL remember, per section, every product name used and the category, location and shelf life after
opening it last had. While adding an item, names, categories and locations already used SHALL be suggested, and
choosing a known name SHALL fill in its category, location and shelf life after opening, which stay editable. Wherever a
category is entered, the section's common categories SHALL also be suggested: Fresh, Long-life, Frozen and Drinks in
Pantry; Pain & fever, Cold & flu, Stomach and First aid in Medicines; Cleaning, Laundry, Paper & bags and Personal care
in Supplies. A category already used with the same name, ignoring case, SHALL be suggested once, as the household spelt
it.

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

#### Scenario: Common categories in an empty section
- **WHEN** Supplies has never had an item and a user starts typing a category there
- **THEN** "Cleaning", "Laundry", "Paper & bags" and "Personal care" are suggested

#### Scenario: Common and own categories together
- **WHEN** the pantry has items in "Spices" and in "fresh" and a user starts typing a category
- **THEN** "Drinks", "fresh", "Frozen", "Long-life" and "Spices" are suggested, "fresh" once

#### Scenario: Suggested on the edit page
- **WHEN** a user edits a row of Medicines
- **THEN** the category field suggests "Pain & fever", "Cold & flu", "Stomach" and "First aid"

#### Scenario: Only suggested
- **WHEN** no pantry row has the category "Frozen"
- **THEN** the Pantry section shows no "Frozen" group or chip
