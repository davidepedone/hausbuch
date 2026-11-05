# Spec Delta

## Purpose

Gives each kind of content a ready-made starting skeleton, so pages of the same kind look alike and nothing important
is forgotten.

## ADDED Requirements

### Requirement: Page types
Every page SHALL have exactly one type, chosen when it is created, from those the application provides: Blank,
Recipe, How-to and Where is it.

#### Scenario: Choosing a type
- **WHEN** a user starts creating a page
- **THEN** they choose one of Blank, Recipe, How-to or Where is it, with Blank preselected

### Requirement: Template skeletons
Choosing a type SHALL prefill the new page's body with that type's skeleton, which the user edits freely before
saving. Blank SHALL start empty. Recipe SHALL provide sections for servings, time, ingredients, steps and notes.
How-to SHALL provide sections for when to do it, what is needed, steps and notes. Where is it SHALL provide sections
for the location, how to find or reach it and notes.

#### Scenario: Recipe skeleton
- **WHEN** a user starts creating a page of type Recipe
- **THEN** the body is prefilled with headings for servings, time, ingredients, steps and notes

#### Scenario: Skeleton is free text
- **WHEN** a user removes or adds sections in a prefilled body and saves
- **THEN** the page is saved with exactly the body they wrote

### Requirement: Skeletons in the page's language
Each skeleton SHALL exist in English and Italian, and a new page SHALL be prefilled in its own language. In Italian,
Recipe SHALL use Porzioni, Tempo, Ingredienti, Procedimento and Note; How-to Quando, Cosa serve, Procedimento and
Note; Where is it Dove si trova, Come trovarlo and Note.

#### Scenario: Italian recipe
- **WHEN** a user starts a new Italian page of type Recipe
- **THEN** the body is prefilled with the headings Porzioni, Tempo, Ingredienti, Procedimento and Note

### Requirement: Skeleton follows choices until the user writes
While creating a page, changing its type or language SHALL replace the body with the matching skeleton as long as the
user has not changed the prefilled body. Once they have, the body SHALL stay as they wrote it.

#### Scenario: Language changed before writing
- **WHEN** a user starts an English Recipe page, changes nothing in the body, and switches the language to Italian
- **THEN** the body becomes the Italian Recipe skeleton

#### Scenario: Language changed after writing
- **WHEN** a user has typed in the prefilled body and then switches the language or the type
- **THEN** the body keeps exactly what they typed

### Requirement: Changing a page's type
A user SHALL be able to change the type of an existing page. Changing the type SHALL NOT change the body.

#### Scenario: Change type
- **WHEN** a user changes a How-to page to Where is it
- **THEN** the page is listed under Where is it and its body is unchanged
