# Task 33: Dockerfile, env example, README

## Goal
Make the service deployable and documented.

## Depends on
31.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/config/config.go`, `Makefile`, `docker-compose.yml`.

## Files to create
- `Dockerfile`
- `.dockerignore`
- `.env.example`
- `README.md`
- `internal/config/envexample_test.go`

## Files to change
- `docker-compose.yml`: add an `app` service.
- `Makefile`: load `.env` for `run` and `seed` when the file exists.

## Spec

### `Dockerfile`
Two stages.
1. Build: `golang:1.24-alpine` (or the Go version in `go.mod` if it is newer). Copy `go.mod` and `go.sum`, run `go mod download`, copy the rest, build with
   `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server`.
2. Run: `alpine:3.20`. `apk add --no-cache ca-certificates tzdata`. Copy the binary to `/app/server`. `ENV APP_ENV=production`. `EXPOSE 3000`. Run as the unprivileged user `nobody`. `ENTRYPOINT ["/app/server"]`.

`tzdata` is required: the service loads `Europe/Moscow` at startup. Migrations are embedded in the binary, so no SQL files are copied.

### `.dockerignore`
```
.git
bin
.env
docs
*.md
```

### `.env.example`
One line per variable read by `internal/config`, in the order of the table in `config.go`, grouped with comment lines. Use safe placeholders:
- `APP_ENV=development`, `PORT=3000`, `DEBUG=false`, `APP_TIMEZONE=Europe/Moscow`, `DEFAULT_PAGE_SIZE=100`
- `DATABASE_URL=postgres://postgres:postgres@localhost:5432/citypark_dev?sslmode=disable`
- `JWT_SECRET=changeme`, `JWT_EXPIRES_IN=24h`
- `ADMIN_EMAIL=dev@dev.io`, `ADMIN_PASSWORD=devdevdev`, `ADMIN_PHONE=+79990000000`
- every iiko and Telegram variable, with the default base URLs and empty secrets
- `MONETA_ID=`, `MONETA_SECRET=`, `MONETA_TEST_MODE=true`
- `IIKO_CATEGORY_MARKER='#'` — the single quotes are required: an unquoted `#` starts a comment when the file is loaded by a shell
- `WORKING_HOURS_START=9:00`, `WORKING_HOURS_END=22:00`
- `THANK_YOU_PAGE_PAYMENT_CASH=http://localhost:3001/thankyou`

### `docker-compose.yml`
Add:
```yaml
  app:
    build: .
    profiles: ["app"]
    env_file: .env
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/citypark_dev?sslmode=disable
    ports:
      - "3000:3000"
    depends_on:
      - postgres
```
The profile keeps `docker compose up -d postgres` and `make db-test-up` unchanged; the app starts with `docker compose --profile app up --build`.

### `Makefile`
Do not use `include .env` (make would treat the `#` marker value as a comment). Change the two recipes to load the file in the shell:
```make
run:
	set -a; [ ! -f .env ] || . ./.env; set +a; go run ./cmd/server

seed:
	set -a; [ ! -f .env ] || . ./.env; set +a; go run ./cmd/seed
```

### `README.md`
Sections, short and factual:
1. What the service is (two sentences).
2. Requirements: Go, Docker.
3. Quick start: `cp .env.example .env`, `make db-up`, `make seed`, `make run`, then `curl localhost:3000/api/v1/health`.
4. Tests: `make test` (starts the test database on port 5433, runs `go test -p 1 ./...`).
5. Configuration: a table of every variable with its default and meaning.
6. Project layout: one line per `internal/` package.
7. API: one line pointing frontend developers to `docs/API.md`.
8. External services: iiko sync (`POST /api/v1/iiko/update-products` with header `X-Update-Token`), Moneta (set the shop's "Pay URL" to `/api/v1/webhooks/moneta` with method POST; the integrity code in the cabinet must equal `MONETA_SECRET`; the frontend needs the same shop id as `MONETA_ID` and the same test mode), Telegram chats.
9. Deployment: `docker build`, the variables that must be set in production (`JWT_SECRET`, `DATABASE_URL`, admin and integration secrets), and that migrations run automatically at startup.

### `internal/config/envexample_test.go`
`TestEnvExampleIsComplete`: read `../../.env.example`, collect the variable names and values (lines of the form `NAME=value`, ignoring comments and blank lines, and removing one pair of surrounding single or double quotes from the value), and call `Load` with a lookup function that records every key it is asked for. The two sets must be equal: every variable the config reads is in the example and the example has no unknown variable. `Load` must also succeed with the example's values.

## Done when
The commands in CONVENTIONS "Done means" pass. If Docker is available, `docker build -t citypark-go .` succeeds.

## Out of scope
CI configuration, Kubernetes, reverse proxy setup.
