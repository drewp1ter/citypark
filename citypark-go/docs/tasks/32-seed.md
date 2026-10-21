# Task 32: Seed command

## Goal
A command that fills an empty development database with sample data, so that the frontend can be developed without iiko.

## Depends on
31.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/config/config.go`, `cmd/server/main.go`.

## Files to create
- `internal/seed/seed.go`
- `internal/seed/seed_test.go`
- `cmd/seed/main.go`

## Files to change
- `Makefile`: add target `seed` → `go run ./cmd/seed`.

## Spec

### `internal/seed/seed.go`
```go
package seed

type Summary struct {
    Users, Categories, Products, DeliveryPrices, Orders int
}

// Run inserts the sample data in one transaction.
// It returns ErrNotEmpty when the categories table already has rows.
func Run(ctx context.Context, db *sqlx.DB) (Summary, error)

var ErrNotEmpty = errors.New("seed: database is not empty")
```
Data to create (fixed values, no random generator, so that tests are stable):

| What | Rows |
|---|---|
| Users | `Тестовый пользователь`, phone `+79990000001`, email `user@example.com`, password `password123` (bcrypt via `auth.HashPassword`), role `user` |
| Delivery prices | `Сочи` 150 ₽, `Адлер` 250 ₽, `Красная Поляна` 500 ₽ |
| Categories | `Грузинская кухня` (sort 0), `Мангал` (sort 1), `Пицца` (sort 2, sale window 12:00–23:00), `Завтраки` (sort 3, 9:00–12:00, days `1111100`) |
| Products | 4 per category, with names fitting the category, prices between 150 ₽ and 1200 ₽, `measure_unit` `порц`, weights between 0.2 and 0.5, `sort_order` 0–3 |
| Product images | one per product: `https://placehold.co/600x400?text=<product id>`, type `seed` |
| Tags | `острое`, `мясо`, `вегетарианское`; link each to at least two products |
| Addresses | `Сочи`, `ул. Навагинская, 1`, linked to the user |
| Orders | 2 cash orders for the user: one with the address (delivery 150 ₽) and two items, one pickup with one item; `items_hash` from `order.ItemsHash` |

Use plain SQL with `RETURNING id` or the existing stores, whichever is shorter. The admin account is not created here (`cmd/server` does that at startup).

### `cmd/seed/main.go`
1. `config.Load(os.LookupEnv)`.
2. Refuse to run in production: if `cfg.IsProduction()` print `seed: refusing to run in production` and exit 1.
3. `store.Open`, `store.Migrate`, `seed.Run`.
4. Print the summary, for example `seeded: 1 users, 4 categories, 16 products, 3 delivery prices, 2 orders`.
5. `ErrNotEmpty` → print `seed: database is not empty, nothing done` and exit 0.

## Tests to write
`db := testutil.NewDB(t)`.
- `TestRun`: summary is `{1, 4, 16, 3, 2}`; table counts match; every product has one image; `product_tags` has at least 6 rows.
- `TestRunTwice`: the second call returns `ErrNotEmpty` and the counts do not change.
- `TestSeedUserCanSignIn`: `auth.CheckPassword(password_hash, "password123")` is true for the seeded user.
- `TestSeedOrdersLoad`: `order.NewStore(db).ListByCustomer` for the user returns 2 orders with items and totals > 0; one has an address and `DeliveryPrice == 15000`.
- `TestSeedSaleWindows`: `Завтраки` has `sale_start = 540`, `sale_end = 720`, `sale_days = '1111100'`.

## Done when
The commands in CONVENTIONS "Done means" pass, and `make db-up && make seed` prints the summary.

## Out of scope
Random data. Dropping or resetting the database (use `docker compose down -v` for that).
