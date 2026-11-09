# Spec Delta

## REMOVED Requirements

### Requirement: Page language
**Reason**: With the skeletons in English only, a page's language served only the search, which now matches English
and Italian word forms in every page.
**Migration**: Nothing to do for users; `HAUSBUCH_DEFAULT_LANGUAGE` can be removed from an installation's
configuration, and is ignored if left. The stored language stays in the database, unused, for the previous release.
