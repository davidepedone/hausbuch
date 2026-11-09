# Spec Delta

## MODIFIED Requirements

### Requirement: List pages
The home page SHALL list all pages, most recently changed first, showing each page's title, type and when and by whom
it was last changed. The list SHALL be filterable with the list filter, its chips being the page types.

#### Scenario: Home page
- **WHEN** a user opens the home page
- **THEN** they see every page that is not deleted, most recently changed first

#### Scenario: Filter by type
- **WHEN** a user chooses the Recipe chip
- **THEN** only pages of type Recipe are listed

#### Scenario: Chips of the page list
- **WHEN** a user opens the home page
- **THEN** the page list's chips are "All" and one per page type, with its icon
