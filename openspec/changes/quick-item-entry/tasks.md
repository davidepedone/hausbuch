# Tasks

## 1. Expiry as a month and a year

- [x] 1.1 Read the expiry in `itemForm`: `expires_on` when present, otherwise `expires_month` and `expires_year` as
  the month's last day, none when both are empty, refused with "The expiry date isn't complete." when only one is
  given; verify web tests for "Month and year", "February", "Exact day", "Month without a year" and "No expiry",
  through the add form, and one through the put-away form
- [x] 1.2 Rewrite `expiryField` as the month and year selects (years from 2 back to 10 ahead, plus the shown value's)
  and the date input, the server choosing which shows and is enabled; add `expiry.js` for the "Exact day" and "Month
  and year" switch, loaded by the three forms; verify web tests for "Editing a month's last day", "Editing another
  day" and a refused form keeping the exact day, and a Chrome test that "Exact day" posts the date and not the selects
- [x] 1.3 Check the field in a browser at phone and desktop width, in the add, edit and put-away forms

## 2. The unit of a known product

- [x] 2.1 Add the migration with `stock_product.unit`, store it in `remember` and return it from `Product`; verify a
  store test that a removed row's unit comes back, and the migration's Down with a throwaway test
- [x] 2.2 Mark the unit field `data-product-field`, add the unit to the product answer, and make `productfields.js`
  fill and track it; verify the web test for "Buy
  something again" and the Chrome test `TestKnownNameKeepsWhatWasTyped` extended to the unit

## 3. Adding several items in a row

- [x] 3.1 Redirect a successful add to the section with the added name and location, and render the "Add item" form
  open, with `autofocus` on the name, the "Added <name>." message, and the form marked `data-kept="location"` when the
  kept location isn't empty; verify web tests for "Next item" and "Location kept"
- [x] 3.2 Make `productfields.js` treat the field a `data-kept` form names as typed; verify Chrome tests for "Known product after a
  kept location" and "Known product with an empty location"

## 4. Integration

- [x] 4.1 Run `make check` green
- [ ] 4.2 On a real iPhone, load a few items in a row with month-year expiries and one exact day, and note in the PR
  whether the keyboard opens on its own after "Add"

## Workflow follow-up

- Archive the change after the PR is merged.
