# Task 06: Store — database connection and transactions

## Goal
A small package that opens PostgreSQL through sqlx + pgx, runs functions in a transaction, and classifies database errors.

## Depends on
01.

## Read first
`docs/tasks/CONVENTIONS.md`.

## Files to create
- `internal/store/store.go`
- `internal/store/errors.go`
- `internal/store/store_test.go`

Run `go get github.com/jmoiron/sqlx github.com/jackc/pgx/v5`.

## Spec

```go
package store

import _ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

// DBTX is implemented by *sqlx.DB and *sqlx.Tx.
type DBTX interface {
    GetContext(ctx context.Context, dest any, query string, args ...any) error
    SelectContext(ctx context.Context, dest any, query string, args ...any) error
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
    QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
}

// Open connects with sqlx.Open("pgx", dsn), pings with ctx and
// sets MaxOpenConns 10, MaxIdleConns 5, ConnMaxLifetime 30 minutes.
// On a failed ping it closes the pool and returns the error.
func Open(ctx context.Context, dsn string) (*sqlx.DB, error)

// InTx runs fn in a transaction.
// fn returns nil  -> commit
// fn returns err  -> rollback, return err
// fn panics       -> rollback, panic again
func InTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error
```

`errors.go`:
```go
// ErrNotFound is returned by stores when a row does not exist.
var ErrNotFound = errors.New("not found")

// NotFound converts sql.ErrNoRows to ErrNotFound and returns every other error unchanged.
// Stores wrap single-row reads with it: return store.NotFound(err).
func NotFound(err error) error

func IsUniqueViolation(err error) bool // PostgreSQL code 23505
func IsFKViolation(err error) bool     // PostgreSQL code 23503
func IsCheckViolation(err error) bool  // PostgreSQL code 23514
```
Implement the three checks with `errors.As(err, &pgErr)` where `pgErr` is `*pgconn.PgError` (`github.com/jackc/pgx/v5/pgconn`) and compare `pgErr.Code`.

Add compile-time assertions in `store.go`:
```go
var (
    _ DBTX = (*sqlx.DB)(nil)
    _ DBTX = (*sqlx.Tx)(nil)
)
```

## Tests to write
The tests need the test database. Start it with `make db-test-up`. Read the DSN from `TEST_DB_DSN`, default `postgres://postgres:postgres@127.0.0.1:5433/test?sslmode=disable`. Put a helper `openTestDB(t)` in the test file that calls `Open` and registers `t.Cleanup(db.Close)`.

Each test creates what it needs with `CREATE TEMP TABLE`. A temp table belongs to one connection, so call `db.SetMaxOpenConns(1)` in the helper.

- `TestOpenBadDSN`: `Open` with port 1 returns an error.
- `TestInTxCommit`: insert inside `InTx`, return nil, the row is visible afterwards.
- `TestInTxRollback`: insert inside `InTx`, return an error, the row is gone and the same error is returned.
- `TestInTxPanic`: `fn` panics; the panic reaches the test (use `recover`) and the row is gone.
- `TestNotFound`: selecting a missing row with `GetContext` and wrapping with `NotFound` gives `errors.Is(err, ErrNotFound)`.
- `TestViolations`: with a temp table `t (id int primary key, n int check (n > 0))`: a duplicate id gives `IsUniqueViolation`, `n = 0` gives `IsCheckViolation`. For the FK case create a second temp table that references the first and insert a missing reference; expect `IsFKViolation`. Each helper returns false for `nil` and for `errors.New("x")`.

## Done when
`make db-test-up` has been run, and the commands in CONVENTIONS "Done means" pass.

## Out of scope
Migrations (task 07). Do not create application tables here.
