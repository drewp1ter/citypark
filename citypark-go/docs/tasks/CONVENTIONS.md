# Conventions

Read this at the start of every task. Follow it exactly; later tasks rely on it.

## Layout

```
cmd/server/            main package
internal/<pkg>/        one package per feature
migrations/            goose SQL files, embedded
docs/tasks/            task files and PROGRESS.md
```

Module path: `github.com/drewp1ter/citypark-go`. A feature package has `store.go` (SQL), `service.go` (only when there is logic beyond SQL), `handler.go` (HTTP), and `*_test.go`.

## Allowed dependencies

`github.com/go-chi/chi/v5`, `github.com/jmoiron/sqlx`, `github.com/jackc/pgx/v5`, `github.com/pressly/goose/v3`, `github.com/golang-jwt/jwt/v5`, `github.com/go-playground/validator/v10`, `golang.org/x/crypto/bcrypt`, `github.com/microcosm-cc/bluemonday`. Add nothing else unless the task says so. Use the standard library for HTTP clients, logging (`log/slog`) and tests (no testify).

## Code rules

- No package-level mutable state. Dependencies go through constructors: `NewStore(db)`, `NewHandler(...)`.
- Every function that does I/O takes `ctx context.Context` first.
- Current time comes from a `clock.Clock` (task 09), never from `time.Now()` in feature code.
- Money is `money.Kopecks` (task 09) everywhere in Go and `*_kopecks bigint` in the database. JSON shows rubles (`599` or `5.99`).
- Text shown to API users is in Russian. Log messages and Go errors are in English.
- Keep exported names exactly as the task spells them. Other tasks call them.

## HTTP rules (package `httpx`, task 03)

- Handlers have the signature `func(w http.ResponseWriter, r *http.Request) error` and are wrapped with `httpx.Handle`.
- Return `*httpx.Error` for expected failures. Any other error becomes a 500.
- Error body: `{"statusCode": 400, "message": "...", "errors": {"field": "message"}}`. `errors` is omitted when empty.
- Each handler type has `Routes(r chi.Router)`. Paths in a task are relative to `/api/v1`; `Routes` registers them without that prefix.
- Request bodies are decoded with `httpx.DecodeJSON` and checked with `validate.Struct` (task 05).
- JSON field names are camelCase. Timestamps are RFC 3339.

## API documentation

`docs/API.md` is the reference for frontend developers. A task that adds or changes an endpoint updates it in the same commit:

- One section per endpoint, in the format shown at the top of that file, under its area heading.
- Each section has: method and path, role, parameters and request fields with their rules, a request example, the success status with a response example, every error status with its exact message, and notes the frontend needs.
- Examples must match what the server really returns. Copy them from test output; do not invent field names.
- When a later task changes an endpoint, edit its existing section. Do not add a second one.

## SQL rules (PostgreSQL 17, sqlx over pgx)

- A store is `type Store struct{ db store.DBTX }` with `NewStore(db store.DBTX)`. `*sqlx.DB` and `*sqlx.Tx` both satisfy `store.DBTX`, so inside a transaction build a new store from the `tx`.
- Placeholders are `$1, $2, ...`. Never build SQL by concatenating values.
- Get generated keys with `INSERT ... RETURNING`. Never use `LastInsertId`.
- Upsert with `INSERT ... ON CONFLICT (...) DO UPDATE SET ...`.
- For lists write `col = ANY($1)` and pass a Go slice (`[]int64`, `[]string`) directly. Do not use `sqlx.In` or `pq.Array`.
- Never pass a nil slice as a parameter: it becomes SQL `NULL` and the condition matches nothing. Use `[]string{}` or `make([]int64, 0)`.
- Every `UPDATE` also sets `updated_at = now()` when the table has that column.
- "No rows" is returned as `store.ErrNotFound`. Constraint errors are detected with `store.IsUniqueViolation`, `store.IsFKViolation`, `store.IsCheckViolation`.
- Struct fields are mapped with `db:"column_name"` tags. Nullable columns scan into pointers (`*string`, `*int`).

## Test rules

- Database tests call `testutil.NewDB(t)` (task 08). It returns a migrated, empty database. The DSN comes from `TEST_DB_DSN`, default `postgres://postgres:postgres@127.0.0.1:5433/test?sslmode=disable`. Start it with `make db-test-up`.
- All packages share that one database, so always run tests with `-p 1` and never call `t.Parallel()` in a test that uses the database.
- HTTP handlers are tested through a `chi.NewRouter()` with the handler's `Routes`, using `testutil.DoJSON`.
- External services (iiko, Telegram) are faked with `httptest.NewServer`; clients take a base URL. Moneta needs no fake: the backend never calls it, it only computes and checks signatures.
- Use table-driven tests for pure logic.
- All logic is covered by tests. Logic is any code with a branch, a loop, a calculation, a parsing or formatting step, an SQL statement, or an error path. Write every test the task lists; if you write logic that those tests do not reach, add a test for it.
- A test checks a result: a returned value, a database row, a response status and body. A test that only calls the code is not enough.
- Never delete, skip or loosen a test to make it pass. Change a test only when it contradicts the task file.
- Statement coverage of every `internal/` package you change must be at least 80% (`cmd/` is exempt).

## Done means

Every task ends with these passing, plus whatever the task adds:

```
gofmt -l .            # prints nothing
go build ./...
go vet ./...
go test -p 1 ./...
go test -p 1 -cover ./internal/<each package you changed>/...   # each at least 80%
```

If the task added or changed an endpoint, `docs/API.md` has an up-to-date section for it.

Then tick the task in `docs/tasks/PROGRESS.md` and commit as `task NN: <title>`.
