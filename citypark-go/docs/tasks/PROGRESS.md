# Progress

Take the first unchecked task. Tick it only when every "Done when" command passes. The numbers in brackets are the tasks it depends on.

## Foundation
- [x] 01 Repo skeleton and health endpoint
- [x] 02 Config [01]
- [x] 03 httpx: JSON responses and errors [01]
- [x] 04 Logging and middleware [03]
- [x] 05 Validation [03]
- [x] 06 Store: database connection and transactions [01]
- [x] 07 Migrations and schema [06]
- [x] 08 Test utilities [07]
- [x] 09 Clock, sale windows and money [01]

## Auth and users
- [x] 10 Passwords and JWT [09]
- [x] 11 Auth middleware [03, 10]
- [x] 12 User store [08]
- [x] 13 Sign-up and sign-in [05, 10, 12]
- [x] 14 Current user endpoints and admin bootstrap [11, 13]

## Simple CRUD
- [x] 15 Addresses [08, 11]
- [x] 16 Delivery prices [08, 09, 11]

## Catalog
- [x] 17 Catalog store [08, 09]
- [x] 18 Category endpoints [17]
- [x] 19 Product endpoints [18]

## Integration clients
- [x] 20 iiko client [01]
- [x] 21 iiko sync: categories and products [08, 09, 20]
- [x] 22 iiko sync: images, tags, endpoint [21]
- [x] 23 Telegram client and order message [09]
- [x] 24 Moneta signatures [09]

## Orders
- [x] 25 Order store [12, 15, 17]
- [x] 25b Order store: payment state and duplicate lookup [25]
- [x] 26 Order service: create [16, 25b]
- [x] 27 Create-order endpoint and working hours [26]
- [x] 28 Order read endpoints [27]
- [x] 29 Order admin endpoints [28]
- [x] 30 Moneta payment callback [27]

## Shipping
- [x] 31 Application wiring [02, 04, 14, 16, 18, 19, 22, 23, 24, 27, 29, 30]
- [x] 32 Seed command [31]
- [x] 33 Dockerfile, env example, README [31]
- [x] 34 OpenAPI document [31]
- [-] 35 Legacy data migration command [31] SKIPPED

## iiko API v2 and menu/v3
- [x] 36 iiko v2 auth and config [33]
- [x] 37 iiko menu/v3 client and menus table [36]
- [x] 38 iiko sync from menu/v3 [37]
- [x] 39 Hidden categories [38]
- [x] 40 Product descriptions from iiko [39]
  - Not checked against real iiko: no credentials in this session. After `POST /iiko/update-products`, run `curl localhost:3000/api/v1/products?pageSize=500` and count non-empty `description`s.

## Hotel
- [x] 41 Hotel free rooms webhook [31]

## Admin panel support (see ../docs/adr/0001-admin-panel.md)
- [x] 42 Order search filters [29]
- [x] 43 Menu sync for admins, one sync at a time [38]
- [x] 44 Scheduled menu sync [43]
- [x] 45 Image uploads [31]
- [-] 46 Dish photos from the admin REMOVED: dish photos come from iiko (`syncImages`)
- [x] 47 Rich text sanitizer
  - Relative links get `rel="noreferrer"` only and external ones `rel="noreferrer noopener" target="_blank"`: bluemonday adds `noopener` only together with `target=_blank`. Tests pin this output.
- [x] 48 Hotel rooms content [45, 47]
- [x] 49 News [45, 47]
- [x] 50 Content blocks [45, 47]
- [x] 51 Site settings [31]
- [x] 52 Hotel prices [41]
- [x] 53 Hotel reports from 1C [31]
  - Owner to do: with real `ONEC_*` values run `curl -s -H "Authorization: Bearer $ADMIN_TOKEN" 'localhost:3000/api/v1/reports/hotel/sales?from=2026-10-01&to=2026-10-21'` and paste the output here; if the real 1C field names differ from the task table, fix the structs in `internal/onec/client.go`.
- [x] 54 Content block rows for the home and hotel pages [50] (needs citypark-modern task 16)

## Content from the old site
- [x] 55 Hotel guest information and room texts from the old site [48, 54]
  - Owner: confirm the guest information is current (rules ПП № 1085 are from 2015, the category certificate in the old text expired in 2019).
  - Assertions in `internal/content/seed_test.go` and `internal/room/room_test.go` were updated for the filled texts (approved by the owner).
- [x] 56 Room summary [55] (needed by citypark-modern task 24 and citypark-admin task 18)
- [x] 57 Room photo gallery [56] (needed by citypark-admin task 19 and citypark-modern task 25)

## Payment
- [x] 58 Moneta payment URL [30] (needed by citypark-modern task 26)
  - Owner check: with the demo shop's real `MONETA_ID`/`MONETA_SECRET`, open `redirectUrl` from a `POST /orders` and see the payment form, not a signature error.

## Delivery
- [x] 59 Delivery prices from the old site, free delivery threshold [16, 26]
  - Follow-up in citypark-modern: read `price` and `freeFrom` from `GET /delivery-prices` instead of `freeFrom: 500` in `src/entities/dish/model/delivery.ts`.

## iiko orders
- [x] 60 iiko delivery client [37]
- [x] 61 Order delivery time [27] (needed by citypark-modern task 28)
- [x] 62 Paid online orders go to iiko [30, 60, 61]
  - Owner check: set the real `IIKO_*` values (including `IIKO_TERMINAL_GROUP_ID` and `IIKO_PAYMENT_TYPE_ID`), pay an order in Moneta's demo shop and see it in iikoFront. Its comment should show the address, the time and the fee, and `orders.iiko_order_id` should be filled.

## Working hours
- [x] 63 Working hours from the database [27, 61] (needed by citypark-admin task 21 and citypark-modern task 29)
  - Owner: remove `WORKING_HOURS_START` and `WORKING_HOURS_END` from local `.env` files (they are no longer read). The hours are now edited with `PUT /working-hours`.

## Live orders
- [x] 64 Live order events over WebSocket [29, 63] (needed by citypark-admin task 22)

## Online payment only
- [x] 65 Online payment only [64] (needed by citypark-admin task 23 and citypark-modern task 31)

## Customer accounts and Telegram Mini App
- [x] 66 Order ownership migration [65]
- [x] 67 Order contact and ownership safety [66]
- [x] [68 Customer identities and optional phones](68-customer-identities.md) [67]
- [x] [69 Opaque customer sessions](69-customer-sessions.md) [68]
- [x] [70 SMTP and single-use email tokens](70-customer-email-tokens.md) [69]
- [x] [71 Verified email registration and sign-in](71-verified-email-auth.md) [70]
- [x] [72 Customer password recovery](72-customer-password-reset.md) [71]
- [x] [73 Google authorization start](73-google-authorization.md) [69, 71 (shared rate-limit helper)]
- [x] [74 Google completion and explicit linking](74-google-completion.md) [73]
- [x] [75 Telegram Mini App authentication](75-telegram-authentication.md) [69, 71 (shared rate-limit helper)]
- [x] [76 Transactional customer account merge](76-customer-account-merge.md) [68, 69, 75]
- [x] [77 Two-sided Telegram account pairing](77-telegram-account-pairing.md) [76]
- [x] [78 Linked methods and linking integration](78-customer-linked-methods.md) [74, 77]

Release sequence and future numbered tasks: [CUSTOMER_ACCOUNTS.md](CUSTOMER_ACCOUNTS.md).

## Automatic paid orders
- [x] [79 Automatic paid orders and one staff Telegram chat](79-automatic-paid-orders.md) [62, 64, 65, 78] (needed by citypark-admin task 24)

- [x] [80 Category descriptions and ordering times](80-category-settings.md) [79]
  - Full Go suite, build, vet and formatting pass. Coverage: catalog 88.8%, iiko 93.7%, app 99.4%, apidocs 100%.

## Dish ordering blocks
- [x] [81 Persistent dish ordering blocks](81-dish-ordering-blocks.md) [80] (needed by citypark-admin task 26 and citypark-modern task 42)
  - Add `is_order_blocked` to products (default false); preserve it through iiko syncs. Blocks apply to the whole dish until manually removed.
  - Expose `isOrderBlocked`; compute `isAvailableForOrder` from the manual block and existing product/category schedules. Unblocking must not override schedules; blocked dishes remain in public menu responses.
  - Add admin-only `PUT /products/{id}/ordering-block` with required boolean `isOrderBlocked`, returning the updated product; follow existing validation and error conventions.
  - Reject blocked dishes during order creation, including duplicate-order retries before returning a payment link, with a Russian error identifying the dish. Existing placed orders and payments remain unchanged.
  - Update OpenAPI and `docs/API.md`. Test block/unblock, sync persistence, schedule interaction, permissions, invalid requests, missing products, direct order rejection and duplicate retries; run required backend checks against the existing database.
  - Build, vet, formatting and serial full suite pass; catalog 88.3%, order 90.9%, iiko 93.7%, apidocs 100%.

## MONETA Assistant correction
- [x] [82 Correct MONETA Assistant integration](82-moneta-assistant.md) [81]
  - Implementation and protocol tests added; validation blocked: TEST_DB_DSN from .env.development refuses connections at host.docker.internal:5433 (2026-10-31). Per AGENTS.md, stopped without starting a database, ticking this task or committing. Moneta and config tests pass; database-dependent tests, coverage and real demo payment remain unverified.
  - Added requested Python Pay URL simulator with order-ID CLI, signed amount lookup, explicit amount override and dry run; six isolated Python tests pass. No real payment callback was sent. Existing backend database-validation block remains.

## iiko order diagnostics
- [x] [83 iiko order error diagnostics](83-iiko-order-error-diagnostics.md) [62, 79]
  - Implemented exception-message fallback and structured failed-attempt diagnostics; pickup/courier mapping and retry behavior are unchanged. Build, vet, formatting, package tests and coverage pass: iiko 94.1%, app 98.8%.
  - Completion blocked (2026-10-31): the serial full suite fails at the unchanged `internal/order/check_test.go:TestPayGET` (expects 200 SUCCESS and a paid order, receives 400 FAIL with missing operation ID/amount). Reproduces in isolation; the order package does not depend on the changed iiko/app packages. Left task 82 untouched, task 83 unchecked, and changes uncommitted until the required full suite passes. No real iiko request or replay of order 16 was made.

## MONETA callback test repair

- [x] [83b Repair MONETA GET callback regression test](83b-moneta-pay-get-regression.md) [82] (unblocks task 85 validation)
  - Corrected the GET success fixture and added missing/mismatched amount rejection tests. Targeted tests, uncached serial full suite, build, vet and formatting pass; order coverage 91.6%. Task 85 remains separate.

## Promo codes

Implementation plan: [Promo codes](PROMOCODES.md). Planned only; existing blockers remain unchanged.

- [x] [84 Promo RMS contract](84-promo-iiko-contract.md) [83]
  - Plan revised (2026-10-31): use one configured RMS flexible-sum discount with saved eligible positions. Owner accepts different iiko per-line rounding; total discount and food payment must match exactly. The earlier exact-line-mapping blocker is superseded. Task 84 remains unchecked pending its revised documentation/fixture checks; restaurant setup and authorized total reconciliation remain release requirements. See [IIKO_PROMOCODES.md](IIKO_PROMOCODES.md).
- [x] [85 Promo campaign persistence](85-promo-campaign-store.md) [84]
  - Added campaign schema/store, normalized immutable codes, validated targeting, transactional replacements and shared pricing locks. Shared database cleanup now resets campaigns.
  - Previous TestPayGET validation blocker resolved by task 83b. Formatting, build, vet, full serial suite and fresh database/coverage tests pass (2026-10-31): promo 94.0%, testutil 83.1%.
- [x] [86 Promo campaign admin API](86-promo-campaign-api.md) [85]
  - Added admin-only list/get/create/update with transactional writes, exact validation/conflict messages and app wiring; API.md/OpenAPI include all four routes and captured examples. Auth, CRUD, disable, invalid requests and database-error tests pass. Formatting/build/vet/full serial suite pass; coverage: promo 95.9%, app 98.8%, apidocs 100%.
- [ ] [87 Promo pricing and quotes](87-promo-pricing-quotes.md) [85]
- [ ] [88 Promo order snapshots and payments](88-promo-order-snapshots.md) [87]
- [ ] [89 Promo RMS and staff integration](89-promo-order-integrations.md) [84, 88]
