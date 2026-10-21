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
| `TELEGRAM_BASE_URL` | `https://api.telegram.org` | Telegram Bot API base URL |
| `TELEGRAM_TOKEN` | empty | Bot token |
| `TELEGRAM_CHAT_ID_FOR_ORDER_CREATED` | empty | Chat that receives new orders |
| `TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED` | empty | Chat that receives paid orders |
| `MONETA_ID` | empty | Moneta shop id (`MNT_ID`) |
| `MONETA_SECRET` | empty | Moneta integrity code |
| `MONETA_TEST_MODE` | `false` | `true` marks payments as test payments |
| `WORKING_HOURS_START` | `9:00` | Orders are accepted from this time |
| `WORKING_HOURS_END` | `22:00` | Orders are accepted until this time |
| `THANK_YOU_PAGE_PAYMENT_CASH` | `http://localhost:3001/thankyou` | Frontend page shown after a cash order |

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
- **Moneta**: in the shop settings set "Pay URL" to `/api/v1/webhooks/moneta` (full public URL) with method POST. The integrity code in the Moneta cabinet must equal `MONETA_SECRET`. The frontend must use the same shop id as `MONETA_ID` and the same test mode as `MONETA_TEST_MODE`.
- **Telegram**: add the bot to the staff chats and put the chat ids in `TELEGRAM_CHAT_ID_FOR_ORDER_CREATED` and `TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED`.

## Deployment

```
docker build -t citypark-go .
```

The image sets `APP_ENV=production`, so the server refuses to start with the default `JWT_SECRET`. Set at least `JWT_SECRET`, `DATABASE_URL`, the `ADMIN_*` variables and the iiko, Telegram and Moneta secrets. Migrations are embedded in the binary and run automatically at startup.

To run the app with Compose: `docker compose --profile app up --build`.
