# page-search Specification

## Purpose

Lets residents find what they wrote down and what they keep at home without remembering where, from one search field:
by searching the words in page titles and bodies, and the names of stock items.

## Requirements

### Requirement: Search pages by words
Any signed-in user SHALL be able to search pages by words. A page SHALL match when it contains all the searched
words, in its title or body, ignoring case and accents and matching word forms in the page's language, or when each
searched word is part of its title, ignoring case and accents. Deleted pages SHALL NOT match.

#### Scenario: English word forms
- **WHEN** a user searches "descale" and the English page "Coffee machine" says "descaling every three months"
- **THEN** "Coffee machine" is among the results

#### Scenario: Italian word forms
- **WHEN** a user searches "irrigare" and the Italian page "Giardino" says "l'irrigazione parte ad aprile"
- **THEN** "Giardino" is among the results

#### Scenario: Searching without choosing a language
- **WHEN** a user searches "ricetta" and there are matching pages in Italian
- **THEN** they are found without the user saying which language they searched in

#### Scenario: All words must match
- **WHEN** a user searches "garden water" and a page contains "garden" but not "water"
- **THEN** that page is not among the results

#### Scenario: Part of a title
- **WHEN** a user searches "irri" and the page "Irrigation" doesn't contain "irri" as a word
- **THEN** "Irrigation" is among the results

#### Scenario: Deleted page
- **WHEN** a user searches a word that only a deleted page contains
- **THEN** there are no results

### Requirement: Search results
Results SHALL be grouped by where they come from: pages first, then each stock section in the sections' order, a group
without results not being shown. Pages SHALL be ordered with title matches before body-only matches, and each SHALL
show the page's title, type and a snippet of the text around the match.

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

#### Scenario: No results
- **WHEN** no page and no stock row matches the search
- **THEN** the system says nothing was found and offers to create a page titled with the searched text

### Requirement: Stock results
Stock rows found by a search SHALL be ordered as in their section, and each SHALL show its name, quantity, unit,
category, location and effective expiry date, marked when expired or expiring, and lead to the row in its section.

#### Scenario: Stock result
- **WHEN** a user searches "passata" and the pantry holds 3 bottles of "Passata" in "Long-life", kept in "Pantry, top
  shelf" and expiring on 30 September 2027
- **THEN** the result shows these values and leads to that row in the Pantry section

#### Scenario: Expiring stock result
- **WHEN** a user searches "milk" and the pantry's "Milk" expires tomorrow
- **THEN** its result is marked as expiring

### Requirement: Search is always at hand
A search field labelled "Search" SHALL be reachable from every screen of a signed-in user, on every screen size.

#### Scenario: Search from a page on a phone
- **WHEN** a user is reading a page on a phone-sized screen
- **THEN** they can start a search without first going back to the home page

#### Scenario: Label
- **WHEN** a signed-in user looks at the search field
- **THEN** it is labelled "Search"

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

### Requirement: Results while typing
While a user types in the search field, a panel under it SHALL show the results for the text typed so far, grouped as
in "Search results". Clearing the field or pressing Escape SHALL close the panel and leave the screen as it was, and
confirming the search SHALL open the full results page.

#### Scenario: Results appear while typing
- **WHEN** a user types "pas" in the search field without confirming
- **THEN** a panel under the field shows "Passata" and "Pasta" under "Pantry"

#### Scenario: Clear the field
- **WHEN** the panel shows results and the user clears the field
- **THEN** the panel closes and the screen underneath is unchanged

#### Scenario: Confirm the search
- **WHEN** a user types "pas" and confirms the search
- **THEN** the full results page for "pas" opens
