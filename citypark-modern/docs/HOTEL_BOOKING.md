# Hotel reservation flow (task 73)

The implementation follows [Booking contract v1](../../citypark-hotel-managment/docs/BOOKING_CONTRACT.md)
and the [Go API contract](../../citypark-go/docs/API.md#hotel-booking). No new public HTTP endpoints
or OpenAPI operations are introduced in the website. All booking calls originate on the Next.js server.

## Routes and state

`/rooms?checkIn=YYYY-MM-DD&checkOut=YYYY-MM-DD&guests=1` searches a whole stay.
Exact guest counts are 1, 2, 3 and 4, on mobile and desktop. Arrival defaults to
today in Moscow, with checkout tomorrow. Capacity is one for single rooms, two
for twin/Euro and four for VIP.

Search requests `POST /hotel/quotes` for each eligible type using an ephemeral server-generated
context. These advisory quotes cannot be used to create a booking from the browser.
Only `ROOM_UNAVAILABLE` establishes sold-out inventory; configuration, network and rate-limit
failures display `Наличие уточняется`. Today's `/hotel/rooms` counts remain a separate homepage
information source and never establish availability for a future stay.

Room selection and detail links preserve room type, arrival, departure and exact occupancy in
`/booking?roomTypeId=single-balcony&checkIn=YYYY-MM-DD&checkOut=YYYY-MM-DD&guests=1`.
The private page has `noindex, nofollow`, `Cache-Control: private, no-store` and `Referrer-Policy: no-referrer`.

The form uses three Server Actions:

| Action | Go request | Browser behavior |
| --- | --- | --- |
| `getQuoteAction(stay)` | `POST /hotel/quotes`, exact stay fields | Shows quote total for explicit submission; never creates a reservation. |
| `createReservationAction(attempt, quote)` | `POST /hotel/reservations`, only key, quote ID and normalized guest | Shows confirmation only for a validated committed result. |
| `lookupReservationAction(key)` | `GET /hotel/reservations/{key}` | Recovers the saved result independently of quote expiry/rate changes. |

The server supplies `X-Hotel-Booking-Context` from a signed HttpOnly SameSite=Lax cookie;
the token never enters the client bundle or action result. Quotes and recovery use that same context.
The browser sends no authoritative price or physical-room reference to Go. API errors use the
contract's fixed Russian messages; upstream free-form text and contacts are not logged.

Before creation, the exact key/quote/normalized contacts are saved in **sessionStorage**, and the
key is added to the page URL as `attempt`. A synchronous guard prevents double dispatch. If local
storage fails, no creation is sent. Reload retrieves the key from the URL, the signed context from
the cookie, and the retry payload from sessionStorage. Closing the tab may discard that local
payload, but a saved URL and the cookie still recover a committed result through Go. The Go worker
owns background recovery and email delivery even when the browser is closed.

Transport failures after dispatch remain pending; a recovery 404 remains unknown. Neither outcome
rotates the key or obtains a new price. Pending/unknown forms lock their inputs, poll the same key
every five seconds while visible, and allow an explicit retry of the saved identical request when
it is available. A missing context cannot recover another session's booking; contact the hotel
before creating another booking. Definitive sold-out/expired/changed-price responses retain inputs
and require a fresh quote and a separate customer submission. Confirmation distinguishes pending
email delivery from an email already sent. There is no payment step.

## Runtime configuration

Set server-only `MENU_API_URL`, `CUSTOMER_PUBLIC_ORIGIN` and `HOTEL_BOOKING_COOKIE_SECRET`.
Generate a signing secret with `openssl rand -base64 48` (at least 32 bytes), store it in the
secret manager, and keep it identical across Next.js processes/restarts. Rotating it invalidates
existing booking contexts. The cookie uses `__Host-citypark_booking` and Secure on HTTPS origins;
local HTTP uses `citypark_booking_dev`. Production must use HTTPS. A missing secret fails closed
with the generic booking-unavailable message; do not use a public or checked-in fallback.

Go must have booking enabled and correctly configured for the intended 1C base and SMTP delivery.
No credentials or 1C UUIDs are added to the website. Next.js does not forward arbitrary incoming
IP headers to Go; quota attribution currently uses the immediate Next.js peer. If per-visitor
limits behind a proxy are required, configure a trusted, verified address chain with the operator;
never trust browser-supplied forwarding headers.

## Verification and development gates

Run `node scripts/qa/hotel-booking-harness.mjs`, `npm run check` and `npm run build`.
The isolated harness covers calendar/capacity/contact validation, strict response schemas,
size bounds, cookie signing/tampering, pending/error outcomes, stable keys and reload snapshots.
It does not establish 1C runtime behavior or mail delivery.

Run the production server on port 3111 with a signing secret in its environment.
`node scripts/qa/flows/hotel-booking.mjs` defaults to `readonly`; `HOTEL_BOOKING_QA_STAGE=keyboard`
also performs no backend mutations. Both check form input preservation, exact occupancy,
keyboard focus, noindex/no-store, browser errors and overflow at 390/1440 px in both themes.
Screenshots are written to `/tmp/citypark-hotel-booking-qa`. Set `BROWSER_PATH` when using an
existing Chrome installation instead of Playwright's bundled Chromium.

**Obtain owner approval and confirm the isolated development target before any other stage.**
Even searching persists quotes in PostgreSQL. Never run booking-write QA against production.
Set `HOTEL_BOOKING_QA_APPROVED=1` only after that approval, and provide:

- `HOTEL_BOOKING_QA_CHECK_IN`, `HOTEL_BOOKING_QA_CHECK_OUT`, `HOTEL_BOOKING_QA_ROOM`, `HOTEL_BOOKING_QA_GUESTS`.
- For creation: approved synthetic `HOTEL_BOOKING_QA_NAME`, `HOTEL_BOOKING_QA_EMAIL`, `HOTEL_BOOKING_QA_PHONE`.
- `BASE_URL` if the production preview uses a different origin.

Stages are `search`, `quote`, `guest-validation`, `confirm`, `double-submit`, `sold-out`,
`price-change`, `pending-recovery`, `outage-preservation` and `keyboard`.
`confirm` and `double-submit` create **one** development fixture, then reuse its signed browser
context and original key for confirmation/reload checks at the other three theme/width combinations.
The browser context is saved outside the repository in `/tmp/citypark-hotel-booking-qa/recovery-context.json`;
keep it private and remove it after acceptance.

The approved automatic test modes use the real development services:

- `HOTEL_BOOKING_QA_STAGE=double-submit HOTEL_BOOKING_QA_DROP_CREATE_RESPONSE=1` sends two
  immediate button clicks, asserts one creation action, waits for a real committed confirmation,
  and drops that response before browser delivery. The browser must recover the original key;
  later theme/width combinations reuse the same context and reservation.
- `HOTEL_BOOKING_QA_STAGE=price-change HOTEL_BOOKING_QA_AUTOMATIC_PRICE_CHANGE=1` uses the
  local development administrator credentials in memory. It changes the selected room rate by
  1 RUB after quoting, verifies rejection/input preservation and a fresh total without automatic
  creation, then restores the original rate. The operator must approve isolated rate edits first.
  Use free dates; the script refuses to round-trip an invalid legacy photo position.
- `HOTEL_BOOKING_QA_STAGE=outage-preservation HOTEL_BOOKING_QA_OFFLINE_QUOTE=1` disconnects
  the browser during a quote action, verifies retained inputs and the unavailable message, then
  restores connectivity and obtains a fresh quote. It creates no reservation. This proves browser
  transport failure handling; it does not simulate a 1C server outage.
- `node scripts/qa/hotel-booking-sold-out.mjs` needs `HOTEL_BOOKING_QA_APPROVED=1` and the approved
  `HOTEL_BOOKING_QA_EMAIL`. It obtains an advisory browser quote, creates **one** competing
  reservation through the real Go/1C service, and submits the now-stale quote. One rejected attempt
  is recovered with its original context/session in the other three theme/width combinations.
  Override `HOTEL_BOOKING_QA_CHECK_IN/CHECK_OUT` for a fresh free stay. It saves the winner key/context
  before dispatch; retain that file on any uncertain response and never generate a replacement key.

For operator-controlled `sold-out`, `price-change` and post-dispatch `outage-preservation`,
`HOTEL_BOOKING_QA_INTERACTIVE=1` pauses after quoting so the operator can arrange the real
condition before pressing Enter. Restore rates/service availability afterward. A pre-dispatch
503 is a definitive error and does not establish an ambiguous post-dispatch outcome.

For `pending-recovery`, supply `HOTEL_BOOKING_QA_ATTEMPT` and
`HOTEL_BOOKING_QA_STORAGE_STATE` (the context file from the original attempt). Arrange a real
pending/lost-response attempt first; the stage waits for Go reconciliation and checks reload.
The same cookie secret must remain active across the original attempt and recovery.

After every successful creation, staff must inspect the development 1C booking, exact guests,
full-stay occupancy, immutable price snapshot, and one-document/idempotency counts. Inspect the
confirmation email and its delivery status. Staff cancel the fixture afterward and verify inventory
release; recovery must still return the original creation result. No public cancellation endpoint
is invented. Record actual outcomes rather than marking these gates passed from a script alone.

## Recorded acceptance (2026-11-22)

The owner authorized tests against the isolated 1C database and confirmation emails to
`signup@retand.tech`. That authorization persists for this test scope. No production deployment
or hotel configuration change was performed. The existing local cookie signing secret was preserved.

| Scenario | Observed result |
| --- | --- |
| Search and selection | Pass at 390/1440 px, light/dark: full-stay results, exact VIP occupancy 4, and stay/type/guest parameters preserved in booking links. |
| Quote | Pass in all four states against real Go/1C. Initial single-balcony quote: 2 × 4,500 = 9,000 RUB. |
| Guest validation | Pass in all four states: missing name/email/phone rejected with all three inline errors. Isolated harness also covers capacity, calendar bounds and normalization. |
| Confirmation/double submit | Two immediate clicks dispatch one creation action and create one confirmed reservation. Confirmation/reload rendering passes in all four states using that original context/key. |
| Lost response and recovery | Real confirmed Server Action response dropped before browser delivery; pending UI recovers the same committed result by its original key. Separate recovery/reload flow passes in all four states. |
| Sold-out race | Real competing reservation commits after the browser quote; stale submission receives `ROOM_UNAVAILABLE` and preserves inputs. Original rejection recovery/rendering passes in the other three states. |
| Price change | Pass in all four states: real admin rate changed by 1 RUB after quoting; old consent rejected, input retained, changed total shown after explicit refresh, no automatic resubmission. Original 4,500 RUB rate restored after every case. |
| Outage preservation | Pass in all four states: browser disconnected during quote action, inputs retained, explicit retry succeeds after connectivity returns. |
| Keyboard/responsive | Four-state checks cover Tab navigation through dates/contact fields/submit, visible focus (including native calendar controls), keyboard guest selection, browser errors and horizontal overflow; screenshots saved under the private temporary QA directory. |
| Email | Go reports `confirmationEmailStatus: sent` for both successful fixtures. Delivery to the recipient inbox is not independently inspected. |
| Native cleanup | Owner confirms both 1C bookings have correct stay/guest/cost and are canceled. Public quotes then return 200/available for both stays; GET recovery returns each exact original confirmed result with email status sent. |

Fixture history (no active reservation remains after owner cancellation):

| Fixture | Stay; guests; total | Key / 1C reservation UUID |
| --- | --- | --- |
| Тест сайта 73 — double click/lost response | single-balcony; 2027-04-10–12; 1; 9,000 RUB | `914d36b6-9d7a-43db-970a-a10566248981` / `9e67c297-e6a7-11f1-82c4-e46f13aa4449` |
| Тест занятости сайта 73 — race winner | single-balcony; 2027-04-18–20; 1; 9,000 RUB | `a8533cef-3379-493f-bd98-975c3d04cbde` / `9e67c299-e6a7-11f1-82c4-e46f13aa4449` |
| Тест отказа сайта 73 — rejected attempt | same Apr18–20 stay; no confirmed booking | `14d4fb75-4f08-47ef-bc58-0cd19c3ced76` |

Validation: `node scripts/qa/hotel-booking-harness.mjs`, `npm run check` and production
`npm run build` pass. Backend formatting/build/vet/full serial suite pass; booking coverage
95.5%. The example's documented disabled booking default was restored; active runtime
booking remains enabled. Existing dynamic-rendering build diagnostics remain. Next.js
16.4.2 also reports a known security vulnerability; its upgrade is a separate task.

Production release still requires the intended secure 1C transport, verified production
clock-time copy, stable cookie secret, appropriate trusted proxy/IP attribution, operational
backlog monitoring and real recipient mailbox checks. These development tests do not establish
production deployment or repeat the independently accepted native 1C concurrency matrix.

## Integration corrections verified during acceptance

1C's HTTP response begins with a UTF-8 BOM (`EF BB BF`). The original Go client rejected
it and mapped availability to public quote 503. The client now accepts one leading BOM only
at the bounded HTTP response boundary, preserving strict JSON/status/stay checks. The owner
restarted Go and the public quote returned 200. Both publication URL variants work; no URL
or 1C reference change was needed. Backend fix commit: `2b9829e`.

Go serializes quote expiry as RFC3339 with its Moscow `+03:00` offset. The website validator
now accepts RFC3339 offsets as well as `Z`; a regression covers the actual timestamp precision.
Guest validation messages, including malformed/out-of-range URL occupancy, are Russian.
