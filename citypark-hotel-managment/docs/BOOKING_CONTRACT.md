# Hotel booking integration contract — discovery draft

Status: **not frozen**. Source inspection performed on 2026-11-09. Owner decisions,
live reference mappings and development-base verification are pending. Owner confirmed
that guest name, email and phone are required and no separate development base currently exists. This document
does not clear task 01 or authorize downstream implementation against a final API.

## Scope

Website → Go → authenticated 1C HTTP service. One physical room, free for the
entire stay, instantly confirmed only after the reservation transaction commits.
No prepayment, multi-room cart or guest cancellation/change API.
1C remains authoritative for inventory. **Owner decision (2026-11-09): Go calculates
online booking prices, not 1C.** This supersedes tariff ownership in the original
task specifications. The source of backend rates and persistence of the accepted
price into 1C must be finalized before freezing the contract.

## Verified source behavior

Paths below are relative to the hotel project root.

| Source | Observed behavior and implementation consequence |
| --- | --- |
| `Documents/Бронирование/Ext/ObjectModule.bsl` | Open-document posting checks `УправлениеЗаселением.НомерСвободенВДанномПериоде` and writes reserved occupancy for each `Проживание` row. No explicit room lock appears in this module. Filling defaults to one guest and one daily-tariff unit, even when values are passed; integration must explicitly populate the final stay after filling. |
| `Documents/Бронирование.xml` | Required metadata fields include building, room, arrival/departure, counterparty, organization and responsible user. Guest count and pricing fields do not have the same fill checks. Metadata checks alone are not service validation. |
| `Documents/Бронирование/Forms/ФормаДокумента/Ext/Form/Module.bsl` | The form recalculates stay dates and calls `Ценообразование.ЗаполнитьТаблицуСтоимостиПроживания`. A headless HTTP handler cannot rely on form events. |
| `CommonModules/УправлениеЗаселением/Ext/Module.bsl` | Arrival/departure times come from `Константы.ВремяЗаселения` / `ВремяВыселения`. Daily tariff dates use the departure constant. Tariff quantity uses rounding of duration divided by tariff hours. The availability query uses `СрезПоследних` at a supplied timestamp, organization/room filters, optional basis exclusion and overlap checks. It emits staff-facing document details through `Сообщить`; these must not enter public errors. |
| `InformationRegisters/СостояниеНомеров.xml` | Recorder-subordinate register, recorder-position periodicity; dimensions include organization, room and arrival date. Slice semantics must be exercised with multiple reservations for the same room before treating the helper as proven for online use. |
| `InformationRegisters/СостояниеНомеров/Ext/RecordSetModule.bsl` | Movement writer stores departure minus one second; the availability helper also adjusts the requested departure. This represents an exclusive checkout boundary at second precision. Adjacent-stay runtime acceptance is required. |
| `Documents/Заселение/Ext/ObjectModule.bsl` | Posting closes/reposts its basis, checks availability excluding the basis, writes occupied movements and financial movements. Staff check-in must participate in the same room-lock protocol as online booking. |
| `Documents/ОтменаБронирования/Ext/ObjectModule.bsl` | Cancellation closes/reposts the basis in a transaction; undo reopens/reposts it. Verify removal/restoration of occupancy and concurrent cancellation behavior in Designer/Enterprise. |
| `Documents/Выселение/Ext/ObjectModule.bsl` | Checkout closes/reposts its basis and writes completed occupancy. Undo clears movements and reopens the basis. Include both paths in the concurrency review. |
| `CommonModules/Ценообразование/Ext/Module.bsl` | Prices depend on date, organization, price type, room type and optionally tariff. Missing price defaults to zero. `ПолучитьМассивЦенПроживания` reads only the first selected price-change row; multiple changes during one stay need explicit regression coverage before reuse. |
| `DataProcessors/РабочееМестоАдминистратораГостиницы/Forms/Форма/Ext/Form/Module.bsl` | Staff workspace opens booking/check-in/cancellation document forms. Locking only the new HTTP handler would leave these paths unprotected. |

These are static observations, not runtime test results. In particular, existing
availability checks do not establish that concurrent posting is safe.

## Decisions required from owner

| Decision | Pending information |
| --- | --- |
| Guest identity/contact | **Owner confirmed: only name, email and phone are required guest/contact fields (2026-11-09).** Validation limits and counterparty creation/reuse policy remain to specify. Do not add required identity-document fields to online booking. |
| Occupancy | **Owner confirmed: do not ask guests about children or extra beds (2026-11-09).** Do not add these form questions or require child ages/extra-bed selections in requests. Internal room capacity/allocation still needs a verified 1C source; the inspected room metadata provides no explicit capacity attribute. |
| Dates | **Owner confirmed: dates only, no exact-time input; check-in after 12:00, check-out before 12:00; no maximum stay length; bookings six months ahead (2026-11-09).** Treat the advance horizon as an arrival-date constraint, independently of stay duration; this interpretation and exact calendar-month boundary need final specification. Do not introduce a maximum-night or checkout-within-horizon restriction. Verify actual 1C constants against noon. **Owner confirmed hotel timezone: Moscow (`Europe/Moscow`, 2026-11-09).** Interpret guest dates and noon boundaries in this timezone, independently of the Go host timezone. **Owner confirmed: same-day bookings are unavailable (2026-11-09).** Earliest arrival is tomorrow in `Europe/Moscow`; reject arrival dates on or before the current hotel-local date. Revalidate this rule when creating a reservation, including when a quote crosses Moscow midnight. Actual 1C timezone configuration, minimum stay and early/late arrival handling remain pending. |
| Confirmation | **Owner confirmed: send confirmation to the guest's email (2026-11-09).** Delivery ownership, retries and failure handling after a committed reservation remain to specify; email failure must not duplicate or undo a booking. |
| Abuse prevention | Anonymous versus signed-in access, rate limits, verification, duplicate contact policy and retention period for idempotency data. |
| Pricing | **Owner confirmed: Go calculates booking prices, not 1C (2026-11-09).** **Owner confirmed: use room prices managed in the admin panel (2026-11-09).** Nightly calculation, rate changes, rounding and missing-price handling remain to specify. Define how the backend's accepted price populates 1C reservation cost rows and survives staff check-in/repricing; existing 1C recalculation must not silently replace the online price. |
| Availability | Maintenance/dirty-room exclusion policy and which room states block online allocation. |

Only the explicit owner confirmations above are approved. Remaining technical
policies must be specified without reintroducing rejected contact/date defaults.
Rate limiting and other abuse-prevention details remain to finalize.

## Stable mapping required

Use data-reference UUIDs from the development base, not metadata UUIDs, labels or
mutable display numbers. Each enabled room type must map to an organization,
building, 1C room type and an explicit eligible physical-room set. Identify any
1C tariff/price-type references required for document structure separately from
the backend rate source; they do not make 1C the online price calculator.
Revalidate references and their relationships before activation.

Owner supplied the following room-number mapping (2026-11-09). Room numbers
identify the intended inventory for reference lookup; they are not stable API keys.

| Website ID | Room numbers | Owner description | Physical-room reference UUIDs |
| --- | --- | --- | --- |
| `vip` | 301 | VIP-номер; две комнаты (гостиная + спальня), балкон; 45 кв. м + балкон 7 кв. м | Pending lookup |
| `single-balcony` | 201 | Одноместный номер с двуспальной кроватью и балконом; 23 кв. м + балкон 7 кв. м | Pending lookup |
| `single` | 204, 304 | Одноместные номера с двуспальной кроватью; 19 кв. м | Pending lookup |
| `double-twin` | 203, 206, 302, 303 | Двухместные номера с двумя односпальными кроватями, которые можно сдвинуть; 23–25 кв. м | Pending lookup |
| `double-euro` | 202, 205, 305, 306 | Двухместные номера с двуспальной кроватью евро; 23–25 кв. м | Pending lookup |

The mapping contains 12 distinct physical rooms across five types. Organization,
building and catalog-reference relationships remain to verify against 1C data.
Do not infer VIP capacity from its floor area or number of rooms.

Website IDs were checked in `../citypark-modern/src/entities/room/api/get-rooms.ts`.
Source XML contains catalog structure, not the live hotel room records.

## Proposed wire protocol for review

Everything in this section is a proposal. Paths are relative to a future dedicated
1C HTTP service publication; the existing `reports` service remains unchanged.
Exact validation, examples and limits must be finalized after the decisions above.

* `POST /booking/v1/quotes`: proposed internal availability operation taking room
  type, hotel-local arrival/departure dates and any approved occupancy fields,
  without child/extra-bed questions. Returns availability for one continuous room
  and resolved check-in/out timestamps. **1C does not calculate or return a price
  quote.** The final operation name should reflect availability. Go builds the
  customer-facing price quote and expiry from backend rates. Neither operation
  reserves inventory.
* `POST /booking/v1/reservations`: same stay details, required contacts, quote
  backend quote reference, accepted price snapshot and UUID idempotency key.
  Go validates pricing/consent before dispatch; 1C rechecks inventory and persists
  the trusted backend price without calculating it;
  return confirmed only after commit. The public client cannot nominate arbitrary
  internal catalog references or prices.
* `GET /booking/v1/reservations/{idempotencyKey}`: authenticated server-only
  recovery of a committed result. The key is not a public authorization credential.
  Go must bind any public recovery operation to its original booking context.

Proposed responses contain an opaque reservation reference, room-type ID, stay,
guest count, total/currency and confirmation state. Do not expose physical-room
UUIDs, counterparty records, credentials, staff names or raw 1C exceptions.
Public responses use `Cache-Control: no-store`.

Candidate error vocabulary to finalize (not an implemented API):

| Status | Code | Russian message |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | `Проверьте данные бронирования` |
| 401 | `UNAUTHORIZED` | `Требуется авторизация` |
| 403 | `FORBIDDEN` | `Доступ запрещён` |
| 404 | `RESERVATION_NOT_FOUND` | `Бронирование не найдено` |
| 409 | `ROOM_UNAVAILABLE` | `На выбранные даты нет свободного номера` |
| 409 | `PRICE_CHANGED` | `Стоимость проживания изменилась. Подтвердите новую стоимость` |
| 409 | `IDEMPOTENCY_CONFLICT` | `Ключ бронирования уже использован с другими данными` |
| 409 | `QUOTE_EXPIRED` | `Расчёт стоимости устарел. Обновите стоимость` |
| 429 | `RATE_LIMITED` | `Слишком много запросов. Попробуйте позже` |
| 503 | `BOOKING_UNAVAILABLE` | `Бронирование временно недоступно` |

## Required transaction and recovery invariants

1. Persist the canonical request identity and reservation reference under a unique
   idempotency key in the same transaction as document posting and occupancy writes.
   Define canonicalization, contact normalization and exact compared fields in the
   final contract; a different request under the same key is a conflict.
2. Serialize duplicate keys before allocating inventory. A successful replay returns
   the original reservation, even if current prices or availability have changed.
3. Use one shared room-lock protocol for HTTP allocation, staff booking/check-in,
   edits, reposting, unposting and cancellation/undo paths. Acquire locks before the
   authoritative availability check; hold them through commit. Lock old and new
   rooms in deterministic order for moves. Final lock namespace and acquisition
   order need verification against actual managed-lock behavior in the dev base.
4. One candidate room must cover every instant of the complete stay. Combining
   different rooms across nights is forbidden. Quotes do not guarantee allocation.
5. Go calculates and validates the complete price before reservation creation.
   Persist an immutable accepted price snapshot with the reservation and include
   it in idempotency comparison. 1C receives prices only from the authenticated Go
   service, never directly from browser input, and must not independently reprice
   online reservations. Finalize rate-version/quote-expiry and cross-system failure
   handling without assuming a shared Go/1C database transaction. A price change
   requires explicit new consent; never silently replace the accepted total.
6. On rollback, neither the idempotency result nor occupancy/document writes survive.
   On timeout after commit, lookup and retries with the **same** key recover the
   saved result. A missing lookup after a timeout does not prove creation failed:
   the original request may still be running. Never switch to a new key automatically.
7. Confirmation delivery failure does not undo a committed booking or trigger a new
   creation. Define lookup behavior for subsequent staff cancellation separately
   from replay of the original creation result.

## Development acceptance matrix (all unrun)

| Scenario | Required assertion |
| --- | --- |
| Valid one-room booking | Posted open reservation, correct contacts/stay/cost and occupancy; response only after commit. |
| Full-stay allocation | Reject a stay where each night has space but no single room covers the whole interval. |
| Interval overlap | Reject partial overlap, contained/containing stays and identical dates. |
| Adjacent stays | Checkout boundary permits the next valid arrival without a phantom extra occupied second. |
| Guest fields | Name/email/phone required; no required additional guest/contact fields or child/extra-bed questions. |
| Capacity | Verified physical-room allocation rules enforced without child/extra-bed questions. |
| Booking horizon/duration | Reject today/past arrival dates using Moscow time; allow tomorrow when otherwise valid. Revalidate after Moscow midnight. Six-month arrival boundary checked; no arbitrary maximum stay or checkout-within-horizon limit. Long stays still require full-stay inventory and complete tariff calculation. |
| Email confirmation | Confirmation sent after commit; delivery failure/retry never creates a duplicate reservation. |
| Pricing | Go calculation covers rate changes, missing/zero price, rounding and date boundaries. 1C persists the exact accepted snapshot; staff check-in does not silently replace it. |
| Price consent | Changed/expired quote creates nothing; explicit refreshed consent is required. |
| Concurrent website/staff | With one eligible room and simultaneous attempts, exactly one overlapping reservation commits. |
| Duplicate key | Simultaneous identical requests create one document and return the same reference. |
| Changed payload | Same key with different stay/contact/consent fields returns conflict without writes. |
| Failure before commit | Inject posting failure; no reservation, occupancy or successful key record remains. |
| Timeout after commit | Drop response; same-key lookup/retry returns original reservation with no duplicate. |
| Cancellation and undo | Inventory released/restored as appropriate; concurrent allocation cannot overbook. |
| Edit/repost/unpost | Old/new room locking and existing staff workflows remain correct. |
| Unauthorized service access | Missing/incorrect credentials and unrelated roles cannot quote/create/read booking data. |
| Guest data isolation | Public recovery cannot retrieve another context's reservation; logs/errors contain no contact data or secrets. |

## Completion gate

Freeze exact JSON schemas, examples, limits, authentication and error envelopes;
resolve every owner decision and reference mapping; record the environment in
[BOOKING_ENVIRONMENT.md](BOOKING_ENVIRONMENT.md). Then expand downstream task 02,
backend 106/107 and website 73 with exact paths, signatures and named tests.
Until then those tasks remain blocked; this discovery draft is not task completion.
