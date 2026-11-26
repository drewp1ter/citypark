# Task 38: Mark dishes and categories on the iiko stop list

## Goal

Show staff which dishes are on the iiko delivery stop list and how many listed dishes each menu category contains.

## Depends on

Backend task 111.

## Read first

`docs/tasks/CONVENTIONS.md`, `src/features/menu/ui/dish-row.tsx`, `src/features/menu/ui/category-list.tsx`, `../citypark-go/docs/API.md`.

## Changes

- Regenerate the admin API types from the backend OpenAPI schema.
- Show `В стоп-листе iiko` for a dish with `isOnStopList`. The iiko visibility label and manual ordering warning can appear alongside it.
- Show a circular stop-list count on the right for categories with listed dishes, including the synthetic `Скрытые в iiko` group. Include `В стоп-листе: N` in the accessible name. Count all dishes in the group before applying search. The existing menu sync invalidation reloads the status.

## Tests

Extend `e2e/dish-visibility.spec.ts` to verify dish state, counts, search, and refresh after the sync action at phone and desktop widths in both themes.

## Done when

Run `npm run api:gen`, `npm run check`, `npm run build`, and `npm run e2e -- e2e/dish-visibility.spec.ts`. Verify the menu against a running backend that includes task 111.
