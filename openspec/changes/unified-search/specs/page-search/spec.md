# Spec Delta

## ADDED Requirements

### Requirement: Search stock items by name
The search SHALL also look into every stock section. A row SHALL match when each searched word is part of its name,
ignoring case and accents. Only the name SHALL be searched.

#### Scenario: Part of a name
- **WHEN** the pantry holds "Passata", "Pasta" and "Rice" and a user searches "pas"
- **THEN** "Passata" and "Pasta" are among the results and "Rice" is not

#### Scenario: Case and accents
- **WHEN** the pantry holds "Caffè" and a user searches "CAFFE"
- **THEN** "Caffè" is among the results

#### Scenario: Every word must be in the name
- **WHEN** the pantry holds "Tomato passata" and "Passata" and a user searches "passata tomato"
- **THEN** "Tomato passata" is among the results and "Passata" is not

#### Scenario: Other fields are not searched
- **WHEN** "Milk" is kept in the "Fridge" and a user searches "fridge"
- **THEN** "Milk" is not among the results

## MODIFIED Requirements

### Requirement: Search results
Results SHALL be grouped by where they come from: pages first, then each stock section in the sections' order, a group
without results not being shown. Pages SHALL be ordered with title matches before body-only matches, and each SHALL
show the page's title, type and a snippet of the text around the match. Stock rows SHALL be ordered as in their
section, and each SHALL show its name, quantity, unit, category, location and effective expiry date, marked when
expired or expiring, and lead to the row in its section.

#### Scenario: Title match first
- **WHEN** a user searches "irrigation", the page "Irrigation" has it in its title and "Garden" only in its body
- **THEN** "Irrigation" is listed before "Garden"

#### Scenario: Snippet
- **WHEN** a result matches in the body
- **THEN** it shows a short excerpt of the body containing the matched word

#### Scenario: Grouped results
- **WHEN** a user searches "coffee", the page "Coffee machine" matches, the pantry holds "Coffee" and the supplies
  hold "Coffee machine descaler"
- **THEN** the results show the page under "Pages", then "Coffee" under "Pantry", then "Coffee machine descaler" under
  "Supplies", and no "Medicines" group

#### Scenario: Stock result
- **WHEN** a user searches "passata" and the pantry holds 3 bottles of "Passata" in "Long-life", kept in "Pantry, top
  shelf" and expiring on 30 September 2027
- **THEN** the result shows these values and leads to that row in the Pantry section

#### Scenario: Expiring stock result
- **WHEN** a user searches "milk" and the pantry's "Milk" expires tomorrow
- **THEN** its result is marked as expiring

#### Scenario: No results
- **WHEN** no page and no stock row matches the search
- **THEN** the system says nothing was found and offers to create a page titled with the searched text

### Requirement: Search is always at hand
A search field labelled "Search" SHALL be reachable from every screen of a signed-in user, on every screen size.

#### Scenario: Search from a page on a phone
- **WHEN** a user is reading a page on a phone-sized screen
- **THEN** they can start a search without first going back to the home page

#### Scenario: Label
- **WHEN** a signed-in user looks at the search field
- **THEN** it is labelled "Search"
