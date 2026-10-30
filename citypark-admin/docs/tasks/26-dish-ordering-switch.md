# Task 26: Admin dish ordering switch

## Goal
Allow admins to persistently block/unblock ordering for individual dishes from the menu.

## Depends on
25; citypark-go task 81 must be implemented and deployed for live checks.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/features/menu/api.ts`, `src/features/menu/ui/dish-row.tsx`, `e2e/menu.spec.ts`.

## Files to create
None required.

## Files to change
- `src/features/menu/api.ts`: authenticated mutation and menu invalidation.
- `src/features/menu/ui/dish-row.tsx`: labelled switch and status handling.
- `src/shared/api/schema.d.ts`: regenerate with `npm run api:gen`.
- `e2e/menu.spec.ts`: block/unblock and failure coverage.

## Spec
Read HeroUI v3 docs before using the switch. Add “Заблокировать заказ” to each row, checked from `isOrderBlocked`, independent of schedule availability. PUT `/products/{id}/ordering-block` with the explicit desired boolean. Disable the control during saving; do not leave an unsuccessful change visually applied. On success show `Настройки заказа блюда сохранены` and invalidate menu queries. On failure show backend field errors and message using existing patterns.

Distinguish a manual block with `Заказ заблокирован`; retain `Не в продаже сейчас` for schedule unavailability when not manually blocked. Unblocking does not override schedules. Support keyboard use, 390 px and desktop, light/dark themes. No bulk action or confirmation dialog.

## Tests to write
Extend the menu flow: block a dish, verify persisted state after reload and sync, unblock and verify restoration; restore original state after live checks. Verify pending controls, failed save without false success, and schedule-unavailable dishes remaining unavailable after unblocking. Check accessible labels and mobile layout.

## Done when
`npm run api:gen`, `npm run check`, `npm run build`, and `npm run e2e -- e2e/menu.spec.ts` pass. Use configured existing backend; do not start services. Update PROGRESS and commit this task separately.

## Out of scope
Backend implementation and public website changes.
