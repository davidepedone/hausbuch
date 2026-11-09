# Design

## Context

See `proposal.md` for the motivation and the delta specs for the required behaviour. The current state that shapes
the approach:

- `stock.Sections` lists the sections; routes, the home page tiles, `TestEverySectionHasAnIcon` and
  `Store.Warnings` already work from it or from the stored section, so a new entry is picked up without further
  wiring.
- `Section.Label` returns "Medicines" for Medicines and "Pantry" for anything else; `DefaultLead` and `leadTimes` in
  `cmd/hausbuch` name each section explicitly.
- The home page shows the section tiles in a two-column grid, icon and label side by side.

## Goals / Non-Goals

**Goals:**

- Three tiles that fit a 390px screen without overflowing sideways or leaving a hole in the grid.

**Non-Goals:**

- Turning sections into data (a table or a configuration): three fixed sections don't need it.

## Decisions

### A constant `Supplies = "supplies"`, after Pantry

The value is the URL path (`/supplies`) and what rows store in `stock_item.section`. `Label` becomes a switch over the
three constants, so that a fourth section can't silently be called "Pantry".

### Lead time in `DefaultLead` and `HAUSBUCH_SUPPLIES_LEAD_DAYS`

7 days by default, read and validated like the other two.

### Three tiles in one row

The tiles' grid goes from two to three columns at every width, and each tile stacks its icon above the label, like
the page type tiles below them. At 390px each tile is about 110px wide, enough for "Medicines" on one line.

*Alternative considered*: keeping two columns, which leaves Supplies alone on a second row next to an empty cell.

### Icon: a spray bottle

Drawn with a few strokes in the 24×24 box like the capsule and the jar, so that it reads at the tiles' size.

## Risks / Trade-offs

- [Supplies already recorded in the Pantry stay there] → Users move them by removing and adding them again; the
  Pantry's suggestions keep remembering their names, which is harmless.
