# Task 23: Online payment only

## Goal
The admin no longer knows about cash. Orders have no payment type: the «Оплата» chip, the «Оплата» filter on
«Заказы» and the `order.created` live event are gone. Everything else works as before.

## Depends on
22, and citypark-go task 65.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/lib/order-labels.ts`, `src/features/orders/model/search.ts`,
`../citypark-go/docs/API.md` (sections `POST /orders`, `GET /orders`, `GET /orders/live (WebSocket)`).

## Background
Decided by the owner (2026-10-28): orders are paid only online through MONETA.RU. The backend (citypark-go task 65)
removed `paymentType` from orders and from `POST /orders`, the `paymentType` filter of `GET /orders` and the
`order.created` event. A new order is now always `pending` («Ожидает оплаты») until Moneta confirms the payment.

## Files to change
- `src/shared/api/schema.d.ts`: `npm run api:gen`.
- `src/shared/lib/order-labels.ts`: remove `PAYMENT_TYPES`, `PaymentType`, `paymentLabels`, `isPaymentType`,
  `paymentLabel`. `src/shared/lib/order-labels.test.ts`: remove the payment test and the `paymentLabel('card')` line.
- `src/shared/ui/order-chips.tsx`: remove `PaymentChip`. Remove its use in `src/features/orders/ui/orders-table.tsx`
  (desktop row and phone card, if both show it), `src/features/orders/ui/order-details.tsx` and
  `src/features/live-orders/ui/live-order-card.tsx`. If the table then has an empty «Оплата» column, remove the column
  header too.
- `src/features/orders/ui/order-filters.tsx`: remove the «Оплата» `FilterSelect` and its imports.
- `src/features/orders/model/search.ts` (+ its test, if it covers `paymentType`): remove `paymentType` from
  `OrderSearch`, from parsing, from the list of keys and from `hasFilters`. An old link with `?paymentType=cash` simply
  ignores the parameter (it is not parsed any more).
- `src/features/orders/api.ts`: if it forwards `paymentType` to `GET /orders`, remove it.
- `src/features/live-orders/model/socket.ts`: `EVENT_TYPES` without `'order.created'`.
- `src/features/live-orders/model/board.test.ts`, `alerts.test.ts`: use `order.pending` where they used
  `order.created`; fixtures without `paymentType`. `board.ts` and `alerts.ts` need no change (check with `grep`).
- `e2e/orders.spec.ts`, `e2e/live-orders.spec.ts`: Spec.

## Spec
No new UI. After `npm run api:gen`, `npm run check` must show no type errors; fix every place that read
`order.paymentType` by removing it, not by casting.

### e2e
- `e2e/orders.spec.ts`: create the order without `paymentType`. The row shows «Ожидает оплаты» instead of «Новый»
  and no «Наличными». The link test opens `/orders?status=pending&q=<customer>`.
- `e2e/live-orders.spec.ts`: drop the cash order. Create one order without `paymentType` (skip on 503 as now): it
  appears in the «Ждут оплаты» column without a reload, it is not in «Новые», there is no «Оплачен заказ» toast;
  open it from that column and check the drawer shows `Заказ № <id>` and the customer's name. The page still shows
  «Онлайн» (the connection status).

## Tests to write
Only the changes above: the labels test, `board.test.ts`, `alerts.test.ts`, `search.test.ts` (if present), and the
two e2e specs.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/orders.spec.ts e2e/live-orders.spec.ts`, and
`grep -rniE "cash|Наличн|paymentType|PaymentChip|order\.created" src e2e --exclude=schema.d.ts` finds nothing.

## Out of scope
- Renaming the «Новые» column or changing which statuses it holds.
- Old task files in `docs/tasks/`: they are history.
