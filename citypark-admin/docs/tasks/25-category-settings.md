# Task 25: Category descriptions and ordering schedule

## Goal
Show iiko category descriptions and edit ordering hours and weekdays in the menu.

## Depends on
24; citypark-go task 80.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/features/menu/ui/menu-page.tsx`, `src/features/working-hours/ui/working-hours-card.tsx`.

## Files to create
`src/features/menu/ui/category-settings.tsx`, `src/features/menu/model/category-hours.ts`, its test.

## Files to change
Menu API/page, generated schema, `e2e/menu.spec.ts`.

## Spec
Selected categories show their description, start/end text fields and seven weekday checkboxes. Empty times remove time limits. Save through the authenticated category ordering-hours endpoint. Preserve input on errors, show field errors and toast, invalidate menu data after saving. Category changes reset the form.

## Tests to write
Weekday conversion and selected category editor in the menu flow.

## Done when
Conventions check, build and menu e2e pass against the backend with task 80 deployed.

## Out of scope
Editing iiko descriptions from the admin.
