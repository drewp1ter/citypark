# citypark-go

Backend of a restaurant ordering service. The menu is synced from the iiko POS, online payments go through MONETA.RU, and staff are notified about orders through Telegram.

## Requirements

- Go (the version in `go.mod`)
- Docker with Compose

## Quick start

```
cp .env.example .env
make db-up
make seed
make run
curl localhost:3000/api/v1/health
```

## Tests

```
make test
```

This starts the test database on port 5433 and runs `go test -p 1 ./...`. Packages share one database, so tests must not run in parallel across packages.

## Configuration

All settings are environment variables. `make run` and `make seed` load `.env` when it exists.

| Variable | Default | Meaning |
| --- | --- | --- |
| `APP_ENV` | `development` | `development`, `production` or `test` |
| `PORT` | `3000` | HTTP port |
| `DEBUG` | `false` | `true` enables debug logging |
| `APP_TIMEZONE` | `Europe/Moscow` | Time zone for working hours and sale windows |
| `DEFAULT_PAGE_SIZE` | `100` | Page size when a list request gives none |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/citypark_dev?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | `changeme` | JWT signing key; must be changed in production |
| `JWT_EXPIRES_IN` | `24h` | Token lifetime (Go duration) |
| `ADMIN_EMAIL` | `dev@dev.io` | Admin account email |
| `ADMIN_PASSWORD` | `devdevdev` | Admin account password |
| `ADMIN_PHONE` | `+79990000000` | Admin account phone |
| `IIKO_BASE_URL` | `https://api-ru.iiko.services/api` | iiko API base URL (endpoints carry their own version) |
| `IIKO_API_KEY` | empty | iiko API key |
| `IIKO_APP_ID` | empty | iiko application id |
| `IIKO_CLIENT_SECRET` | empty | iiko client secret |
| `IIKO_ORGANIZATION_ID` | empty | iiko organization id |
| `IIKO_EXTERNAL_MENU_ID` | empty | iiko external menu id |
| `IIKO_UPDATE_TOKEN` | empty | Secret for the `X-Update-Token` header of the sync endpoint |
| `IIKO_TERMINAL_GROUP_ID` | empty | iiko terminal group that receives paid orders; empty = orders are not sent |
| `IIKO_PROMO_DISCOUNT_TYPE_ID` | empty | Optional nonzero UUID of the RMS FlexibleSum discount; missing blocks new promo orders |
| `IIKO_PAYMENT_TYPE_ID` | empty | iiko payment type for online payments; empty = orders are not sent |
| `IIKO_PAYMENT_TYPE_KIND` | `Card` | `paymentTypeKind` of that payment type |
| `IIKO_ADDRESS_CITY` | `Белореченск` | City of the placeholder address in iiko orders |
| `IIKO_ADDRESS_STREET` | `Гоголя` | Street of the placeholder address |
| `IIKO_ADDRESS_HOUSE` | `61` | House of the placeholder address |
| `TELEGRAM_BASE_URL` | `https://api.telegram.org` | Telegram Bot API base URL |
| `TELEGRAM_PROXY_URL` | empty | Telegram-only SOCKS5 proxy, e.g. `socks5h://user:password@proxy-host:1080`; proxy failure never falls back to direct access. Both schemes resolve the target through the proxy. Empty preserves environment proxy behavior. |
| `TELEGRAM_TOKEN` | empty | Bot token |
| `TELEGRAM_STAFF_CHAT_ID` | empty | Staff chat for paid orders and iiko alerts |
| `HOTEL_BOOKING_TRUSTED_PROXIES` | empty | Comma-separated trusted proxy CIDRs for verified booking client IP forwarding; empty ignores forwarded headers |
| `HOTEL_BOOKING_ENABLED` | `false` | Enable hotel booking integration; requires all booking credentials and SMTP settings |
| `ONEC_BOOKING_URL` | empty | Private service root ending `/hs/booking/v1`; HTTPS required in production |
| `ONEC_BOOKING_USER` | empty | Dedicated restricted booking service user; independent of reporting credentials |
| `ONEC_BOOKING_PASSWORD` | empty | Server-only booking password |
| `MONETA_ID` | empty | Moneta shop id (`MNT_ID`) |
| `MONETA_SECRET` | empty | Moneta integrity code |
| `MONETA_TEST_MODE` | `false` | `true` marks payments as test payments |
| `MONETA_URL` | `https://demo.moneta.ru/assistant.htm` | MONETA.Assistant address (production: `https://www.payanyway.ru/assistant.htm`) |
| `MONETA_SUCCESS_URL` | `http://localhost:3001/checkout/done` | `MNT_SUCCESS_URL`; empty = not sent |
| `MONETA_FAIL_URL` | `http://localhost:3001/checkout?payment=failed` | `MNT_FAIL_URL`; empty = not sent |

## Project layout

- `cmd/server`: the HTTP server
- `cmd/seed`: fills an empty development database with sample data
- `internal/address`: delivery addresses of users
- `internal/app`: wires stores, services, handlers and clients into one router
- `internal/auth`: password hashing and JWT
- `internal/catalog`: categories, products and images
- `internal/clock`: current time behind an interface
- `internal/config`: environment configuration
- `internal/deliveryprice`: delivery price per town
- `internal/health`: health endpoint
- `internal/httpx`: JSON responses, errors and handler wrapper
- `internal/iiko`: iiko client and menu sync
- `internal/logging`: logger and HTTP middlewares
- `internal/moneta`: Moneta signature computing and checking
- `internal/money`: the `Kopecks` type
- `internal/order`: order store, service and endpoints, Moneta callback
- `internal/salewindow`: time and weekday limits of sales
- `internal/seed`: sample data for development
- `internal/store`: database connection, transactions and error helpers
- `internal/telegram`: Telegram client and order message
- `internal/testutil`: test helpers
- `internal/user`: user store and endpoints
- `internal/validate`: request validation

## API

Frontend developers: see [`docs/API.md`](docs/API.md).

## External services

- **iiko**: `POST /api/v1/iiko/update-products` with the header `X-Update-Token` equal to `IIKO_UPDATE_TOKEN` syncs categories, products and images.
- **Orders in iiko**: Moneta payment confirmation notifies staff and adds the order to the approval queue. Staff review the order in the admin panel, add kitchen comments or exclude stop-listed lines, then click «Отправить в IIKO». The worker saves confirmed IDs in `orders.iiko_order_id`. For a definite rejection, staff receive a Telegram alert. For an uncertain result, staff check iikoFront before creating anything manually. Deploy the backend and admin panel together and stop old backend instances before switching traffic to the approval workflow. The iiko address is a fixed placeholder (`IIKO_ADDRESS_*`); the real address, the delivery time and the delivery fee are written in the comment. Get the ids with a token from `v2/access_token`: `POST /api/1/terminal_groups` and `POST /api/1/payment_types` with `{"organizationIds": ["<IIKO_ORGANIZATION_ID>"]}`; take the payment type the restaurant uses for online payments, and put its `paymentTypeKind` in `IIKO_PAYMENT_TYPE_KIND`.
- **Moneta**: in the shop settings set "Pay URL" to `/api/v1/webhooks/moneta` (full public URL) with method GET or POST. Set "Check URL" to the full public URL ending in `/api/v1/webhooks/moneta/check`. Enable mandatory payment-form signatures. The integrity code in the Moneta cabinet must equal `MONETA_SECRET`. The server builds the payment URL (`MONETA_URL`: demo or production assistant). Turn on «Можно переопределять настройки в URL» in the shop settings so `MONETA_SUCCESS_URL` and `MONETA_FAIL_URL` are used. Success/Fail redirects do not confirm payment; only a verified Pay URL notification does. Replace the old `https://assistant.demo.moneta.ru/` URL in local environment files with `https://demo.moneta.ru/assistant.htm`. Keep account test mode consistent with `MONETA_TEST_MODE`. The demo and production shops have different `MONETA_ID`.
- **Telegram**: add the bot to the staff chat and set `TELEGRAM_STAFF_CHAT_ID` to its id. Before deploying, replace the two old chat variables with this one and restart the backend. A paid order sends its details followed by a separate phone message.

### Hotel booking client

The private [booking contract](../citypark-hotel-managment/docs/BOOKING_CONTRACT.md)
defines availability, creation and same-key recovery. Task 106 adds the client and
configuration only; public routes, pricing, email delivery and worker wiring follow
in task 107. Booking stays disabled by default. Enabling requires
`SMTP_HOST`, `SMTP_USERNAME`, `SMTP_PASSWORD` and a valid `SMTP_FROM` in addition
to the separate booking credentials. Existing `ONEC_HOTEL_URL`, `ONEC_USER` and
`ONEC_PASSWORD` remain exclusively for reporting and never provide booking defaults.
Use an isolated development base for HTTP tests; production requires HTTPS.

The client rejects redirects, limits responses to 64 KiB and bounds each request
to 15 seconds (or a shorter injected client/context deadline). It accepts one
leading UTF-8 BOM emitted by 1C's HTTP writer; all remaining JSON validation stays
strict, including errors and saved reservation results. Errors exclude
upstream bodies, URLs and credentials. `AmbiguousCreateError` means a reservation
may have committed: lookup or replay using the **original key and payload**;
never generate a replacement key. Recovery returns the saved creation result even
after a staff cancellation. No automatic create retry or live 1C request is needed
for client tests; they use local HTTP fakes.

## Deployment

```
docker build -t citypark-go .
```

The image sets `APP_ENV=production`, so the server refuses to start with the default `JWT_SECRET`. Set at least `JWT_SECRET`, `DATABASE_URL`, the `ADMIN_*` variables and the iiko, Telegram and Moneta secrets. Migrations are embedded in the binary and run automatically at startup.

To run the app with Compose: `docker compose --profile app up --build`.

Do all unchecked tasks:
CLAUDE_FLAGS="--dangerously-skip-permissions" scripts/run-tasks.sh
### Simulate a MONETA Pay URL notification

With the backend running, Python 3.9+ can send a signed notification for an existing order:

```bash
python3 scripts/simulate_moneta.py 16
```

The script reads literal values from the backend's `.env` (no shell expansion); exported environment variables override them. It requires `MONETA_ID` and `MONETA_SECRET`, with `MONETA_TEST_MODE` matching the running backend.

It first calls the signed Check URL to obtain the order total including delivery, then posts the signed Pay URL notification. This can mark the order paid, notify staff in Telegram, and make the order available for approval in the admin panel. The simulation transfers no money. Admin approval queues iiko submission.

Options:
- `--base-url http://localhost:3000`: backend origin.
- `--env-file .env.development`: choose a different credentials file.
- `--amount 790.00`: supply the amount manually and bypass Check URL.
- `--operation-id 555`: override the default simulated operation ID `simulated-16`.
- `--dry-run`: check the order and print its amount without sending the payment notification. With `--amount`, makes no HTTP requests.

No packages need installing. The simulator exits with an error on a failed request, invalid signed Check response, nonpayable order, or response other than `SUCCESS`. It does not follow HTTP redirects.

Run its isolated tests with `python3 -m unittest discover -s scripts -p 'test_*.py'`.


### Promo RMS deployment

Set `IIKO_PROMO_DISCOUNT_TYPE_ID` to a real discount returned by `POST /api/1/discounts` for the configured organization. The restaurant must configure assign amount (`FlexibleSum`) and selective application (`canBeAppliedSelectively=true`). Confirm activation, manual/nonautomatic use, no category/time/minimum-spend/automatic-discount/tax settings change the required result, and compatibility with the configured external payment type and terminal group (payment processing External or Both).

New promo orders require the discount ID plus terminal group and payment type settings; otherwise POST /orders returns 503 `Заказы с промокодом временно недоступны.` before customer/order writes or payment links. No-code orders and existing pending retries keep their existing behavior. An already-paid dispatch with missing/incompatible settings must alert staff; it never retries without the discount.

Keep campaigns disabled until an explicitly authorized deployment acceptance verifies both pickup and courier orders: original unit prices/quantities, all saved eligible positions (including lines with zero backend allocation), no ineligible positions, exact total RMS discount and exact discounted food payment. iiko may distribute that total differently across eligible lines. Delivery stays a paid-delivery comment; MONETA collects discounted food plus delivery. Check command completion and the resulting iikoFront order. See [the verified RMS contract](docs/tasks/IIKO_PROMOCODES.md). Automated tests use local fakes and do not send live orders, payments or Telegram messages.

## Sentry

Optional HTTP error and trace monitoring is documented in [docs/SENTRY.md](docs/SENTRY.md), including environment variables, privacy behavior, verification and manual deployment. Empty `SENTRY_DSN` disables reporting.

Production public-content export and restore: [docs/CONTENT_SEED.md](docs/CONTENT_SEED.md). This uses a separate command from the development demo seeder.

## Hotel booking API and workers

Task 107 exposes `/api/v1/hotel/quotes` and `/api/v1/hotel/reservations`; see
[the exact public contract](docs/API.md#hotel-booking). Booking remains disabled
by default. Migration 40 adds PostgreSQL quotes, attempts, rate counters and a
confirmation-email outbox. Migrations run through the existing startup path.
There is no automatic deployment or production activation.

When enabled, each server process starts a cancellation-aware worker. Pending
creation recovery runs every 30 seconds using the original persisted request/key,
without holding SQL transactions across HTTP. Row claims use one-minute leases,
`SKIP LOCKED` and fencing tokens. A worker crash is recoverable after lease expiry;
multiple processes can run safely. Confirmation and its unique outbox row commit
atomically. SMTP retries use 30 seconds, 1 minute, 5 minutes, then 30 minutes;
delivery is at-least-once. The worker reads the current hotel contact phone from
site settings. Missing contact configuration keeps email pending.

Configure `HOTEL_BOOKING_TRUSTED_PROXIES` with comma-separated CIDRs only for
trusted immediate proxies/website servers which securely build the forwarded IP
chain. Empty means all forwarded IP headers are ignored. This setting is separate
from reporting credentials and other API middleware. Quotes use fixed UTC minute
windows; new valid attempts use UTC hour windows. Review limits with the actual
website/proxy topology before release.

Monitor the following aggregate queries with restricted DB access; alert on
increasing age/backlog and repeated sanitized worker warnings. Do not log canonical
requests, results, context headers/cookies, keys, email addresses or phone numbers.

```sql
SELECT count(*) AS pending_count,
       extract(epoch FROM (now() - min(created_at))) AS oldest_pending_seconds
FROM hotel_booking_attempts WHERE state = 'pending';
SELECT count(*) AS email_backlog,
       max(attempts) AS max_delivery_attempts,
       extract(epoch FROM (now() - min(next_attempt_at))) AS oldest_due_seconds
FROM hotel_booking_email_outbox WHERE state = 'pending' AND next_attempt_at <= now();
```

Booking access logs redact paths and IPs; worker messages omit upstream/database/
SMTP error strings. Contacts and canonical JSON remain sensitive data in the
restricted database for recovery/email. Keep attempt/key records for the
reservation lifetime; there is no automatic expiry/reuse. Quotes referenced by
attempts must remain. Rate-counter rows may be pruned operationally after their
window ends; unused expired quotes may be pruned only when no attempt references
them. Do not delete pending attempts/outbox rows to clear alerts. Back up and
restore the booking tables together; inspect pending recovery after a restore.
Live 1C/SMTP and production acceptance are separate release checks; automated
backend tests use the configured test database and local fakes.
