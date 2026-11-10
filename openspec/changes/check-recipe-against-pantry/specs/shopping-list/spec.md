# Spec Delta

## ADDED Requirements

### Requirement: Check a recipe against the pantry
A recipe page SHALL offer "Check the pantry", which lists the recipe's ingredients: the list items under the body's
"Ingredients" heading, ignoring case, up to the next heading of the same or a higher level, in order. Each ingredient
SHALL be shown as in the pantry, with the names and locations of the rows it matches; otherwise as on the shopping
list, when it matches an entry not yet bought; otherwise as missing. Pages of other types SHALL NOT offer the check.

#### Scenario: In the pantry
- **WHEN** the pantry has a "Flour" row kept in "Cupboard" and a recipe lists "200 g flour"
- **THEN** the check shows "200 g flour" as in the pantry, as "Flour" in "Cupboard"

#### Scenario: Missing
- **WHEN** no pantry row and no entry matches "Eggs" and a recipe lists "Eggs"
- **THEN** the check shows "Eggs" as missing

#### Scenario: Already on the list
- **WHEN** the shopping list has "Butter" not yet bought, no pantry row matches it, and a recipe lists "50 g butter"
- **THEN** the check shows "50 g butter" as on the shopping list

#### Scenario: Only the Ingredients list
- **WHEN** a recipe has list items under "Ingredients" and under "Steps"
- **THEN** the check lists only those under "Ingredients"

#### Scenario: No ingredients
- **WHEN** a recipe has no "Ingredients" heading, or no list items under it
- **THEN** the check says the recipe lists no ingredients

#### Scenario: Not a recipe
- **WHEN** a user views a how-to page
- **THEN** no "Check the pantry" is offered

### Requirement: Ingredients matched by their words
An ingredient SHALL match a pantry row or an entry when, ignoring case, the two names share a word, a word being a run
of letters and digits. Words holding a digit and filler words such as "a", "of", "the", "di" and "con" SHALL not count.
Quantities SHALL never be compared.

#### Scenario: Quantity words ignored
- **WHEN** the pantry has a "Flour" row and a recipe lists "200 g flour"
- **THEN** the ingredient matches the row

#### Scenario: Shorter ingredient
- **WHEN** the pantry has an "Olive oil" row and a recipe lists "Oil"
- **THEN** the check shows "Oil" as in the pantry

#### Scenario: Longer ingredient
- **WHEN** the pantry has an "Olive oil" row and a recipe lists "1 tbsp oil"
- **THEN** the check shows "1 tbsp oil" as in the pantry

#### Scenario: Filler words don't match
- **WHEN** the pantry has a "Bag of rice" row and a recipe lists "A pinch of salt"
- **THEN** the ingredient doesn't match the row

### Requirement: Add a recipe's missing ingredients
The check SHALL let the user choose ingredients to add to the shopping list: each SHALL have a tick, ticked for the
missing ones and unticked for the others, and a name field filled in with the ingredient as written. "Add to shopping
list" SHALL add each ticked ingredient, under the name in its field, as a Pantry entry without a quantity, merging as
any added entry does, and then show the shopping list. Nothing SHALL be added without that action.

#### Scenario: Add the missing ones
- **WHEN** the check shows "Eggs" as missing and "200 g flour" as in the pantry, and the user chooses "Add to shopping
  list" without changing anything
- **THEN** the shopping list has a Pantry entry "Eggs", and no entry for the flour

#### Scenario: Rename before adding
- **WHEN** the check shows "3 eggs" as missing and the user changes its name to "Eggs" and adds it
- **THEN** the shopping list has a Pantry entry "Eggs"

#### Scenario: Untick a missing one
- **WHEN** the check shows "Salt" and "Eggs" as missing and the user unticks "Salt" and adds
- **THEN** only "Eggs" is added

#### Scenario: Nothing ticked
- **WHEN** the user unticks every ingredient and adds
- **THEN** nothing is added and the check is shown again, saying that nothing was chosen
