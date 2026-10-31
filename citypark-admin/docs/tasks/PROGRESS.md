# Progress

Take the first unchecked task. Tick it only when every "Done when" command passes. The numbers in brackets are the
tasks it depends on. A task that needs a citypark-go task says so; check `../citypark-go/docs/tasks/PROGRESS.md` and
stop if that task is not ticked.

## Foundation
- [x] 01 Project scaffold
- [x] 02 Typed API client and token storage [01]
- [x] 03 Sign-in, access guard and app shell [02]

## Operations
- [x] 04 Delivery prices [03]
- [x] 05 Order history [03] (needs citypark-go task 42)
- [x] 06 Free rooms (read only) [03]
- [x] 07 Menu (read only) and the sync button [03] (needs citypark-go task 43)

## Shared editors
- [x] 08 Image field (upload, alt text, focal point) [02] (needs citypark-go task 45)
- [-] 09 Dish photos REMOVED: dish photos come from iiko
- [x] 10 Rich text editor [01]

## Content
- [x] 11 Rooms [08, 10] (needs citypark-go task 48)
- [x] 12 News [11] (needs citypark-go task 49)
- [x] 13 Page content (home and hotel) [08, 10] (needs citypark-go tasks 50, 54)
- [x] 14 Site settings (contacts) [11] (needs citypark-go task 51)
- [x] 15 Hotel prices [14] (needs citypark-go task 52)

## Reports and shipping
- [x] 16 Hotel reports [03] (needs citypark-go task 53)
- [x] 17 Docker image and deployment notes [16]
- [x] 18 Room summary field [11] (needs citypark-go task 56)
- [x] 19 Room photo gallery [11, 18] (needs citypark-go task 57)

## Delivery
- [x] 20 Free delivery threshold [04] (needs citypark-go task 59)
- [x] 21 Working hours on «Доставка» [20] (needs citypark-go task 63)

## Live orders
- [x] 22 Live orders [05] (needs citypark-go task 64)

## Online payment only
- [x] 23 Online payment only [22] (needs citypark-go task 65)

## Automatic paid orders
- [x] [24 Remove manual order confirmation](24-remove-order-confirmation.md) [23] (needs citypark-go task 79)

- [x] [25 Category descriptions and ordering schedule](25-category-settings.md) [24; backend80]
  - `npm run check`, `npm run build`, and `npm run e2e -- e2e/menu.spec.ts` pass against the updated backend. The live flow restores the original category schedule.

## Dish ordering blocks
- [x] [26 Admin dish ordering switch](26-dish-ordering-switch.md) [25] (needs citypark-go task 81)
  - Regenerate API types and add a labelled “Заблокировать заказ” switch to each dish row using `PUT /products/{id}/ordering-block`.
  - Show the persistent manual block separately from schedule-based unavailability. Disable the switch while saving, report success/failure and refresh menu data after success.
  - Test block/unblock, failed saves and state refresh on mobile and desktop; run required admin checks and the menu flow against the updated backend.
  - API types regenerated; check (274 unit tests), build and menu e2e pass, including save failure and restored block/unblock fixtures. Optional QA_PUBLIC_BASE_URL runs the public blocked-dish flow while the fixture is blocked.

## Promo codes

Implementation plan: [Promo codes](../../../citypark-go/docs/tasks/PROMOCODES.md). Planned only; existing blockers remain unchanged.

- [x] [27 Promo campaign management](27-promo-campaigns.md) [admin26; backend86]
  - Added campaign creation/editing/disabling, searchable independent targets and explicit Moscow schedules with preserved form errors. API generation, check (294 unit tests), build and four real-backend Playwright flows pass at 390/1440 px in both themes with a non-Moscow browser timezone. All test campaigns are disabled.
- [x] [28 Promo order breakdown](28-promo-order-details.md) [admin27; backend89]
  - Previous backend 503 blocker resolved after owner configuration (2026-10-31). Added stored subtotal/code/discount/food total and line discounts without changing no-code presentation or subtracting discounts twice. API generation, check (294 unit tests), build and four real-backend promo-order flows pass at 390/1440 px in both themes, covering capped discounts, pickup/delivery, immutable snapshots after campaign edits, legacy display and live totals. Test campaigns are disabled; test orders remain unpaid. No payment callback, real charge or iiko order was sent.
