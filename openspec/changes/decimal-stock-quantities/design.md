# Design

## Context

See `proposal.md` for the motivation and `specs/household-stock/spec.md` for the required behaviour. The current state
that shapes the approach:

- `stock_item.quantity` is `INTEGER NOT NULL CHECK (quantity >= 1)` (migration `00006_stock.sql`), and
  `stock.Values.Quantity` is an `int`. The form parses it with `strconv.Atoi`; the inputs are `type="number"
  step="1" min="1"`.
- Increase, decrease, merging on add and opening are single SQL statements relative to the stored quantity, so that
  concurrent changes all count.
- Migrations may only add (README, development section): no change of what a column means, no constraint existing
  rows or code could break, so that the previous release keeps working on a database the new one migrated. The
  `CHECK (quantity >= 1)` rules out storing 0.5 in the current column, and storing tenths in it would change its
  meaning.

## Goals / Non-Goals

**Goals:**

- Exact arithmetic on one-decimal quantities: no floating point anywhere in storage or computation.
- Going back to the previous release keeps working, and coming forward again doesn't trust a value the previous
  release left stale.

**Non-Goals:**

- Removing the integer `quantity` column: it can only go in a later release, once no release in use reads it.

## Decisions

### Quantities as integer tenths

`stock.Quantity` is an `int` counting tenths, with a constructor that parses user text and a `String` method:

- Parsing trims the text, accepts a dot or a comma as the separator, at most one digit after it, and refuses anything
  that isn't above 0 (`ErrQuantity`). "1,5" → 15, "2" → 20, "2.0" → 20; "1.25", "0", "-1", "abc" are refused.
- `String` prints whole quantities without decimals ("2") and others with one ("1.5"), always with a dot. Templates
  print `{{.Quantity}}` as today, and the dot form is what the edit form and the undo strip's hidden field post back,
  so it parses again unchanged.
- − and + are ±10 tenths; "1 or less" is `<= 10`.

*Alternative considered*: `float64` (or a `REAL` column). Sums of tenths such as 0.1 + 0.2 don't come out exact, and
the comparisons with 1 that decide removal and opening would depend on rounding.

### A new column, with the old one kept in step

Migration `00007` adds `quantity_tenths INTEGER` (nullable) to `stock_item` and fills it with `quantity * 10`. From
then on every write sets both columns: `quantity_tenths` to the exact value and `quantity` to the tenths rounded up,
at least 1 (0.5 → 1, 2.5 → 3), so the previous release still reads a valid whole quantity.

The previous release, if gone back to, changes only `quantity` and inserts rows with `quantity_tenths` NULL. So the
current quantity is read as `quantity_tenths` only when it is set and `quantity` still equals its rounded-up value;
otherwise as `quantity * 10`. This rule is one SQL expression in a Go constant, used both in the `SELECT` and in the
relative updates (`SET quantity_tenths = <current> + 10, quantity = <rounded up of current + 10>`), so increase,
decrease, merge and opening stay single statements as today.

*Alternatives considered*: changing the meaning of `quantity` or relaxing its `CHECK` (needs rebuilding the table, and
breaks the previous release); a generated column for the current value (the same expression, one more schema object
to keep in mind).

### A text field with a decimal keyboard

The quantity inputs become `type="text" inputmode="decimal"`, keeping `required`. A `type="number"` field drops a value
typed with the separator the browser's locale doesn't expect, and some phones offer only a comma on a decimal keypad:
the field must pass the text through and let the server parse it. The server's message for a refused quantity becomes
"The quantity must be above 0, with at most one decimal."

## Risks / Trade-offs

- [The previous release shows 0.5 as 1 and 2.5 as 3] → Only after going back, which is a recovery step; its own
  changes are then picked up correctly when coming forward again.
- [A previous-release change that lands exactly on the rounded-up value, e.g. editing 2.5 (shown as 3) and saving it
  as 3] → The new release keeps reading 2.5: the user saved what they were shown, so the difference is under one
  unit and only after going back.
- [Tests write quantities in tenths, where `Quantity: 3` now means 0.3] → Every existing test literal is converted
  in the same task as the type change, and a whole-unit constant keeps them readable.

## Migration Plan

The migration only adds a nullable column and fills it, so it applies on start as usual. Going back to the previous
release needs nothing: it ignores the new column.
