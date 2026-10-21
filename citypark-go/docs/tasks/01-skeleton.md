# Task 01: Repo skeleton and health endpoint

## Goal
Create the Go module, a Makefile, a docker-compose file with two PostgreSQL databases, and a server that answers `GET /api/v1/health`.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`.

## Files to create
- `go.mod` (run `go mod init github.com/drewp1ter/citypark-go`, then `go get github.com/go-chi/chi/v5`)
- `.gitignore`
- `Makefile`
- `docker-compose.yml`
- `cmd/server/main.go`
- `internal/health/handler.go`
- `internal/health/handler_test.go`

## Spec

### `.gitignore`
```
/bin/
.env
*.log
```

### `docker-compose.yml`
Two services, both `image: postgres:17`:

| Service | Host port | Database | Storage |
|---|---|---|---|
| `postgres` | 5432 | `citypark_dev` | named volume `pgdata` |
| `postgres_test` | 5433 | `test` | `tmpfs: /var/lib/postgresql/data` |

Both set `POSTGRES_USER=postgres`, `POSTGRES_PASSWORD=postgres` and `POSTGRES_DB` to the database name. The container port is 5432 in both.

### `Makefile`
Targets (all `.PHONY`):

| Target | Command |
|---|---|
| `build` | `go build -o bin/server ./cmd/server` |
| `run` | `go run ./cmd/server` |
| `db-up` | `docker compose up -d postgres` |
| `db-test-up` | `docker compose up -d --wait postgres_test` |
| `test` | depends on `db-test-up`; runs `go test -p 1 ./...` |
| `lint` | `gofmt -l . && go vet ./...` |

Add a `healthcheck` to `postgres_test` (`pg_isready -U postgres`, interval 1s, retries 30) so that `--wait` works.

### `internal/health/handler.go`
```go
package health

// Routes registers GET /health.
func Routes(r chi.Router)
```
`GET /health` answers 200 with `Content-Type: application/json` and body `{"status":"ok"}`. Write the JSON with `encoding/json` directly; the `httpx` package does not exist yet.

### `cmd/server/main.go`
- Build a `chi.NewRouter()`.
- Mount everything under `/api/v1` with `r.Route("/api/v1", func(r chi.Router) { health.Routes(r) })`.
- Listen on the port from the `PORT` environment variable, default `3000`.
- Use `http.Server` with `ReadHeaderTimeout: 10 * time.Second`. Exit with a non-zero status when `ListenAndServe` fails.

Keep `main.go` short. Task 31 replaces it.

## API documentation
Add the area heading `### Health` and the section `#### GET /health` to `docs/API.md` (role public, response 200 `{"status":"ok"}`, no errors).

## Tests to write
`internal/health/handler_test.go`:
- `TestHealth`: build a chi router with `Routes`, send `GET /health` through `httptest.NewRecorder`, expect status 200 and body JSON equal to `{"status":"ok"}`.

## Done when
```
gofmt -l .
go build ./...
go vet ./...
go test -p 1 ./...
```
`gofmt -l .` must print nothing.

## Out of scope
Configuration loading, logging, database access.
