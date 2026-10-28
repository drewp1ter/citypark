# Task 59: Delivery prices from the old site, free delivery threshold

## Goal
`delivery_prices` holds the 58 real towns and prices of the old site after `make migrate`, and a town can have a
free-delivery threshold: Белореченск costs 100 ₽ and is free from an order of 500 ₽. `POST /orders` applies it.

## Depends on
16, 26.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/deliveryprice/store.go`, `internal/order/service.go` (`Create`, `saveParams`,
`save`), `migrations/00010_content_blocks_seed.sql` (seed migration style).

## Background
The old Rails backend had no table: prices were a hard-coded `case city` in `lib/order_ext/item_methods.rb`. Белореченск
was 100 ₽, and 0 ₽ when the order sum was at least `MIN_AMOUNT_TO_FREE_DELIVERY` (500). Any other town cost 500 ₽; the
new backend keeps rejecting unknown towns with 400 (the site shows a picker from `GET /delivery-prices`), so this
default is **not** ported. The old data wrote `c. Школьное` with a Latin `c`; the rows below use Cyrillic `с.`.

## Files to create
- `migrations/00014_delivery_prices_seed.sql`

## Files to change
- `internal/deliveryprice/store.go`, `internal/deliveryprice/deliveryprice_test.go`: `FreeFrom`.
- `internal/order/service.go`, `internal/order/service_test.go`: the free-delivery rule.
- `internal/seed/seed.go`, `internal/seed/seed_test.go`: drop the fake towns.
- `internal/store/migrate_test.go`: the seeded rows.
- `internal/apidocs/openapi.yaml` (`DeliveryPrice`, `DeliveryPriceInput`, the `POST /orders` description),
  `docs/API.md` (`### Delivery prices`, `#### POST /orders`).

## Spec

### Migration
```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE delivery_prices
    ADD COLUMN free_from_kopecks bigint NOT NULL DEFAULT 0 CHECK (free_from_kopecks >= 0);

INSERT INTO delivery_prices (name, price_kopecks, free_from_kopecks) VALUES
    ('Белореченск', 10000, 50000),
    ('Майкоп', 85000, 0),
    ('п. Родники', 25000, 0),
    ('п. Южный', 25000, 0),
    ('п. Садовый', 26000, 0),
    ('п. Заречный', 27000, 0),
    ('п. Новый', 25000, 0),
    ('ст. Пшехская', 45000, 0),
    ('х. Долгогусевский', 72000, 0),
    ('с. Великовечное', 80000, 0),
    ('п. Восточный', 42000, 0),
    ('п. Степной', 60000, 0),
    ('п. Первомайский', 75000, 0),
    ('ст. Кубанская', 85000, 0),
    ('ст. Бжедуховская', 110000, 0),
    ('ст. Ханская', 65000, 0),
    ('х. Кубанский', 80000, 0),
    ('х. Подгорный', 40000, 0),
    ('п. Гончарка', 65000, 0),
    ('х. Гавердовский', 80000, 0),
    ('х. Грушовый', 42000, 0),
    ('ДНТ Надежда', 20000, 0),
    ('ДНТ Золотая Осень', 20000, 0),
    ('ДНТ Весна', 20000, 0),
    ('ДНТ Солнышко', 20000, 0),
    ('ДНТ Водитель', 20000, 0),
    ('ДНТ Садко', 20000, 0),
    ('х. Новоселовский', 65000, 0),
    ('ДНТ Аграрник', 20000, 0),
    ('ДНП Строитель', 25000, 0),
    ('ДНТ Химик', 20000, 0),
    ('ДНТ Дружба', 20000, 0),
    ('СНТ Росинка', 20000, 0),
    ('СОТ Колосок', 20000, 0),
    ('СОТ Колосок 2', 20000, 0),
    ('СОТ Рубин', 20000, 0),
    ('СОТ Коммунальник', 20000, 0),
    ('СОТ Коммунальник 2', 20000, 0),
    ('СОТ Бытсервис', 20000, 0),
    ('СОТ Ветерок', 22000, 0),
    ('СОТ Ромашка', 20000, 0),
    ('СОТ Ручеек', 25000, 0),
    ('ст. Тверская', 120000, 0),
    ('х. Приречный', 26000, 0),
    ('с. Школьное', 70000, 0),
    ('п. Верхневеденеевский', 70000, 0),
    ('х. Привольный', 70000, 0),
    ('п. Дружный', 65000, 0),
    ('п. Мирный', 80000, 0),
    ('п. Молодежный', 85000, 0),
    ('х. Фадеевский', 80000, 0),
    ('х. Терновый', 75000, 0),
    ('ст. Черниговская', 90000, 0),
    ('Майкопское Шоссе', 25000, 0),
    ('ХИМ завод', 65000, 0),
    ('Бригада (азерчай)', 20000, 0),
    ('ст. Ерик', 110000, 0),
    ('п. Комсомольский', 120000, 0)
ON CONFLICT (name) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM delivery_prices WHERE name = ANY(ARRAY[ ...the same 58 names... ]);
ALTER TABLE delivery_prices DROP COLUMN free_from_kopecks;
-- +goose StatementEnd
```
`ON CONFLICT DO NOTHING` keeps a price an admin already entered under the same name. `value` stays `''`.

### deliveryprice
- `DeliveryPrice` gets `FreeFrom money.Kopecks \`db:"free_from_kopecks" json:"freeFrom"\`` after `Price`, comment
  `// order total from which delivery is free; 0 = never free`.
- `Input` gets `FreeFrom money.Kopecks \`json:"freeFrom" validate:"gte=0"\``. Missing in JSON = 0.
- `columns`, `INSERT` and `UPDATE` include `free_from_kopecks` (`$4` in INSERT, `$5` in UPDATE).

### order
- `saveParams.delivery` becomes `deliveryprice.DeliveryPrice` (zero value for pickup). `Create` passes `dp` as it is.
- In `save`, while building `insertItems`, sum `subtotal += pr.Price.Mul(it.Amount)`. Then
  ```go
  // deliveryCost is the price of the town, or 0 when the order reaches the town's free-delivery threshold.
  func deliveryCost(dp deliveryprice.DeliveryPrice, subtotal money.Kopecks) money.Kopecks
  ```
  returns 0 when `dp.FreeFrom > 0 && subtotal >= dp.FreeFrom`, else `dp.Price`. Insert its result as `DeliveryPrice`.
- The subtotal uses the same prices as `OfferedPrice`, so it equals the order `total`.

### seed
Remove `deliveryPrices`, its insert loop and `Summary.DeliveryPrices` (the migration fills the real towns). The seed
address city `Сочи` becomes `Белореченск`.

### API docs
- `DeliveryPrice` and `DeliveryPriceInput`: `freeFrom`, number, rubles, `minimum: 0`, description «Order total from
  which delivery to this town is free. 0 = never free.»; required in `DeliveryPrice`, optional in the input.
- Replace the `Сочи`/`Адлер` examples in `docs/API.md` and `openapi.yaml` with `Белореченск` (100, `freeFrom` 500) and
  `п. Родники` (250, `freeFrom` 0); add `freeFrom` to the request tables.
- `POST /orders` notes: «`deliveryPrice` is the town's `price`, or 0 when `total` is at least the town's `freeFrom`
  (when `freeFrom` is above 0).»

## Tests to write
- `TestSeededDeliveryPrices` (`internal/store`): `MigrateDown` (ignore error), `Migrate`, then
  `SELECT count(*) FROM delivery_prices` → 58; `Белореченск` → `10000, 50000`; `п. Комсомольский` → `120000, 0`;
  `с. Школьное` exists and `c. Школьное` (Latin) does not. End with the database migrated, as `TestMigrateUpDown` does.
- `TestDeliveryCost` (table, `order`), `dp = {Price: 10000, FreeFrom: 50000}`: subtotal 49999 → 10000; 50000 → 0;
  60000 → 0; `dp = {Price: 25000, FreeFrom: 0}`, subtotal 500000 → 25000; zero `dp` → 0.
- `TestCreateFreeDelivery` (`order`): town `Белореченск` 100 ₽ free from 500 ₽ (insert it with SQL or extend
  `testutil.InsertDeliveryPrice` only if needed); one product 250 ₽ × 2 → stored `deliveryPrice` 0; × 1 → 100.
- `deliveryprice`: `TestCreate` and `TestUpdate` send and check `freeFrom` (for example 500 → `50000` in the row, `500`
  in JSON); `TestCreateValidation` gets the case `freeFrom: -1` → 400 with an error for `freeFrom`; a create without
  `freeFrom` stores 0.
- `seed_test.go`: the expected `Summary` without `DeliveryPrices`.

## Done when
CONVENTIONS "Done means" pass and `docs/API.md` is updated. On the dev database `curl -s
localhost:3000/api/v1/delivery-prices | jq length` prints 58.

## Out of scope
- The website: `citypark-modern/src/entities/dish/model/delivery.ts` keeps `freeFrom: 500` and
  `lib/quote-delivery.ts` keeps its zones; reading prices and `freeFrom` from the API is a citypark-modern task.
- The admin panel form for `freeFrom` (citypark-admin).
- A default price for unknown towns.
