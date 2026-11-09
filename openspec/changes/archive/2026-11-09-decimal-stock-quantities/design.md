# Design

## Context

See `proposal.md` for the motivation and `specs/household-stock/spec.md` for the required behaviour. The current state
that shapes the approach:

- `stock_item.quantity` is `INTEGER NOT NULL CHECK (quantity >= 1)` (migration `00006_stock.sql`), and
  `stock.Values.Quantity` is an `int`. The form parses it with `strconv.Atoi`; the inputs are `type="number"
  step="1" min="1"`.
- Increase, decrease, merging on add and opening are single SQL statements relative to the stored quantity, so that
  concurrent changes all count.
- SQLite can't drop a `CHECK` constraint or change a column's meaning in place: the table has to be rebuilt.
- Before migrating a database that already has a schema, the application saves a copy of it in the backup directory.

## Goals / Non-Goals

**Goals:**

- Exact arithmetic on one-decimal quantities: no floating point anywhere in storage or computation.

**Non-Goals:**

- Keeping the previous release working on a migrated database (see the exception below).

## Decisions

### Quantities as integer tenths

`stock.Quantity` is an `int` counting tenths, with a constructor that parses user text and a `String` method:

- Parsing trims the text, accepts a dot or a comma as the separator, at most one digit after it, and refuses anything
  that isn't above 0 (`ErrQuantity`). "1,5" → 15, "2" → 20, "2.0" → 20; "1.25", "0", "-1", "abc" are refused.
- `String` prints whole quantities without decimals ("2") and others with one ("1.5"), always with a dot. Templates
  print `{{.Quantity}}` as today, and the dot form is what the edit form and the undo strip's hidden field post back,
  so it parses again unchanged.
- − and + are ±10 tenths; "1 or less" is `<= 10`.

*Alternative considered*: `float64` (or a `REAL` column). Sums such as 0.1 + 0.2 don't come out exact, and the
comparisons with 1 that decide removal and opening would depend on rounding.

### Rebuilding the table, as an exception to the additive rule

Migration `00007` rebuilds `stock_item` the usual SQLite way: create the new table, copy every row with
`quantity_tenths = quantity * 10`, drop the old one, rename the new one, recreate the `(section, name_key)` index. The
new table is the old one with `quantity` replaced by `quantity_tenths INTEGER NOT NULL CHECK (quantity_tenths > 0)`;
the new name makes the unit visible in every query. Nothing references `stock_item`, so dropping it needs no change
to foreign key enforcement. The down migration rebuilds it back, rounding up to whole units.

This breaks the README's rule that migrations only add: the previous release can't run on a migrated database (its
queries name `quantity`). The rule exists to make going back possible without touching the data; here the alternative
was keeping two quantity columns in step with a rule for which one to trust, for a step that is not expected to be
needed. Going back to the previous release after this one means restoring the copy saved before the migration, losing
the changes made since. The release notes say so.

*Alternative considered*: a new `quantity_tenths` column next to the old one, both written, and the old one trusted
when the previous release changed it; it keeps the rule but puts a consistency check in every quantity query.

### A text field with a decimal keyboard

The quantity inputs become `type="text" inputmode="decimal"`, keeping `required`. A `type="number"` field drops a value
typed with the separator the browser's locale doesn't expect, and some phones offer only a comma on a decimal keypad:
the field must pass the text through and let the server parse it. The server's message for a refused quantity becomes
"The quantity must be above 0, with at most one decimal."

## Risks / Trade-offs

- [Going back to the previous release needs the pre-migration backup] → Stated in the release notes; the backup is
  taken automatically.
- [Tests write quantities in tenths, where `Quantity: 3` now means 0.3] → Every existing test literal is converted
  in the same task as the type change, and a whole-unit constant keeps them readable.

## Migration Plan

The migration runs on start as usual, after the automatic copy. Rolling back with `scripts/promote.sh <previous>` alone
is not enough, since the previous release would fail on the migrated database: stop the container, put the copy saved
before the migration in place of the database, then promote the previous release. Release notes are generated from
pull request titles, so the warning is added to this release's notes by hand when it is cut.
