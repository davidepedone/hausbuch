# Spec Delta

## REMOVED Requirements

### Requirement: Skeletons in the page's language
**Reason**: The interface is in English, and so are the skeletons: a page written in Italian renames its headings if
its author wants. Pages no longer have a language to choose a skeleton by.
**Migration**: None needed; pages created from an Italian skeleton keep their body.

### Requirement: Skeleton follows choices until the user writes
**Reason**: With no page language, the type is the only choice left; replaced by "Skeleton follows the type until the
user writes".
**Migration**: None needed.

## ADDED Requirements

### Requirement: Skeleton follows the type until the user writes
While creating a page, changing its type SHALL replace the body with the new type's skeleton as long as the user has
not changed the prefilled body. Once they have, the body SHALL stay as they wrote it.

#### Scenario: Type changed before writing
- **WHEN** a user starts a Recipe page, changes nothing in the body, and switches the type to How-to
- **THEN** the body becomes the How-to skeleton

#### Scenario: Type changed after writing
- **WHEN** a user has typed in the prefilled body and then switches the type
- **THEN** the body keeps exactly what they typed
