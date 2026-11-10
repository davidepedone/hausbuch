# Tasks

## 1. Store

- [x] 1.1 Add the common categories per section and merge them into `Categories`; verify store tests for "Common categories in an empty section", "Common and own categories together" and "Suggestions stay in their section", with `TestCategorySuggestions` updated to the merged list

## 2. Web interface

- [x] 2.1 Verify web tests for "Suggested on the edit page", the add form's datalist holding a section's common categories, and "Only suggested" (no group or chip for an unused common category), with the existing section and put-away tests still green

- [x] 2.2 Show the category suggestions as choices under the category field of the add form, the edit page and the
  put-away page, filling the field when chosen; verify a web test for the choices on the three forms and a browser test
  for "Choose a category under the field", with `TestKnownNameKeepsWhatWasTyped` still green

## 3. Integration

- [x] 3.1 With `make check` green, in the browser at 390px, open the category field of an empty section's add form and of an edit page; verify the common categories are offered and nothing overflows

- [x] 3.2 With `make check` green, on an iPhone-sized Safari or Chrome at 390px, tap a choice under the category field
  of the add form, the edit page and the put-away page; verify the field is filled and nothing overflows

## Workflow follow-up

- Archive the change once merged.
