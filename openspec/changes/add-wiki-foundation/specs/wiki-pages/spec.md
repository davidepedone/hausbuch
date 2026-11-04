# Spec Delta

## Purpose

Lets residents write down what the household needs to know as pages of formatted text, link them together, and keep
them up to date.

## ADDED Requirements

### Requirement: Create a page
Any signed-in user SHALL be able to create a page with a title and a body. Titles SHALL be unique ignoring case and
surrounding whitespace, and SHALL NOT be empty.

#### Scenario: New page
- **WHEN** a user creates a page with an unused title and a body
- **THEN** the page exists, is shown to them, and records them as its creator

#### Scenario: Title already used
- **WHEN** a user creates a page whose title matches an existing page's ignoring case and surrounding whitespace
- **THEN** the system refuses and links to the existing page

#### Scenario: Empty title
- **WHEN** a user creates a page with an empty or whitespace-only title
- **THEN** the system refuses and asks for a title

### Requirement: Formatted text in Markdown
Page bodies SHALL be written in Markdown and shown formatted: headings, bold and italic, bulleted and numbered lists,
checklists, tables, links and code. Raw HTML in a body SHALL NOT be rendered as HTML.

#### Scenario: Formatted body
- **WHEN** a page body contains Markdown headings, lists and a table
- **THEN** the page shows them formatted

#### Scenario: HTML in a body
- **WHEN** a page body contains an HTML tag such as a script element
- **THEN** the page shows it as text and no script runs

### Requirement: Links between pages
A body SHALL be able to link another page by its title as `[[Title]]`, or `[[Title|shown text]]`. A link to an
existing page SHALL lead to it; a link to a title with no page SHALL look different and lead to creating that page.

#### Scenario: Link to an existing page
- **WHEN** a page body contains `[[Coffee machine]]` and a page titled "Coffee machine" exists
- **THEN** the viewed page shows a link to that page

#### Scenario: Link to a missing page
- **WHEN** a page body links a title for which no page exists
- **THEN** the link is shown as missing and following it opens the creation of a page with that title

### Requirement: Pages linking here
Viewing a page SHALL show the other pages that link to it.

#### Scenario: Backlinks
- **WHEN** pages "Pasta" and "Weekly menu" both link to "Tomato sauce"
- **THEN** the "Tomato sauce" page lists "Pasta" and "Weekly menu" as linking to it

### Requirement: Edit a page
Any signed-in user SHALL be able to edit the title and body of any page. Renaming SHALL follow the same title rules
as creation, and links using a former title SHALL keep leading to the renamed page.

#### Scenario: Edit the body
- **WHEN** a user changes a page's body and saves
- **THEN** the page shows the new body

#### Scenario: Rename a page
- **WHEN** a user renames "Coffee maker" to "Coffee machine" and another page links `[[Coffee maker]]`
- **THEN** that link leads to the renamed page

### Requirement: List pages
The home page SHALL list all pages, most recently changed first, showing each page's title, type and when and by whom
it was last changed. The list SHALL be filterable by page type.

#### Scenario: Home page
- **WHEN** a user opens the home page
- **THEN** they see every page that is not deleted, most recently changed first

#### Scenario: Filter by type
- **WHEN** a user filters the list by the Recipe type
- **THEN** only pages of type Recipe are listed

### Requirement: Delete and restore a page
Any signed-in user SHALL be able to delete a page. A deleted page SHALL disappear from the list, from search and from
links, which show as missing, and SHALL be listed among deleted pages, from where any user can restore it with its
history.

#### Scenario: Delete a page
- **WHEN** a user deletes a page
- **THEN** it no longer appears in the page list or search results, and links to it show as missing

#### Scenario: Restore a page
- **WHEN** a user restores a page from the deleted pages
- **THEN** the page is back with its body and full history

#### Scenario: Title of a deleted page
- **WHEN** a user creates a page whose title matches a deleted page's
- **THEN** the system refuses and offers to restore the deleted page

### Requirement: Deleted pages are removed after 30 days
A page that has stayed deleted for 30 days SHALL be removed for good, with its history. The deleted pages list SHALL
show for each page who deleted it, when, and when it will be removed.

#### Scenario: Removal after 30 days
- **WHEN** a page was deleted more than 30 days ago and has not been restored
- **THEN** it is no longer among the deleted pages, cannot be restored, and its title can be used for a new page

#### Scenario: Restored before removal
- **WHEN** a page deleted 29 days ago is restored
- **THEN** it is not removed when the 30 days would have passed

#### Scenario: Deleted pages list
- **WHEN** a user opens the deleted pages
- **THEN** each page shows who deleted it, when, and the date it will be removed

### Requirement: Usable on any screen size
Every screen SHALL be usable on a phone, a tablet and a desktop browser without horizontal scrolling of the page,
including reading and editing a page.

#### Scenario: Editing on a phone
- **WHEN** a user edits a page on a phone-sized screen
- **THEN** the editor and its controls fit the screen width and can be used by touch
