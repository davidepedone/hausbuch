# shopping-list Specification

## Purpose
Keeps the household's shopping list: what needs buying, ticking it off while shopping, and putting what was bought
away into the stock with as little typing as possible.

## Requirements

### Requirement: Shopping list page
The system SHALL have one shopping list shared by the whole household, reached from a tile on the home page. Entries
still to buy SHALL be listed first, in groups by stock section in the sections' order and then a group for entries
without a section, each ordered by name; bought entries SHALL be listed after them, apart.

#### Scenario: Tile on the home page
- **WHEN** a user opens the home page
- **THEN** it shows a "Shopping list" tile leading to the list

#### Scenario: Grouped list
- **WHEN** the list holds "Paracetamol" in Medicines, "Rice" and "Passata" in Pantry, and "Bread" with no section
- **THEN** it shows "Paracetamol" under Medicines, then "Passata" and "Rice" under Pantry, then "Bread" under Other

#### Scenario: Bought entries apart
- **WHEN** "Rice" is bought and "Passata" is not
- **THEN** "Passata" is listed to buy and "Rice" is listed under Bought

#### Scenario: Shared by the household
- **WHEN** one user adds "Rice" to the list
- **THEN** another user opening the list sees "Rice"

#### Scenario: Empty list
- **WHEN** the list has no entries
- **THEN** it says there is nothing to buy

### Requirement: List kept up to date
While the list is open, it SHALL show other users' changes within about ten seconds without being reloaded, keeping
the chosen shop chip and anything typed in the add form.

#### Scenario: Bought by someone else
- **WHEN** two users have the list open and one ticks "Butter"
- **THEN** within about ten seconds the other user's list shows "Butter" under Bought

#### Scenario: Typing meanwhile
- **WHEN** a user is typing a new entry and the list refreshes
- **THEN** what was typed stays in the form

### Requirement: Add an entry
Any signed-in user SHALL be able to add an entry with a name and optionally a quantity, a unit, one of the stock
sections and a shop. The name SHALL NOT be empty, and a quantity SHALL follow the same rules as in the stock.

#### Scenario: Entry for the stock
- **WHEN** a user adds "Passata", quantity 3, unit "bottles", section Pantry
- **THEN** the list shows "Passata", 3 bottles, under Pantry, to buy

#### Scenario: Entry without a section
- **WHEN** a user adds "Bread" with no quantity and no section
- **THEN** the list shows "Bread" under Other

#### Scenario: Refused entry
- **WHEN** a user adds an entry with an empty name, or a quantity that is 0 or less, has more than one decimal or is
  not a number
- **THEN** the system refuses, says which field is wrong, and keeps what was entered

### Requirement: One entry per product
The list SHALL hold at most one entry to buy and one bought entry per name, ignoring case, and section. Adding,
ticking or unticking onto a matching entry SHALL keep that one entry: when both have a quantity, the quantities SHALL be
added; otherwise it SHALL keep the quantity and unit it has, or take the other's when it has none. A shop given when
adding SHALL replace the entry's.

#### Scenario: Added twice
- **WHEN** "Rice", 1, in Pantry is to buy and a user adds "rice", 2, in Pantry
- **THEN** there is one "Rice" entry with quantity 3

#### Scenario: Added again without a quantity
- **WHEN** "Rice", 1, in Pantry is to buy and a user adds "Rice" in Pantry with no quantity
- **THEN** there is one "Rice" entry with quantity 1

#### Scenario: Already bought
- **WHEN** "Rice" in Pantry is bought and a user adds "Rice" in Pantry
- **THEN** there is a "Rice" entry to buy and a "Rice" entry under Bought

#### Scenario: Different section
- **WHEN** "Soap" in Supplies is to buy and a user adds "Soap" with no section
- **THEN** there are two "Soap" entries, one under Supplies and one under Other

### Requirement: Suggestions while adding an entry
While adding an entry, the product names already used in any stock section or on the list SHALL be suggested, and the
shops already used. Choosing a name used in exactly one section SHALL fill in that section, and a name with a
remembered shop SHALL fill in that shop; both stay editable.

#### Scenario: Known product
- **WHEN** "Passata" was used only in Pantry and a user starts adding an entry and chooses the suggested "Passata"
- **THEN** the section Pantry is filled in

#### Scenario: Name used in two sections
- **WHEN** "Soap" was used in Pantry and in Supplies and a user chooses "Soap"
- **THEN** no section is filled in

#### Scenario: Shop suggestions
- **WHEN** entries were bought at "Migros" and "Pharmacy" and a user starts typing a shop
- **THEN** "Migros" and "Pharmacy" are suggested

### Requirement: Shop remembered per product
The system SHALL remember, for each name used on the list, ignoring case, the shop its entry last had, also after the
entry is gone. An entry added without a shop, by hand or from the stock, SHALL take the remembered one.

#### Scenario: Bought there again
- **WHEN** "Butter" was on the list with the shop "Migros", was put away, and a user adds "butter" from the Pantry row
- **THEN** the new entry has the shop "Migros"

#### Scenario: Shop changed
- **WHEN** "Butter" was last listed with "Migros" and a user adds "Butter" with the shop "Coop"
- **THEN** the entry has "Coop", and "Coop" is remembered for "Butter"

### Requirement: Filter by shop
When some entry has a shop, the list SHALL show a chip per shop used by its entries and an "All" chip. Choosing a shop
SHALL show only its entries and the entries without a shop, in the same groups; choosing "All" SHALL show every entry.

#### Scenario: At the pharmacy
- **WHEN** "Paracetamol" has the shop "Pharmacy", "Butter" has "Migros" and "Bread" has none, and a user chooses
  "Pharmacy"
- **THEN** the list shows "Paracetamol" and "Bread", and not "Butter"

#### Scenario: Same shop, different case
- **WHEN** one entry has the shop "Migros" and another "migros"
- **THEN** there is one "Migros" chip, showing both

#### Scenario: No shops
- **WHEN** no entry has a shop
- **THEN** the list shows no chips

### Requirement: Tick an entry as bought
Each entry still to buy SHALL offer a way to mark it as bought, and each bought entry a way to mark it as still to buy.

#### Scenario: Tick
- **WHEN** a user ticks "Rice" while shopping
- **THEN** "Rice" moves under Bought

#### Scenario: Untick
- **WHEN** a user unticks the bought "Rice"
- **THEN** "Rice" is back among the entries to buy

#### Scenario: Ticked twice
- **WHEN** "Rice", 1, in Pantry is to buy, a bought "Rice", 2, in Pantry exists, and a user ticks the first one
- **THEN** there is one bought "Rice" entry with quantity 3

### Requirement: Put a bought entry away
Each bought entry with a section SHALL offer to put it away: a form to add an item to that section, filled in with the
entry's name, quantity and unit and with the category, location and shelf life after opening the product last had.
Saving SHALL add the item as adding it to the section does and take the entry off the list.

#### Scenario: Only the expiry left
- **WHEN** a bought "Passata", 3 bottles, in Pantry is put away, and "Passata" was last kept in "Pantry, top shelf",
  in "Long-life", lasting 4 days once opened
- **THEN** the form shows "Passata", 3, "bottles", "Long-life", "Pantry, top shelf" and 4 days, with no expiry date

#### Scenario: Saved
- **WHEN** a user puts away "Passata" with expiry date 30 September 2027 and saves
- **THEN** the Pantry section has the "Passata" row and the entry is no longer on the list

#### Scenario: Added to an existing row
- **WHEN** "Passata", 2, in "Pantry, top shelf", expiring 30 September 2027 exists and a user puts away "Passata", 3,
  with the same location and expiry date
- **THEN** there is one "Passata" row with quantity 5

#### Scenario: Entry without a quantity
- **WHEN** a bought "Rice" in Pantry has no quantity and is put away
- **THEN** the form's quantity is 1

#### Scenario: Refused form
- **WHEN** a user puts away an entry and empties the name
- **THEN** the system refuses, says which field is wrong, keeps what was entered, and the entry stays on the list

#### Scenario: No section
- **WHEN** a bought entry has no section
- **THEN** it offers no way to put it away

### Requirement: Remove an entry
Any signed-in user SHALL be able to remove an entry, bought or not.

#### Scenario: No longer needed
- **WHEN** a user removes "Rice" from the list
- **THEN** "Rice" is no longer on the list

### Requirement: Clear bought entries without a section
When some bought entry has no section, the list SHALL offer one action that removes every bought entry without a
section, leaving the others.

#### Scenario: Clear
- **WHEN** "Bread" with no section and "Passata" in Pantry are bought and a user clears
- **THEN** "Bread" is no longer on the list and "Passata" is still under Bought

#### Scenario: Nothing to clear
- **WHEN** every bought entry has a section
- **THEN** the list offers no clearing action

### Requirement: Check a recipe against the pantry
A recipe page SHALL offer "Check the pantry", which lists the recipe's ingredients: the list items under the body's
"Ingredients" heading, ignoring case, up to the next heading of the same or a higher level, in order. Each ingredient
SHALL be shown as in the pantry, with the names, locations and quantities of the rows it matches; otherwise as on the shopping
list, when it matches an entry not yet bought; otherwise as missing. Pages of other types SHALL NOT offer the check.

#### Scenario: In the pantry
- **WHEN** the pantry has a "Flour" row of 100 g kept in "Cupboard" and a recipe lists "200 g flour"
- **THEN** the check shows "200 g flour" as in the pantry, as "Flour" in "Cupboard", 100 g

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

### Requirement: Age of a matched row
On the check, a matched pantry row not changed for three months or more SHALL show how many whole months ago it last
changed, as it may have been used up without the stock being updated.

#### Scenario: Not changed for a while
- **WHEN** the "Flour" row a recipe's "200 g flour" matches was last changed 8 months ago
- **THEN** the check shows that the row changed 8 months ago

#### Scenario: Changed recently
- **WHEN** the "Flour" row a recipe's "200 g flour" matches was last changed 2 months ago
- **THEN** the check shows no age for the row

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
