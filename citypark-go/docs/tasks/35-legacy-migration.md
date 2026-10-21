# Task 35: Legacy data migration command

## Goal
A one-off command that copies users, addresses, delivery prices and orders from the old MySQL database into the new PostgreSQL database. The menu is not copied: it is re-synced from iiko first.

## Depends on
31.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/design/schema.sql`, `internal/store/store.go`.

## Files to create
- `internal/legacy/migrate.go`
- `internal/legacy/source.go`
- `internal/legacy/migrate_test.go`
- `internal/legacy/convert.go`
- `internal/legacy/convert_test.go`
- `cmd/migrate-legacy/main.go`

This is the only task that may add `github.com/go-sql-driver/mysql`. Import it only in `cmd/migrate-legacy/main.go`.

## Old schema (MySQL)
```
user(id, created_at, updated_at, name, email NULL, phone, password, is_email_confirmed, is_phone_confirmed)
role(id, role)                      -- role is 'user' or 'admin'
user_roles(user_id, role_id)
address(id, created_at, updated_at, city, address)
user_address(user_id, address_id)
delivery_price(id, created_at, updated_at, value NULL, name, price FLOAT)
product(id, iiko_id VARCHAR(36) NULL, ...)
`order`(id, created_at, updated_at, customer_id, address_id NULL, status, comment NULL,
        payment_type, items_hash, payment_id)
order_item(order_id, product_id, amount, offered_price FLOAT)
```
`order` is a reserved word in MySQL: write it with backticks. `user.password` is the legacy hash (64 hex chars) or `''` for customers who never registered. Old timestamps have no time zone; treat them as `Europe/Moscow`.

## Spec

### `convert.go` — pure functions
```go
package legacy

// Role returns "admin" when roles contains "admin", else "user".
func Role(roles []string) string

// Kopecks converts a float ruble amount to kopecks (rounded).
func Kopecks(rubles float64) int64

// Phone normalises with validate.NormalizePhone; when that fails it returns the input unchanged.
func Phone(s string) string

// Email lower-cases and trims; "" becomes nil.
func Email(s *string) *string

// Status returns s when it is one of new, pending, paid, canceled, confirmed; otherwise "new".
func Status(s string) string

// PaymentType returns "online" for "online", otherwise "cash".
func PaymentType(s string) string
```

### `migrate.go`
```go
type Report struct {
    Users, Addresses, UserAddresses, DeliveryPrices, Orders, OrderItems int
    SkippedOrderItems int // items whose product does not exist in the new database
    SkippedUsers      int // users whose normalised phone or email collides with an earlier user
}

// Source reads the old data. MySQLSource implements it; tests use an in-memory fake.
type Source interface {
    Users(ctx context.Context) ([]OldUser, error)               // with Roles []string filled
    Addresses(ctx context.Context) ([]OldAddress, error)
    UserAddresses(ctx context.Context) ([]OldUserAddress, error)
    DeliveryPrices(ctx context.Context) ([]OldDeliveryPrice, error)
    Products(ctx context.Context) ([]OldProduct, error)         // ID and IikoID only
    Orders(ctx context.Context) ([]OldOrder, error)
    OrderItems(ctx context.Context) ([]OldOrderItem, error)
}

// NewMySQLSource wraps the old database. It contains only SELECT statements and row scanning.
func NewMySQLSource(db *sql.DB) Source

// Run copies the data from src into dst, the new PostgreSQL database.
// Everything is written in one PostgreSQL transaction.
func Run(ctx context.Context, src Source, dst *sqlx.DB, log *slog.Logger) (Report, error)
```
The `Old*` structs are plain structs with one field per column of the old tables listed above (nullable columns as pointers). Put them and `MySQLSource` in `internal/legacy/source.go`. All decisions (mapping, skipping, merging, converting) live in `Run` and `convert.go`, never in `MySQLSource`, so that they are testable without MySQL.
Steps, all inside `store.InTx` on `dst`:
1. **Guard.** If `users` in `dst` has any row other than admins, or `orders` has any row, return the error `legacy: destination is not empty`.
2. **Users.** Read users with their roles. Insert with the **old id**, `password_hash = ''`, `legacy_password_hash = old password`, role from `Role`, phone from `Phone`, email from `Email`. If the phone or email already exists in `dst` (for example the bootstrap admin), do not insert: map the old id to the existing row's id and count it in `SkippedUsers`. Keep a map `oldUserID → newUserID`.
3. **Addresses.** Insert with `INSERT ... ON CONFLICT (city, address) DO UPDATE SET updated_at = addresses.updated_at RETURNING id` (the new table has a unique constraint; the old one had duplicates). Keep a map `oldAddressID → newAddressID`.
4. **User addresses.** Insert through both maps with `ON CONFLICT DO NOTHING`.
5. **Delivery prices.** Copy with `Kopecks(price)`; `value` NULL → `''`.
6. **Product map.** Read `id, iiko_id` from old `product`; read `id, iiko_id` from new `products`; build `oldProductID → newProductID` through `iiko_id`.
7. **Orders.** Insert with the **old id**, mapped customer and address, `Status`, `PaymentType`, `comment` NULL → `''`, `delivery_price_kopecks = 0`.
8. **Order items.** Insert with the mapped product id and `Kopecks(offered_price)`. If the product has no mapping, skip the item, count it in `SkippedOrderItems` and log a warning with the order id and old product id. If two old items of one order map to the same product, add their amounts.
9. **Sequences.** Explicit ids do not advance identity sequences. For `users`, `addresses`, `delivery_prices`, `orders` run:
```sql
SELECT setval(pg_get_serial_sequence('users', 'id'), COALESCE((SELECT max(id) FROM users), 0) + 1, false)
```

Timestamps: read `created_at` / `updated_at` as strings or `time.Time` and write them as times in `Europe/Moscow`. In the MySQL DSN use `parseTime=true&loc=Europe%2FMoscow`.

### `cmd/migrate-legacy/main.go`
Environment: `LEGACY_MYSQL_DSN` (for example `root:@tcp(localhost:3306)/citypark_prod?parseTime=true&loc=Europe%2FMoscow`) and `DATABASE_URL`. Both are required. Open both, run `store.Migrate` on the destination, call `legacy.Run`, print the report. Exit 1 on error.

Print this notice before starting:
```
Run the iiko sync on the new database first (POST /api/v1/iiko/update-products); order items are matched to products by iiko id.
```

## Tests to write
A MySQL server is not available in tests. `Run` is tested with a fake `Source` (a struct holding slices) against `testutil.NewDB(t)`; only `MySQLSource` itself is untested, so the 80% coverage rule applies to `migrate.go` and `convert.go`, not to `source.go` (check with `go tool cover -func`).
- `TestRole`, `TestKopecks` (5.99 → 599, 320 → 32000), `TestPhone` (`89991234567` → `+79991234567`; `abc` → `abc`), `TestEmail`, `TestStatus`, `TestPaymentType`: table tests.
- `TestSequenceReset`: with `testutil.NewDB(t)`, insert a user with explicit id 50 using plain SQL, run the `setval` statement for `users`, insert another user without an id → it gets id 51.

`migrate_test.go`, each with a fake `Source`. Seed the new database with a category and two products that have `iiko_id` values `p-1` and `p-2`.
- `TestRunCopiesEverything`: two users (one with roles `user` + `admin`), two addresses, one link, two delivery prices, one order with two items → the report counts match; users keep their old ids, have `password_hash = ''` and the old hash in `legacy_password_hash`; the admin has role `admin`; a delivery price of `150.5` is stored as `15050`; the order has the mapped customer and address; item prices are in kopecks.
- `TestRunExistingAdmin`: the destination already has an admin with the same phone as an old user → that user is not inserted, `SkippedUsers == 1`, and the old user's order points to the existing admin row.
- `TestRunDuplicateAddresses`: two old addresses with the same city and text → one new row, and both old ids map to it (two orders end up with the same `address_id`).
- `TestRunUnknownProduct`: an order item whose old product has no `iiko_id` match → the item is skipped, `SkippedOrderItems == 1`, the order itself is copied.
- `TestRunMergesItems`: two old products with the same `iiko_id` in one order, amounts 1 and 2 → one item with amount 3.
- `TestRunNormalises`: phone `89991234567` → `+79991234567`; email `A@B.io` → `a@b.io`; empty email → NULL; status `weird` → `new`; NULL comment → `''`.
- `TestRunDestinationNotEmpty`: an order already exists in the destination → the error `legacy: destination is not empty` and nothing is written.
- `TestRunRollsBack`: the fake source returns an error from `Orders` → `Run` returns it and the destination has no copied users.
- `TestRunSequences`: after `Run` with a user of old id 50, a user inserted without an id gets id 51.

## Done when
The commands in CONVENTIONS "Done means" pass, and `go build ./cmd/migrate-legacy` succeeds.

## Out of scope
Copying categories, products, images and tags. Running the command against production; that is a manual step for the project owner, after a database backup.
