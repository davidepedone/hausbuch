# Spec Delta

## MODIFIED Requirements

### Requirement: Search pages by words
Any signed-in user SHALL be able to search pages by words. A page SHALL match when it contains all the searched
words, in its title or body, ignoring case and accents and matching their English and Italian word forms, or when each
searched word is part of its title, ignoring case and accents. Deleted pages SHALL NOT match.

#### Scenario: English word forms
- **WHEN** a user searches "descale" and the page "Coffee machine" says "descaling every three months"
- **THEN** "Coffee machine" is among the results

#### Scenario: Italian word forms
- **WHEN** a user searches "irrigare" and the page "Giardino" says "l'irrigazione parte ad aprile"
- **THEN** "Giardino" is among the results

#### Scenario: Searching without choosing a language
- **WHEN** a user searches "ricetta" and there are pages saying "ricette"
- **THEN** they are found without anyone saying which language the search or the pages are in

#### Scenario: All words must match
- **WHEN** a user searches "garden water" and a page contains "garden" but not "water"
- **THEN** that page is not among the results

#### Scenario: Part of a title
- **WHEN** a user searches "irri" and the page "Irrigation" doesn't contain "irri" as a word
- **THEN** "Irrigation" is among the results

#### Scenario: Deleted page
- **WHEN** a user searches a word that only a deleted page contains
- **THEN** there are no results
