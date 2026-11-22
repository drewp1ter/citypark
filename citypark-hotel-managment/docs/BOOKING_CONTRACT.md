# Hotel booking integration contract v1

Status: **frozen for implementation, 2026-11-16**. Product decisions are from the
owner; technical defaults below resolve the implementation details. This is a
documentation/discovery completion, not runtime or deployment acceptance. A
separate development file base has now been identified by the owner through
untracked `1c.txt`, and CityParkHotelDev is confirmed to point to it. Task 02
cannot pass acceptance until owner-run Designer and runtime checks succeed.

## Scope

Website → Go → authenticated 1C HTTP service. One physical room, free for the
entire stay, instantly confirmed only after the reservation transaction commits.
No prepayment, multi-room cart or guest cancellation/change API.
1C remains authoritative for inventory. **Owner decision (2026-11-09): Go calculates
online booking prices, not 1C.** This supersedes tariff ownership in the original
task specifications. Go uses `rooms.price_kopecks`; the accepted snapshot is persisted in 1C as specified below.

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

## Agreed product rules and implementation defaults

- Required guest/contact fields: name, email, phone. No passport fields, child-age
  or extra-bed questions. Keep one exact total `guests` count from the room search.
  Capacity limits: single/single-balcony 1, twin/Euro 2; VIP 4, matching the existing
  site description. Do not derive capacity from the shared 1C room-type reference.
- Dates only; minimum one night and no business maximum stay. Earliest arrival is
  tomorrow in `Europe/Moscow`. There is no business limit on how far ahead arrival
  or departure may be booked (owner revision, 2026-11-16); the earlier six-month
  horizon is superseded. Use half-open intervals `[arrival at configured check-in,
  departure at configured check-out)` in Moscow local time. The authoritative
  times are `Константы.ВремяЗаселения` and `Константы.ВремяВыселения`, shared with
  staff workflows. Owner reports production check-in is 13:00; production check-out
  is not yet confirmed. Owner authorized this revision on 2026-11-20, superseding
  the earlier noon-to-noon rule. Do not change hotel constants to satisfy online
  booking. Customer copy must match the verified configured times; the earlier
  check-in-after-12:00 copy must be corrected before release. No early/late
  options in v1. Dates must be valid Gregorian dates representable by 1C and the
  backend. The 1C database storage range is years 0001–3999; years outside
  that range return `400 INVALID_REQUEST` before inventory queries or writes.
  Unlimited advance booking and duration do not bypass date-type limits,
  checked price arithmetic or full-stay inventory validation.
- Anonymous guest booking allowed; no prior email verification. Confirmation goes
  to the supplied email after 1C commits. Email failure cannot undo the booking.
- Nights are the difference between the two calendar dates, independent of the
  configured arrival/departure clock times. For example, check-in at 13:00 on
  December 1 and check-out at 12:00 on December 2 is one night. Availability and
  posting use the actual configured timestamp interval. Adjacent calendar stays
  are allowed only when their actual timestamp intervals do not overlap.
- Go computes `nights × rooms.price_kopecks` using checked integer arithmetic,
  independent of guests. No extras, seasonal schedules or discounts in v1. Reject
  missing/nonpositive rates and overflow. Public JSON money uses rubles, following
  existing `money.Kopecks` serialization; internal 1C requests use integer kopecks.
- Configured clock times are applied consistently within each availability/new
  reservation request. They are not client input and are not added to canonical
  request JSON. Same-key replay and GET recovery return the original saved result
  without recalculating times or moving the existing reservation when constants
  change. A new request uses current constants; advisory availability is rechecked
  during reservation creation.
- One physical room must cover the complete interval. Reuse existing availability
  semantics, including blocking non-free occupancy; retain the staff housekeeping
  policy rather than inventing a new cleaning buffer. Disabled/deleted references
  never allocate. Quote availability is advisory until creation commits.
- Technical validation: trim name (1–150 Unicode characters), trim and validate
  email (max 254 bytes, preserve local-part case and lowercase domain), normalize
  phone to `+` followed by 8–15 digits (strip spaces, parentheses and hyphens;
  Russian 8 plus ten digits becomes +7). Reject control characters and unknown JSON
  fields. No identity/account linking or counterparty matching by unverified email.
- Rate limits: availability/quote 30/minute/IP; creation attempts 5/hour/IP plus
  3/hour/normalized contact-pair hash. Existing-key recovery/replay 30/minute/IP,
  outside new-creation quotas. Persist counters in PostgreSQL for multiple workers;
  hash contacts, trust forwarded IP only from configured trusted proxies. No PII
  or request bodies in logs. These are technical defaults, not email verification.

## Stable mapping required

Use data-reference UUIDs from the intended information base, not metadata UUIDs, labels or
mutable display numbers. Each enabled room type must map to an organization,
building, 1C room type and an explicit eligible physical-room set. Identify any
1C tariff/price-type references required for document structure separately from
the backend rate source; they do not make 1C the online price calculator.
Revalidate references and their relationships before activation.

Owner supplied the following room-number mapping (2026-11-09). Room numbers
identify the intended inventory for reference lookup; they are not stable API keys.

| Website ID | Room numbers | Owner description | Physical-room reference UUIDs |
| --- | --- | --- | --- |
| `vip` | 301 | VIP-номер; две комнаты (гостиная + спальня), балкон; 45 кв. м + балкон 7 кв. м | 301: `fb170aba-5e75-11e4-80cd-b8a38697f1fc` |
| `single-balcony` | 201 | Одноместный номер с двуспальной кроватью и балконом; 23 кв. м + балкон 7 кв. м | 201: `fb170ab4-5e75-11e4-80cd-b8a38697f1fc` |
| `single` | 204, 304 | Одноместные номера с двуспальной кроватью; 19 кв. м | 204: `fb170ab7-5e75-11e4-80cd-b8a38697f1fc`; 304: `fb170abd-5e75-11e4-80cd-b8a38697f1fc` |
| `double-twin` | 203, 206, 302, 303 | Двухместные номера с двумя односпальными кроватями, которые можно сдвинуть; 23–25 кв. м | 203: `fb170ab6-5e75-11e4-80cd-b8a38697f1fc`; 206: `fb170ab9-5e75-11e4-80cd-b8a38697f1fc`; 302: `fb170abb-5e75-11e4-80cd-b8a38697f1fc`; 303: `fb170abc-5e75-11e4-80cd-b8a38697f1fc` |
| `double-euro` | 202, 205, 305, 306 | Двухместные номера с двуспальной кроватью евро; 23–25 кв. м | 202: `fb170ab5-5e75-11e4-80cd-b8a38697f1fc`; 205: `fb170ab8-5e75-11e4-80cd-b8a38697f1fc`; 305: `fb170abe-5e75-11e4-80cd-b8a38697f1fc`; 306: `fb170abf-5e75-11e4-80cd-b8a38697f1fc` |

Owner supplied the reference response on 2026-11-16: all 12 room UUIDs above are
unique, match the expected printed numbers, and have `deletionMark: false`.
All supplied room types, buildings and complexes also have false deletion marks.

The five website types do **not** map one-to-one to 1C room types:

| Website types | Supplied 1C room-type UUID |
| --- | --- |
| `single-balcony`, `double-twin`, `double-euro` | `fb170ab1-5e75-11e4-80cd-b8a38697f1fc` |
| `single` | `fb170ab2-5e75-11e4-80cd-b8a38697f1fc` |
| `vip` | `fb170ab3-5e75-11e4-80cd-b8a38697f1fc` |

Allocate from the explicit physical-room UUID set for each website type. Filtering
only by 1C room type would mix balcony, twin and Euro rooms.

The response also supplies building `fb170ab0-5e75-11e4-80cd-b8a38697f1fc`,
complex `fb170aaf-5e75-11e4-80cd-b8a38697f1fc` for all rooms, and one
organization candidate `fb170aae-5e75-11e4-80cd-b8a38697f1fc`. The presence of
one organization does not prove it is the booking organization; confirm during
integration setup rather than inferring an ownership relationship.

The pasted response contained question marks in Cyrillic labels. A direct,
owner-authorized GET on 2026-11-16 returned HTTP 200 with
`Content-Type: application/json; charset=utf-8`; the body parsed as UTF-8 JSON
and contained `Номер 201` and `Двухместный (обычный)`. Server-side Cyrillic
serialization is working in the observed deployment. The earlier corruption
occurred after response delivery; the exact client/display/save step is unknown.
Do not use display labels as mapping keys.
The response contains building/complex/organization fields absent from the latest
local exporter, indicating a different deployed/export revision; exact cause is
unverified. This evidence is not a runtime pass for the latest export revision.
Do not infer VIP capacity from its floor area or number of rooms.

Website IDs were checked in `../citypark-modern/src/entities/room/api/get-rooms.ts`.
Source XML contains catalog structure, not the live hotel room records.

## HTTP contract

All bodies are UTF-8 JSON; unknown fields are rejected. Success examples below
specify the exact keys. Private responses use `Cache-Control: no-store`. The public
API is under `/api/v1/hotel`; internal 1C service root is `/hs/booking/v1` under the
existing publication. Do not expose internal routes through public Next.js proxies.
1C uses publication Basic authentication and a dedicated restricted role, without
privileged mode. Credentials are server-only; production activation requires HTTPS
or a protected transport. Current HTTP discovery access is not production approval.

### Internal 1C availability

`POST /availability`, handler `availabilityPost`:

```json
{"roomTypeId":"single-balcony","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1}
```

200:

```json
{"roomTypeId":"single-balcony","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"available":true}
```

Unavailable inventory returns the same shape with `available:false`, not a price
or physical-room list. Use explicit room sets from the mapping above. Configuration
resolves the organization and service responsible-user references; derive building
from the selected room internally. Organization candidate above must be verified
by the installer before activation, not guessed from the catalog's row count.

### Internal 1C creation and recovery

`POST /reservations`, handler `reservationsPost`:

```json
{"idempotencyKey":"eaf9d7d3-2b21-43a9-8e27-e718b65d2588","roomTypeId":"single-balcony","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"guest":{"name":"Иван Иванов","email":"ivan@example.com","phone":"+79180000000"},"price":{"currency":"RUB","nights":2,"nightlyPriceKopecks":400000,"totalKopecks":800000}}
```

201 for first committed creation; 200 for same-payload replay:

```json
{"reservationId":"6e37e63e-b130-4ff6-a1b4-e0166f470922","status":"confirmed","roomTypeId":"single-balcony","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"price":{"currency":"RUB","nights":2,"nightlyPriceKopecks":400000,"totalKopecks":800000}}
```

`GET /reservations/{idempotencyKey}`, handler `reservationGet`, returns 200 with
exactly the saved creation result, including after staff cancellation; this is a
creation-recovery API, not current cancellation status. Missing committed key:
404. Reject noncanonical UUID keys. Creation replay identity compares all normalized
request values above (including price) independent of JSON key order. Store
canonical JSON for equality, not only a collision-prone digest. Retain key/result
records for the reservation lifetime; no automatic expiry/reuse in v1.

Reuse `Документы.Бронирование.СоздатьДокумент()`, filling and normal posting.
After filling, explicitly replace default dates/guest count and `Проживание` rows.
Populate daily `Сутки` row with calendar nights, configured arrival/departure times and backend price in
`Стоимость`/`Сумма`; do not invoke the 1C price calculator for online reservations.
Create a separate counterparty per new reservation, with the required name; save
email/phone in restricted online-booking metadata, not a public comment. Replays
create neither another counterparty nor another document. Preserve the accepted
cost rows during `Заселение` creation and automatic form recalculation for these
online reservations; explicit staff edits remain staff responsibility. Do not
change default pricing of ordinary staff-created bookings.

Staff with document edit permission can explicitly change an open online
booking/check-in cost row using «Изменить цену» on its «Стоимость» tab. The command
changes the selected row's nonnegative unit price, rounds its amount to kopecks
and updates the document total; saving/posting remains explicit. It does not
change stay dates/quantity, hotel tariff settings or the immutable API request
and creation result. The dedicated service role cannot invoke this command;
closed/deletion-marked documents cannot use it. Ordinary staff-created bookings
retain their existing automatic pricing. Native validation of the command is
required before accepting this task's manual-price-edit scenario.


### Public Go API

`POST /quotes` accepts the internal availability request shape. 200:

```json
{"quoteId":"97a98c7c-109f-4d01-aac1-3bdce27cbf47","expiresAt":"2026-11-16T12:10:00Z","roomTypeId":"single-balcony","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"available":true,"price":{"currency":"RUB","nights":2,"pricePerNight":4000,"total":8000}}
```

No availability: 409 `ROOM_UNAVAILABLE`, no quote. The same `X-Hotel-Booking-Context` header described below is required on quotes
and recovery as well as creation. Persist a 10-minute quote and price snapshot. Read rate through the existing room store; `nights` is a calendar
date difference, not elapsed UTC hours. Recheck dates and current rate on initial
creation; changed rate returns 409 before any 1C write. Never send a new total
without obtaining a fresh quote and explicit customer submission.

`POST /reservations` accepts exactly `idempotencyKey` (UUID), `quoteId` (UUID) and
`guest` (shape above). The website server supplies an opaque booking-context token
in `X-Hotel-Booking-Context`; it is 32 cryptographically random bytes encoded as
base64url, hashed in Go storage and kept in a signed HttpOnly SameSite=Lax cookie
in the browser. Bind quote/key records to this context. No public caller sends a
physical-room reference or authoritative price. Public response 201/200:

```json
{"reservationId":"6e37e63e-b130-4ff6-a1b4-e0166f470922","status":"confirmed","roomTypeId":"single-balcony","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"price":{"currency":"RUB","nights":2,"pricePerNight":4000,"total":8000},"confirmationEmailStatus":"pending"}
```

Email status is `pending` or `sent`; failed attempts remain pending for retry.
`GET /reservations/{idempotencyKey}` requires the same context, returns the same
public result, or 202 `{"status":"pending"}` for an unresolved creation. Wrong
context/unknown key returns 404 without revealing another booking. Replay after
expiry/rate change uses the persisted original request/result; no fresh repricing.

### Errors and bounds

Error JSON: `{"statusCode":409,"code":"ROOM_UNAVAILABLE","message":"На выбранные даты нет свободного номера"}`.
Use this envelope for both services (publication authentication errors may be
platform-generated). Exact codes/messages:

| Status | Code | Russian message |
| --- | --- | --- |
| 400 | INVALID_REQUEST | Проверьте данные бронирования |
| 401 | UNAUTHORIZED | Требуется авторизация |
| 403 | FORBIDDEN | Доступ запрещён |
| 404 | RESERVATION_NOT_FOUND | Бронирование не найдено |
| 409 | ROOM_UNAVAILABLE | На выбранные даты нет свободного номера |
| 409 | PRICE_CHANGED | Стоимость проживания изменилась. Подтвердите новую стоимость |
| 409 | IDEMPOTENCY_CONFLICT | Ключ бронирования уже использован с другими данными |
| 409 | QUOTE_EXPIRED | Расчёт стоимости устарел. Обновите стоимость |
| 429 | RATE_LIMITED | Слишком много запросов. Попробуйте позже |
| 503 | BOOKING_UNAVAILABLE | Бронирование временно недоступно |

Invalid/oversized body: 400; maximum 16 KiB. Transport/parse/unexpected upstream
failure before creation: 503. After ambiguous creation: 202 pending, never a
claim that booking failed. Bound internal responses to 64 KiB and HTTP timeout
to 15 seconds; context cancellation is honored. Disable redirects so credentials
cannot be forwarded. Never log credentials, contacts or response bodies.

## Atomicity, locking, uncertain results and email

- Add independent information register `КлючиОнлайнБронирования`: UUID-string key,
  canonical request, document reference, immutable JSON result and contacts. Acquire
  exclusive managed key lock before reading/writing; persist in the same transaction
  as counterparty creation, reservation posting and occupancy movements.
- Add independent information register `БлокировкиНомеровБронирования` with room
  reference dimension as the shared managed-lock namespace (no business rows
  needed). Lock all affected old/new room references in UUID order before checking
  occupancy and hold to transaction end. All posting/unposting paths for booking,
  check-in, cancellation/undo and checkout/undo acquire the same locks, including
  staff edits. HTTP creation locks candidate room sets in that same order after
  locking its idempotency key. Staff paths never acquire idempotency locks. Query
  old document room before mutation; protect against concurrent edits/recheck it.
- Reuse the full-interval availability helper with basis exclusion for reposting;
  fix any helper slice/self-conflict defects found by required runtime tests instead
  of duplicating occupancy logic in HTTP. Shared room locks serialize competing
  allocations even across organizations. Cancel/unpost/create rollback is atomic.
- Go persists the normalized attempt and its context before dispatch. Never hold a
  PostgreSQL transaction across HTTP. On timeout, retain pending and recover by
  internal same-key GET; 404 can mean the original request is still executing.
  Background recovery checks every 30 seconds and may resend the identical saved
  create using the same key. Never regenerate a key or reprice a pending request.
- On confirmation, Go saves result and one uniquely keyed email-outbox row in the
  same local transaction. Reconciliation after a lost local commit recreates this
  using the same key. Use existing SMTP mailer; retries with bounded backoff continue
  while pending. SMTP delivery is at-least-once: a lost acknowledgement may cause
  duplicate email, never duplicate booking. Confirmation includes reservation
  reference, room type, dates, guests, RUB total and hotel contact phone; no payment
  link. Worker owns retries so browser refresh is not required.

## Implementation map and named acceptance scenarios

### 1C task 02

Create `HTTPServices/Booking.xml`, `HTTPServices/Booking/Ext/Module.bsl`,
`CommonModules/ОнлайнБронирование.xml` and its `Ext/Module.bsl`, the two independent
registers named above (root `.xml` files under `InformationRegisters/`), the
restricted `НастройкиОнлайнБронирования` setup register, and
`Roles/ОнлайнБронированиеСервис.xml` with `Roles/ОнлайнБронированиеСервис/Ext/Rights.xml`.
Register metadata in `Configuration.xml` and `ConfigDumpInfo.xml` following the
existing export format. Preserve existing object UUIDs.

HTTP exports: `availabilityPost(Запрос)`, `reservationsPost(Запрос)`,
`reservationGet(Запрос)`, returning `HTTPСервисОтвет`. Shared server exports:
`ПроверитьДоступность(Данные)`, `СоздатьБронирование(Данные)`,
`ПолучитьРезультат(Ключ)`, `ЗаблокироватьНомера(Номера)`.
Use a dedicated configured service user/organization and explicit room mapping;
store setup values in restricted service configuration, never HTTP request fields
or source-code credentials. Start disabled until references are verified. The
setup register has one website-type key and enabled/organization/responsible-user/
1C-room-type/explicit-room-UUID-JSON fields. Its rows are absent by default. The
service role can read, but cannot edit them; ordinary hotel staff cannot activate
them or change idempotency records. Accepted idempotency records are immutable
through their record-set writer. See the development setup and scenario handoff in
[BOOKING_RUNTIME_RESULTS.md](BOOKING_RUNTIME_RESULTS.md).

Update object modules for `Бронирование`, `Заселение`, `ОтменаБронирования`,
`Выселение` to share locks. Adapt booking/check-in form recalculation to preserve
online prices. Store online origin/accepted snapshot through the idempotency
register's document reference; staff documents without a matching record retain
normal pricing. Use the existing daily predefined tariff (not a guessed UUID).

Record platform and results under `docs/BOOKING_RUNTIME_RESULTS.md`. Named scenarios:
`BookingValid`, `FullStayOneRoom`, `AdjacentStays`, `OverlapVariants`,
`WebsiteStaffRace`, `DuplicateKeyRace`, `KeyPayloadConflict`,
`PostingRollback`, `CommitTimeoutRecovery`, `CancellationUndoRace`,
`RepostSelfExclusion`, `BackendPriceAtCheckIn`, `UnauthorizedAccess`, `RestrictedRole`.
Each includes document/register assertions. Freeze metadata lock compatibility in
Designer testing; failure blocks task 02 rather than weakening the lock invariant.

Task 02 acceptance scope amendment (owner decision, 2026-11-22):
`ConcurrentRoomMove` is excluded from required acceptance, including the manual
three-way two-staff-session/API orchestration and its reverse-order/stale-snapshot
variants. It is not a completion blocker or a deferred mandatory test. Prior
attempts remain historical evidence: a managed-lock conflict was observed and
its cause was not diagnosed; this exclusion does not turn that result into a pass
or prove shared locking correct. Existing shared-room-lock implementation and
other required scenarios (including `WebsiteStaffRace`, `CancellationUndoRace`
and `RepostSelfExclusion`) remain in scope. No technological-log setup or further
manual repeat is required for the excluded scenario.

Online checkout pricing: checkout linked through a check-in to an online
booking uses the saved check-in cost-row rates, including explicit staff edits,
instead of silently applying current 1C tariff prices. Recalculation clips saved
rate periods to checkout stay periods; unchanged source periods keep saved
quantities, partial periods use the existing tariff quantity helper. A period
without saved rate coverage must be corrected in the check-in before checkout.
Order charges remain included; immutable API creation results are not rewritten.
Ordinary checkout retains its previous pricing path. Native validation required.

### Backend tasks 106–107

Task 106 creates `internal/hotelbooking/client.go`, `types.go` and `client_test.go`.
`NewClient(baseURL, user, password string, httpClient *http.Client) *Client`;
`Availability(ctx context.Context, req AvailabilityRequest) (AvailabilityResponse, error)`;
`Create(ctx context.Context, req CreateRequest) (Reservation, error)`;
`Lookup(ctx context.Context, key string) (Reservation, error)`.
Types follow the internal JSON shapes above; timestamps are not fabricated by 1C.
Use typed errors for protocol status/code and ambiguous create transport outcomes.
Add `HOTEL_BOOKING_ENABLED` (false), `ONEC_BOOKING_URL` (service root ending
`/hs/booking/v1`), `ONEC_BOOKING_USER`, `ONEC_BOOKING_PASSWORD` in config/env docs;
validate all required integration and SMTP settings when enabled. Keep reporting
credentials/config independent. Tests: `TestClientAvailability`, `TestClientCreate`,
`TestClientLookup`, `TestClientProtocolErrors`, `TestClientResponseLimit`,
`TestClientTimeoutCancellation`, `TestClientCredentialIsolation`,
`TestClientAmbiguousCreateRecovery`, `TestBookingConfig`.

Task 107 adds `handler.go`, `service.go`, `store.go`, `worker.go` and corresponding
`*_test.go` in that package. Constructors inject store, room-price reader, client,
clock and mailer. Handler `Routes(r chi.Router)` registers the public paths above;
`Quote`, `Create`, `Lookup` use existing error-returning HTTP handler conventions.
Service signatures: `Quote(ctx context.Context, contextToken string, req QuoteRequest) (QuoteResponse, error)`, `Create(ctx context.Context, contextToken string, req PublicCreateRequest) (PublicReservationResponse, error)` and `Lookup(ctx context.Context, contextToken, key string) (PublicReservationResponse, error)`. PublicReservationResponse represents pending or confirmed with conditional fields matching the wire shapes; never serialize empty confirmed fields for pending. Typed DTOs are derived
from the exact wire shapes above. Store methods take context first.

Add the next available embedded migration (do not reserve an already-used number):
`hotel_booking_quotes` (UUID, context hash, stay/guests, immutable rate/total and
expiry), `hotel_booking_attempts` (key primary key, quote FK, context hash,
canonical request, pending/confirmed/rejected state, saved result),
`hotel_booking_email_outbox` (attempt key unique, delivery state/attempt count,
next attempt time), `hotel_booking_rate_limits` (scope/hash/window counters).
Quote rows referenced by attempts and attempt/key records have no automatic
expiry deletion. Money columns are bigint kopecks. Store sensitive request/result
JSON only as needed for recovery/delivery, with restricted DB access.

Worker supports multiple processes via row claims/leases and retry backoff
30s, 1m, 5m, then 30m; recovery remains pending on persistent upstream failures.
Worker stops on application context cancellation. Document operational monitoring
of pending age and email backlog without PII. Add app wiring and `docs/API.md` /
`internal/apidocs/openapi.yaml` from real handler test examples.
Tests: `TestQuoteCalendarBounds`, `TestQuoteCapacity`, `TestQuoteMoney`,
`TestCreateRateChange`, `TestCreateIdempotency`, `TestConcurrentCreate`,
`TestPendingRecovery`, `TestContextIsolation`, `TestGuestValidation`,
`TestRateLimits`, `TestOutboxRecovery`, `TestMailerRetry`, `TestNoPIILogging`.
Backend serial DB tests and changed-package coverage requirements apply.

### Website task 73

Create `app/(site)/booking/page.tsx` re-exporting `src/pages/booking`, with
`api/actions.ts`, `model/schema.ts`, `ui/booking-page.tsx`, `ui/booking-form.tsx`
and `index.ts` in that page slice. Put the signed booking-context cookie helper
and server-only backend requests in that slice's `api/` segment.
`getQuoteAction` / `createReservationAction` / `lookupReservationAction` preserve
values, UUID key and pending status. Load guest form from selected room type,
check-in/out and exact total guests; no child/extra-bed options. Search exact
counts 1–4 (replace ambiguous 3–4); server validates against selected capacity.
Update rooms search/detail links to `/booking` carrying these stay parameters.
A price refresh never automatically resubmits creation. Pending UI polls the
same key every five seconds while visible; reload recovers from signed context
and a saved key. Recovery works independently through the Go worker.

Use existing room content/photos/tokens and Russian labels `Имя`, `Электронная
почта`, `Телефон`, `Забронировать`, `Проверяем бронирование`,
`Бронирование подтверждено`. State that confirmation is sent by email; never
claim successful delivery while pending. No payment UI. Server-only integration;
no 1C credentials/references in client code. Private booking route is noindex.
QA `scripts/qa/flows/hotel-booking.mjs` stages: `search`, `guest-validation`,
`quote`, `confirm`, `double-submit`, `sold-out`, `price-change`, `pending-recovery`,
`outage-preservation`, `keyboard`. Run at 390/1440 px in both themes. Writes only
in an authorized isolated development environment; production builds/checks remain
required. None of these runtime scenarios has passed merely by writing this spec.

## Development acceptance matrix (all unrun)

| Scenario | Required assertion |
| --- | --- |
| Valid one-room booking | Posted open reservation, correct contacts/stay/cost and occupancy; response only after commit. |
| Full-stay allocation | Reject a stay where each night has space but no single room covers the whole interval. |
| Interval overlap | Reject partial overlap, contained/containing stays and identical dates. |
| Adjacent stays | Checkout boundary permits the next valid arrival without a phantom extra occupied second. |
| Guest fields | Name/email/phone required; no required additional guest/contact fields or child/extra-bed questions. |
| Capacity | Verified physical-room allocation rules enforced without child/extra-bed questions. |
| Booking horizon/duration | Reject today/past arrival dates using Moscow time; allow tomorrow when otherwise valid. Revalidate after Moscow midnight. Accept arrival and departure more than six months ahead and stays longer than 90 nights when inventory is available and dates/totals are representable. No maximum advance horizon or stay duration; no month-clamping rule. Reject checkout on/before arrival, invalid/unrepresentable dates and arithmetic overflow. Long stays require one room for the full stay and complete price calculation. |
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

## Completion and release boundary

Task 01 is documentation complete with the explicit schemas/defaults above, stable
room sets, source inspection and environment prerequisites. Examples are normative
contracts, not captured runtime responses. Task 02 implements and tests them in a
separate development base. The owner has now identified an isolated file base
and its CityParkHotelDev publication; account rights, organization selection,
timezone/constant verification, new-service publication/TLS,
Designer checks and concurrent runtime acceptance remain release requirements.
The discovery endpoint's earlier deployment is not booking-service acceptance.
See [BOOKING_ENVIRONMENT.md](BOOKING_ENVIRONMENT.md).
