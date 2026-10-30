# Task 41: Category descriptions

## Goal
Display the selected category description from iiko under the menu heading.

## Depends on
40; citypark-go task 80.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/dish/api/to-dish.ts`, `src/widgets/food-ordering/ui/menu-catalog.tsx`.

## Files to change
Category type and adapter; menu catalog.

## Spec
Preserve category descriptions as plain text, hide empty descriptions and hide the selected category description during cross-category search. Preserve line breaks, use existing tokens, and let React escape text.

## Tests to write
No new logic tests for this small display change; run project checks and build.

## Done when
`npm run check` and `npm run build` pass.

## Out of scope
Editing descriptions on the public website.
