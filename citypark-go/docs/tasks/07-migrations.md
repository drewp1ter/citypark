# Task 07: Migrations and schema

## Goal
Add the first goose migration (the full schema) and a function that applies migrations from files embedded in the binary.

## Depends on
06.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/store/store.go`.

## Files to create
- `migrations/00001_init.sql`
- `migrations/embed.go`
- `internal/store/migrate.go`
- `internal/store/migrate_test.go`

Run `go get github.com/pressly/goose/v3`.

## Spec

### `migrations/00001_init.sql`
Do not write the schema by hand. Build the file from `docs/design/schema.sql`:

```
-- +goose Up
<the complete content of docs/design/schema.sql, unchanged>

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;
DROP TABLE delivery_prices;
DROP TABLE product_tags;
DROP TABLE tags;
DROP TABLE product_images;
DROP TABLE products;
DROP TABLE category_images;
DROP TABLE categories;
DROP TABLE user_addresses;
DROP TABLE addresses;
DROP TABLE users;
```

A shell command that does it:
```
{ echo '-- +goose Up'; cat docs/design/schema.sql; echo; echo '-- +goose Down'; \
  for t in order_items orders delivery_prices product_tags tags product_images products \
           category_images categories user_addresses addresses users; do echo "DROP TABLE $t;"; done; \
} > migrations/00001_init.sql
```

### `migrations/embed.go`
```go
// Package migrations holds the goose SQL migrations.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

### `internal/store/migrate.go`
```go
// Migrate applies all pending migrations from the embedded files.
func Migrate(ctx context.Context, db *sqlx.DB) error

// MigrateDown rolls back every migration. Used by tests only.
func MigrateDown(ctx context.Context, db *sqlx.DB) error
```
Use the goose provider API, not the global functions:
```go
p, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
// Migrate:     _, err = p.Up(ctx)
// MigrateDown: _, err = p.DownTo(ctx, 0)
```

## Tests to write
`internal/store/migrate_test.go`, using the `openTestDB(t)` helper from `store_test.go`. These tests use real tables, so remove the `SetMaxOpenConns(1)` limit for them if it gets in the way (it does not have to).

- `TestMigrateUpDown`:
  1. `MigrateDown` (ignore the state before the test), then `Migrate`.
  2. Query `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'` and check that these 12 tables exist: `users, addresses, user_addresses, categories, category_images, products, product_images, tags, product_tags, delivery_prices, orders, order_items`.
  3. Call `Migrate` again: no error (idempotent).
  4. `MigrateDown`: the 12 tables are gone.
  5. `Migrate` again so that the database is left migrated for other packages.
- `TestSchemaConstraints` (after `Migrate`; delete the rows it creates at the end):
  - inserting two users with the same `phone` gives `IsUniqueViolation`;
  - inserting a user with `role = 'boss'` gives `IsCheckViolation`;
  - inserting a category with `sale_days = '11'` gives `IsCheckViolation`;
  - inserting a category named `Шашлыки` and running `SELECT count(*) FROM categories WHERE search @@ plainto_tsquery('russian', 'шашлык')` gives 1.

## Done when
The commands in CONVENTIONS "Done means" pass, and `git diff --stat docs/design/schema.sql` shows no change.

## Out of scope
Changing the schema. If the schema looks wrong, stop and write a note in `PROGRESS.md`.
