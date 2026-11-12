# Design

## Context

The expiry field is one shared template part, `expiryField`, used by the add form (`section.html`), the edit form
(`item.html`) and the put-away form (`putaway.html`); all three post to handlers that read it through `readForm` and
`itemForm.values`. A known product's values come from `stock_product` (no unit column today): the name field asks
`GET /{section}/product` through htmx, which answers with the `productFields` part, and `productfields.js` copies its
values into the fields the user hasn't typed in. After a successful add, `addItem` redirects to `/{section}?added=<name>&location=<location>`. The section page then opens the "Add item"
details, puts `autofocus` on the name, says "Added <name>." in the same passing notice as "Added <name> to the
shopping list.", and fills the location with the one passed. When that location isn't empty the form is marked
`data-kept="location"`, and `productfields.js` counts the field it names as typed, so a known product doesn't replace
it; an empty kept location is not marked, so a known product fills it. Reloading the page shows the message again,
which is harmless; a redirect keeps a reload from adding the item twice.

Alternatives: keeping the last location in the session would survive leaving the page, which is not wanted; answering
the POST with the page itself makes a reload post the form again.

## Risks / Trade-offs

- [iOS doesn't open the keyboard for `autofocus` on page load] → the cursor is in the name field and one tap opens
  the keyboard; the form is open and empty, which is most of the gain. Checked on a real iPhone during the test task.
- [The year list ends 10 years ahead] → medicines and tins rarely go further; a further date can still be entered with
  "Exact day".
- [`added` and `location` in the address] → they only prefill the form, and the address stays inside the signed-in
  pages.

## Migration Plan

The migration only adds a column with a default; rolling back drops it. No data changes.
