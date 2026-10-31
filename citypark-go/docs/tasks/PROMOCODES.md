# Promo codes implementation plan

Status: planned, not implemented. Owner-approved behavior recorded 2026-10-31.
This is a documentation handoff: implement one numbered task per session, in dependency order.
Do not tick tasks while preparing these documents. Existing backend task 83 must be completed first.

## Product rules

- Campaigns apply to delivery and pickup in the website and Telegram Mini App, including guests.
- Staff create, edit and disable codes. One code per order; a replacement replaces the previous code.
- Unlimited reuse; no customer eligibility restrictions, usage counters, or redemption limits.
- Integer percentage 1–99; optional minimum eligible food spend and optional maximum discount in rubles.
- Target all dishes, or selected product IDs and category IDs together (union, never double discount).
- Categories follow current membership. Added dishes qualify automatically; moved-out dishes cease qualifying
  unless explicitly selected or covered by another selected category.
- Minimum spend uses eligible food before discount. Delivery is never discounted.
- Free-delivery thresholds use all food AFTER discount; applying a code can increase the delivery fee.
- Optional start/end timestamps. Defaults: start immediately, no expiry. Start inclusive, end exclusive.
  Admin dates use Europe/Moscow; API timestamps use RFC 3339 with offsets, stored as timestamptz.
- Active campaigns are editable; existing orders retain their original monetary and campaign snapshots.

## Defaults and public contract

Codes: trim outer whitespace, uppercase ASCII, validate `[A-Z0-9_-]{3,32}`, unique permanently.
The code is immutable after creation; staff may edit all other campaign settings.
No delete endpoint. Disable via enabled=false. Optional minimum/cap are null when unset; supplied values
must be positive. End must follow start when both exist. Selected targeting must contain at least one ID;
all targeting requires empty ID arrays. Reject unknown IDs; deduplicate and sort selected IDs.

Campaign JSON (money in rubles, ID arrays never null):
```json
{"id":1,"code":"AUTUMN10","percentage":10,"enabled":true,"target":"selected",
 "productIds":[1],"categoryIds":[2],"minimumEligibleSubtotal":1000,"maximumDiscount":500,
 "startsAt":null,"endsAt":null}
```
Creation/update bodies use the same writable fields; omit id. Update is full replacement, code immutable.

| Method/path (relative to /api/v1) | Role | Result |
|---|---|---|
| GET /promo-codes | admin | 200, array ordered by descending id, including disabled codes |
| GET /promo-codes/{id} | admin | 200, campaign |
| POST /promo-codes | admin | 201, campaign |
| PUT /promo-codes/{id} | admin | 200, updated campaign |
| POST /orders/quote | public | 200, authoritative pricing, no writes |

Quote body:
```json
{"items":[{"productId":1,"amount":2}],"fulfillment":"delivery","city":"Белореченск","promoCode":"AUTUMN10"}
```
Pickup uses fulfillment=pickup and no city. Require city for delivery; use the same town lookup and product
availability rules as order creation. No name, phone, street, or customer identity required for a quote.
Missing/empty code means no discount. Quote result:
```json
{"subtotal":1200,"eligibleSubtotal":1200,"discount":120,"total":1080,"deliveryPrice":100,
 "payableTotal":1180,"promoCode":"AUTUMN10","items":[{"productId":1,"amount":2,
 "offeredPrice":600,"discount":120}]}
```
Example assumes configured delivery cost 100 and a free-delivery threshold above 1080; it is a specification
fixture, not captured server output. Without a code eligibleSubtotal=0, discount=0, promoCode=null.
Order responses add subtotal, discount, promoCode and per-line discount; total remains food payable,
deliveryPrice remains separate. POST /orders gains optional promoCode and optional expectedPayableTotal.
For promo orders expectedPayableTotal is required; mismatch returns 409 before creating a new order.
Existing no-code callers may omit it. Existing duplicate retries compare against their saved payable amount.
Update docs/API.md and OpenAPI together, then regenerate admin types.

## Money and consistency

Use money.Kopecks and bigint columns throughout; never use floating point for discount arithmetic.
Merge duplicate product IDs first, resolve current server prices, then compute eligible subtotal E.
Reject a code if no positive eligible spend or E is below its minimum. D = floor(E * percentage / 100),
then min(D, cap) if configured. Use checked arithmetic. Allocate D proportionally by eligible line subtotal:
floor(D * lineSubtotal / E), distribute leftover kopecks by descending fractional remainder,
tie-break productId ascending. No ineligible line receives a discount. Quantities remain unchanged.
Total = subtotal - D. Delivery = existing town rule evaluated on total; pickup delivery=0.
Rounding down ensures a positive priced food order cannot become free with 1–99%; retain existing
validation for zero-priced carts rather than adding a new payment-free flow.

Persist original offered prices, per-line discount and campaign ID/code/percentage snapshot alongside order
subtotal/discount; historical reads never recalculate from live campaigns. Backfill old orders with zero discount.
Read a consistent campaign + selection snapshot within the order transaction (lock campaign row against edits;
admin updates lock the same row before replacing targets). Reuse pricing logic in quotes and order creation.
Include normalized promo code in existing duplicate matching, preserving ownership/contact/address criteria.
Existing pending-order retries use original snapshots despite campaign edits, disable or expiry; retain existing
product availability safety checks. A fresh order validates the campaign again. Never silently remove a code.
Payment forms, check/pay callbacks and retry links use persisted total + deliveryPrice.
Reject item mutations on orders with promo snapshots with 409; do not retroactively change a paid discount.

## Errors and interface behavior

Use existing httpx validation envelopes and existing 401/403 behavior. Field errors use these exact texts:
- promoCode malformed: «Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.»
- promoCode unavailable (unknown/disabled/not started/expired): «Промокод недействителен.»
- promoCode no eligible items: «В корзине нет блюд для этого промокода.»
- promoCode minimum: «Недостаточная сумма подходящих блюд для промокода.»
- code duplicate (409): «Такой промокод уже существует.»
- code immutable: «Код нельзя изменить после создания.»
- percentage: «Укажите целый процент от 1 до 99.»
- minimumEligibleSubtotal / maximumDiscount: «Укажите положительную сумму.»
- endsAt: «Дата окончания должна быть позже даты начала.»
- productIds / categoryIds unknown: «Выбранное блюдо или категория не найдены.»
- target / selection shape: «Выберите все блюда или укажите блюда и категории.»
- expectedPayableTotal missing/invalid: «Подтвердите итоговую сумму заказа.»
- changed total (409): «Сумма заказа изменилась. Проверьте итог и повторите оформление.»
- missing campaign (404): «Промокод не найден.»
- promo-order item mutation (409): «Нельзя изменять состав заказа с промокодом.»
Use existing messages for unrelated order/cart/town errors and malformed JSON. Unexpected I/O errors remain 500.

UI labels: «Промокод», «Применить», «Удалить», «Скидка по промокоду», «Итого»,
«После скидки доставка стала платной». Retain entered text after failure. Applying/replacing/removing a code,
changing items, town or fulfillment triggers a new quote. Ignore stale responses; disable submit while a quote
is pending/failed. On creation 409 fetch a fresh quote, show the message and require another user submission.
Never redirect automatically after changed pricing. No public campaign browsing or suggested-code list.

## iiko prerequisite and release

Task 84 verifies the official iiko wire contract before integration is implemented. It must resolve discount
representation, rounding/quantity reconciliation and any restaurant-side setup. No fabricated discount IDs,
no live test orders, no assumption that merely reducing payment sum is accepted. Record evidence and exact
payload fixtures in IIKO_PROMOCODES.md. If unavailable, record a blocker and leave task 84 unchecked.
The current iiko payment represents food only; delivery remains in its existing paid-delivery comment.
Deploy backend 85–89 before enabling either checkout. Verify discounted food, iiko payment, MONETA charge
and delivery reconcile; keep campaigns disabled until checks pass. Disabling campaigns stops new redemptions
without invalidating existing payable orders. No refund, loyalty, usage analytics or coupon stacking work.

## Task sequence

Backend: 84 iiko contract → 85 persistence → 86 admin API and 87 quotes → 88 orders → 89 integrations.
Admin: 27 campaign editor after backend86 → 28 order breakdown after backend89.
Website: 43 shared checkout after backend89 → 44 history/QA after admin28.
See each project's PROGRESS.md for linked unchecked tasks.

## Verification

Backend: format/build/vet, full serial Go suite, >=80% statement coverage for every changed internal package.
Use TEST_DB_DSN from .env.development; never start a database, skip tests or substitute mocks for database tests.
Admin: check/build and targeted real-backend Playwright flows. Website: check/build and production-build QA.
Use httptest for external integrations; no real payments or iiko writes in automated tests.
Cover normal/error/boundary cases, admin authorization, campaign concurrency, rounding, immutable retries,
legacy/no-code orders, both fulfillment modes, both checkouts, mobile/desktop and both themes.
Documentation-only preparation: validate links, numbering, dependencies and unchanged existing progress text;
no application tests or services needed. Stop after writing documents; do not implement or tick tasks.
