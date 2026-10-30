# Task 24: Remove manual order confirmation from the admin

## Goal
The admin reflects automatic fulfillment after successful payment and no longer presents manual confirmation as an operational step. Historical confirmed orders remain readable in order history.

## Depends on
23 and citypark-go task 79. Stop if backend task 79 is not ticked.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/features/live-orders/model/board.ts`
- `src/features/live-orders/model/socket.ts`
- `src/features/live-orders/ui/live-orders-page.tsx`
- `../citypark-go/docs/API.md` (orders and live events)

## Files to create
None required.

## Files to change
- `src/shared/api/schema.d.ts`: regenerate with `npm run api:gen`, never edit manually.
- `src/features/live-orders/model/board.ts`, `board.test.ts`: remove the confirmed bucket.
- `src/features/live-orders/model/socket.ts`, `socket.test.ts`: remove confirmation event handling.
- `src/features/live-orders/ui/live-orders-page.tsx`: remove the confirmed column and adjust the responsive layout for two columns.
- `src/features/live-orders/model/alerts.test.ts`: update affected fixtures if needed.
- `e2e/live-orders.spec.ts`, `e2e/orders.spec.ts`: regression coverage.
- Any remaining confirmation action callers found in `src/`: remove the action and its mutation; do not replace it with another approval step.

## Spec
- Regenerate types from the backend contract after task 79. The confirmation endpoint and `order.confirmed` event must be absent.
- Live board has two columns: existing «Ждут оплаты» for pending orders and «Новые» for new/paid orders. Preserve current ordering, pending/paid updates, alerts, reconnect, and order detail behavior.
- Remove «Подтверждены» and its board bucket. Historical confirmed orders are excluded from the live board and remain available in order history, its status filter, and details.
- Keep historical `confirmed` labels/types where needed for those reads. Do not relabel them as paid or assume their payment succeeded.
- Remove confirmation buttons, mutations, and event handling if present. No replacement manual acceptance step.
- Keep UI text Russian, keyboard access, and layouts at 390 px and desktop in both themes.
- Do not change payment flows or claim that a paid order has necessarily been accepted by iiko.

## Tests to write
- Board unit tests: pending goes to pending; new/paid goes to fresh; historical confirmed and canceled orders are excluded; existing ordering remains correct.
- Socket unit tests: hello/pending/paid remain supported; the removed confirmation event is ignored like other unsupported events.
- Update affected alert fixtures; paid alerts still work without confirmation.
- Live-orders e2e: two columns, no «Подтверждены» column or confirmation action; pending order appears without reload and details open normally. Preserve existing paid-update coverage where available.
- Order-history regression: historical confirmed status remains a valid filter and renders with its existing label; cover with existing fixtures/unit tests without adding a production write endpoint for historical statuses.

## API documentation
No backend edits. Use the contract produced by citypark-go task 79.

## Done when
```bash
npm run api:gen
npm run check
npm run build
npm run e2e -- e2e/orders.spec.ts e2e/live-orders.spec.ts
```
Use VITE_API_URL from the environment. Do not start the backend/database; ask the owner if unavailable. Tick this task only after checks pass and commit `task 24: remove manual order confirmation`.

## Out of scope
Backend/configuration changes, changing historical statuses, public website changes, and new order lifecycle states. Historical task files remain unchanged.
