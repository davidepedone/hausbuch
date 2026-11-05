# page-search Specification

## Purpose

Lets residents find what they wrote down without remembering where, by searching the words in page titles and bodies.

## Requirements

### Requirement: Search pages by words
Any signed-in user SHALL be able to search pages by words. A page SHALL match when it contains all the searched
words, in its title or body, ignoring case and accents and matching word forms in the page's language. Deleted pages
SHALL NOT match.

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

#### Scenario: Deleted page
- **WHEN** a user searches a word that only a deleted page contains
- **THEN** there are no results

### Requirement: Search results
Results SHALL be ordered with title matches before body-only matches, and each result SHALL show the page's title,
type and a snippet of the text around the match.

#### Scenario: Title match first
- **WHEN** a user searches "irrigation", the page "Irrigation" has it in its title and "Garden" only in its body
- **THEN** "Irrigation" is listed before "Garden"

#### Scenario: Snippet
- **WHEN** a result matches in the body
- **THEN** it shows a short excerpt of the body containing the matched word

#### Scenario: No results
- **WHEN** no page matches the search
- **THEN** the system says nothing was found and offers to create a page titled with the searched text

### Requirement: Search is always at hand
A search field SHALL be reachable from every screen of a signed-in user, on every screen size.

#### Scenario: Search from a page on a phone
- **WHEN** a user is reading a page on a phone-sized screen
- **THEN** they can start a search without first going back to the home page
