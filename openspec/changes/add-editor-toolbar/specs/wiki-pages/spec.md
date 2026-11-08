# Spec Delta

## ADDED Requirements

### Requirement: Formatting buttons in the editor
The page editor SHALL offer buttons for an external link, a link to a page, bold, a heading, a bulleted list and a
checklist. Each SHALL insert the Markdown for it into the body: around the selected text, at the cursor when nothing
is selected, or at the start of each selected line for a heading or a list. The body SHALL stay plain Markdown text.

#### Scenario: Bold selected text
- **WHEN** a user selects "boiling" in the body and presses the bold button
- **THEN** the body reads `**boiling**` in its place

#### Scenario: Bold with nothing selected
- **WHEN** a user presses the bold button with nothing selected
- **THEN** `****` is inserted at the cursor and the cursor is placed between the two pairs of asterisks

#### Scenario: Link to a page
- **WHEN** a user selects "Coffee machine" and presses the page link button
- **THEN** the body reads `[[Coffee machine]]` in its place

#### Scenario: List from selected lines
- **WHEN** a user selects three lines and presses the bulleted list button
- **THEN** each of the three lines starts with `- `

#### Scenario: Checklist on the current line
- **WHEN** the cursor is in a line "Buy salt" with nothing selected and the user presses the checklist button
- **THEN** the line reads `- [ ] Buy salt`

#### Scenario: Heading
- **WHEN** the cursor is in a line "Ingredients" and the user presses the heading button
- **THEN** the line reads `## Ingredients`

#### Scenario: Usable by touch
- **WHEN** a user edits a page on a phone-sized screen
- **THEN** every formatting button is visible without scrolling the page sideways and can be pressed by touch

### Requirement: External links from the editor
The external link button SHALL ask for an address and insert a Markdown link to it, whose text is the selected text,
or the address itself when nothing is selected. An address without a scheme SHALL be given `https://`. Cancelling or
giving an empty address SHALL leave the body unchanged.

#### Scenario: Link selected text
- **WHEN** a user selects "the manual", presses the link button and gives `https://example.com/manual.pdf`
- **THEN** the body reads `[the manual](https://example.com/manual.pdf)` in its place

#### Scenario: Link with nothing selected
- **WHEN** a user presses the link button with nothing selected and gives `https://example.com`
- **THEN** `[https://example.com](https://example.com)` is inserted at the cursor

#### Scenario: Address without scheme
- **WHEN** a user selects "the shop" and gives `example.com`
- **THEN** the body reads `[the shop](https://example.com)` in its place

#### Scenario: Cancel
- **WHEN** a user presses the link button and cancels
- **THEN** the body is unchanged

### Requirement: Paste an address over selected text
Pasting a single `http://` or `https://` address while text is selected in the body SHALL turn the selected text into a
link to that address. Any other paste SHALL replace the selection as usual.

#### Scenario: Paste a link
- **WHEN** a user selects "the manual" and pastes `https://example.com/manual.pdf`
- **THEN** the body reads `[the manual](https://example.com/manual.pdf)` in its place

#### Scenario: Paste ordinary text
- **WHEN** a user selects "the manual" and pastes "the booklet"
- **THEN** the body reads "the booklet" in its place
