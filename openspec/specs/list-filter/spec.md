# list-filter Specification

## Purpose
Lets users narrow a long list down to the part they are looking for, the same way in every list: by choosing one of
the list's chips.

## Requirements

### Requirement: Filter by chip
Above a list SHALL be its chips and an "All" chip. Choosing a chip SHALL show only the entries it stands for; one chip
SHALL be chosen at a time, and "All" SHALL be chosen when the list is opened.

#### Scenario: Choose a chip
- **WHEN** a user chooses the Recipe chip on the wiki's page list
- **THEN** only recipes are shown

#### Scenario: Back to all
- **WHEN** a user chooses "All" after a chip
- **THEN** every entry is shown

### Requirement: Nothing matches
When the chosen chip leaves no entry to show, the list SHALL say that nothing matches instead of showing nothing.

#### Scenario: No match
- **WHEN** the wiki has no page of type "Where is it" and a user chooses that chip
- **THEN** the list says that nothing matches

### Requirement: The filter survives changes to the list
In a stock section, the chip chosen SHALL keep applying after a row is changed from the list, such as using one,
opening it, removing it or undoing a removal. When the change leaves no entry for that chip, "All" SHALL be chosen.

#### Scenario: Use one while filtering
- **WHEN** a user has chosen the "Long-life" chip and decreases "Passata"
- **THEN** the list still shows only the "Long-life" group

#### Scenario: Last entry of a chip removed
- **WHEN** a user has chosen the "Frozen" chip and removes its only item
- **THEN** the "Frozen" chip is gone and "All" is chosen
