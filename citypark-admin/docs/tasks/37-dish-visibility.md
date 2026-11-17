# Task 37: Show iiko dish visibility to staff

## Goal
Keep hidden dishes manageable and show their iiko visibility in the menu list.

## Depends on
Backend 108; explicit owner-requested fix.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/features/menu/ui/dish-row.tsx`, `src/features/menu/model/group.ts`, `../citypark-go/docs/API.md`.

## Files to create
- `e2e/dish-visibility.spec.ts`

## Files to change
- `src/shared/api/schema.d.ts`: regenerate with `npm run api:gen`.
- `src/features/menu/ui/dish-row.tsx`: read-only visibility Chip.
- `src/features/menu/model/group.test.ts`: required-field fixture and staff retention regression.

## Spec
Products expose required boolean `isHidden`. Keep hidden products in their categories and staff search.
Show `Скрыто в iiko` for hidden dishes. Do not show a schedule-only warning caused solely by hiding;
an independent manual block still shows its own warning and remains editable. Do not add a visibility toggle.
Descriptions and saved homepage configurations remain editable as before.

## Tests to write
Unit: hidden dishes remain searchable in their group.
Browser: an isolated intercepted catalog proves hidden and visible dish labels and retained staff controls,
including simultaneous hidden/manual-block status. This does not replace real-backend menu acceptance.

## API documentation
Regenerate from backend 108's Product schema. No endpoint changes in this repository.

## Done when
`npm run api:gen`, `npm run check`, `npm run build`, `npm run e2e -- e2e/dish-visibility.spec.ts` pass.
Run read-only real-backend menu acceptance when the running backend includes task 108; report old-build limitations.

## Out of scope
Backend/frontend implementation, iiko writes/syncs, orders/payments, production deployment.
