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
| `TELEGRAM_TOKEN` | empty | Bot token |
| `TELEGRAM_STAFF_CHAT_ID` | empty | Staff chat for paid orders and iiko alerts |
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
- **Orders in iiko**: when Moneta confirms a payment, the order is created in iiko in the background (up to 3 attempts, 30 s and 2 min apart) and its iiko id is saved in `orders.iiko_order_id`. If iiko does not create the order, the `TELEGRAM_STAFF_CHAT_ID` chat gets an alert and staff enter the order in iikoFront by hand. The iiko address is a fixed placeholder (`IIKO_ADDRESS_*`); the real address, the delivery time and the delivery fee are written in the comment. Get the ids with a token from `v2/access_token`: `POST /api/1/terminal_groups` and `POST /api/1/payment_types` with `{"organizationIds": ["<IIKO_ORGANIZATION_ID>"]}`; take the payment type the restaurant uses for online payments, and put its `paymentTypeKind` in `IIKO_PAYMENT_TYPE_KIND`.
- **Moneta**: in the shop settings set "Pay URL" to `/api/v1/webhooks/moneta` (full public URL) with method GET or POST. Set "Check URL" to the full public URL ending in `/api/v1/webhooks/moneta/check`. Enable mandatory payment-form signatures. The integrity code in the Moneta cabinet must equal `MONETA_SECRET`. The server builds the payment URL (`MONETA_URL`: demo or production assistant). Turn on «Можно переопределять настройки в URL» in the shop settings so `MONETA_SUCCESS_URL` and `MONETA_FAIL_URL` are used. Success/Fail redirects do not confirm payment; only a verified Pay URL notification does. Replace the old `https://assistant.demo.moneta.ru/` URL in local environment files with `https://demo.moneta.ru/assistant.htm`. Keep account test mode consistent with `MONETA_TEST_MODE`. The demo and production shops have different `MONETA_ID`.
- **Telegram**: add the bot to the staff chat and set `TELEGRAM_STAFF_CHAT_ID` to its id. Before deploying, replace the two old chat variables with this one and restart the backend. A paid order sends its details followed by a separate phone message.

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

It first calls the signed Check URL to obtain the order total including delivery, then posts the signed Pay URL notification. This can mark the order paid and trigger iiko and Telegram processing; it does not transfer money.

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
