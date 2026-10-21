# Task 08: Test utilities

## Goal
Helpers that every later test uses: a clean migrated database, row builders, and a JSON request helper.

## Depends on
07.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/store/store.go`, `internal/store/migrate.go`.

## Files to create
- `internal/testutil/db.go`
- `internal/testutil/builders.go`
- `internal/testutil/http.go`
- `internal/testutil/testutil_test.go`

`testutil` may import only the standard library, `sqlx`, and `internal/store`. It must never import a feature package (that would create import cycles).

## Spec

### `db.go`
```go
// NewDB returns a connection to the test database with all migrations applied and all tables empty.
// DSN: environment variable TEST_DB_DSN, default
// "postgres://postgres:postgres@127.0.0.1:5433/test?sslmode=disable".
// If the database cannot be reached the test fails with t.Fatalf and a hint to run `make db-test-up`.
func NewDB(t *testing.T) *sqlx.DB
```
Steps inside `NewDB`:
1. `store.Open`; register `t.Cleanup` to close it.
2. `store.Migrate`.
3. Empty every table and reset the id counters:
```sql
TRUNCATE users, addresses, user_addresses, categories, category_images, products, product_images,
         tags, product_tags, delivery_prices, orders, order_items RESTART IDENTITY CASCADE
```

### `builders.go`
Each builder inserts one row with plain SQL, fails the test with `t.Fatalf` on error, and returns the new id. Call `t.Helper()` first.

```go
// InsertUser creates a user with a NULL email and empty password. role is "user" or "admin".
func InsertUser(t *testing.T, db *sqlx.DB, name, phone, role string) int64

type Category struct {
    Name      string // default "Категория" when empty
    SortOrder int
    IsDeleted bool
    SaleStart *int   // minutes from midnight
    SaleEnd   *int
    SaleDays  string // "" or 7 chars of 0/1
}
func InsertCategory(t *testing.T, db *sqlx.DB, c Category) int64

type Product struct {
    CategoryID   int64
    Name         string // default "Продукт" when empty
    PriceKopecks int64
    SortOrder    int
    IsDeleted    bool
    SaleStart    *int
    SaleEnd      *int
    SaleDays     string
}
func InsertProduct(t *testing.T, db *sqlx.DB, p Product) int64

func InsertDeliveryPrice(t *testing.T, db *sqlx.DB, name string, priceKopecks int64) int64

// InsertAddress creates an address. When userID is not 0 it also links it in user_addresses.
func InsertAddress(t *testing.T, db *sqlx.DB, userID int64, city, address string) int64

// Count runs a "SELECT count(*) ..." query and returns the number.
func Count(t *testing.T, db *sqlx.DB, query string, args ...any) int

func IntPtr(v int) *int
```

### `http.go`
```go
// DoJSON sends a request to h and returns the recorder.
// body == nil sends no body; otherwise body is JSON-encoded and Content-Type is application/json.
// token != "" adds the header "Authorization: Bearer <token>".
func DoJSON(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder

// Decode JSON-decodes the recorder body into dst and fails the test on error.
func Decode(t *testing.T, rec *httptest.ResponseRecorder, dst any)
```

## Tests to write
`testutil_test.go`:
- `TestNewDBIsEmpty`: insert a user, call `NewDB` again, `Count(... "SELECT count(*) FROM users")` is 0, and the next inserted user gets id 1.
- `TestBuilders`: create a user, a category with `SaleStart: IntPtr(540)`, a product in it, a delivery price, an address linked to the user; check with `Count` that each table has one row and `user_addresses` has one row.
- `TestDoJSON`: a handler that echoes the method, the `Authorization` header and the body; check all three, then check that `Decode` fills a struct.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Builders for orders (task 25 inserts orders through its own store).
