# Task 91: Delivery information settings

## Goal
Persist the cooking-time text and outskirts delivery label/range currently hardcoded in the public website. Expose public reads and admin-only updates without duplicating town tariffs or free-delivery thresholds.

## Depends on
Backend task 90. Consumers: [admin task 30](../../../citypark-admin/docs/tasks/30-delivery-settings.md) and [website task 49](../../../citypark-modern/docs/tasks/49-delivery-settings.md).

## Read first
- `docs/tasks/CONVENTIONS.md`
- `internal/workinghours/handler.go`
- `internal/workinghours/store.go`
- `internal/app/app.go`

## Files to create
- `migrations/00031_delivery_settings.sql`
- `internal/deliverysettings/store.go`
- `internal/deliverysettings/handler.go`
- `internal/deliverysettings/deliverysettings_test.go`

## Files to change
- `internal/app/app.go`: register the feature with the existing admin tokens.
- `internal/app/app_test.go`: verify routes are wired.
- `docs/API.md` and `internal/apidocs/openapi.yaml`: document both routes and schemas.
- `docs/tasks/PROGRESS.md`: completion status.

## Spec

Public types and constructors:

```go
type Settings struct {
    Cooking string `json:"cooking" db:"cooking"`
    OutskirtsLabel string `json:"outskirtsLabel" db:"outskirts_label"`
    OutskirtsMin money.Kopecks `json:"outskirtsMin" db:"outskirts_min_kopecks"`
    OutskirtsMax money.Kopecks `json:"outskirtsMax" db:"outskirts_max_kopecks"`
}
type Store struct { db store.DBTX }
func NewStore(db store.DBTX) *Store
func (s *Store) Get(ctx context.Context) (Settings, time.Time, error)
func (s *Store) Put(ctx context.Context, settings Settings) (time.Time, error)
func NewHandler(s *Store, tokens *auth.Tokens) *Handler
func (h *Handler) Routes(r chi.Router)
```

Migration Up (use the repository's goose directives); Down drops `delivery_settings`:

```sql
CREATE TABLE delivery_settings (
    id smallint PRIMARY KEY CHECK (id = 1),
    cooking text NOT NULL CHECK (char_length(btrim(cooking)) BETWEEN 1 AND 100),
    outskirts_label text NOT NULL CHECK (char_length(btrim(outskirts_label)) BETWEEN 1 AND 60),
    outskirts_min_kopecks bigint NOT NULL CHECK (
        outskirts_min_kopecks BETWEEN 0 AND 10000000 AND outskirts_min_kopecks % 100 = 0),
    outskirts_max_kopecks bigint NOT NULL CHECK (
        outskirts_max_kopecks BETWEEN 0 AND 10000000 AND outskirts_max_kopecks % 100 = 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (outskirts_min_kopecks <= outskirts_max_kopecks)
);
INSERT INTO delivery_settings (id, cooking, outskirts_label, outskirts_min_kopecks, outskirts_max_kopecks)
VALUES (1, '~30 мин', 'СОТ и ДНТ', 20000, 35000);
```

Store statements:

```sql
SELECT cooking, outskirts_label, outskirts_min_kopecks, outskirts_max_kopecks, updated_at
FROM delivery_settings WHERE id = 1;

UPDATE delivery_settings SET cooking = $1, outskirts_label = $2,
    outskirts_min_kopecks = $3, outskirts_max_kopecks = $4, updated_at = now()
WHERE id = 1 RETURNING updated_at;
```

Map missing rows to `store.ErrNotFound`; propagate other I/O errors. Do not recreate a missing row in GET or PUT. Updates replace all four fields atomically. No writes to `site_settings`, `delivery_prices`, orders or payment logic.

Paths below are relative to `/api/v1`:

| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| GET | `/delivery-settings` | Public | Current settings and update time | 200 |
| PUT | `/delivery-settings` | Admin | Validate, replace all four fields, return saved settings/time | 200 |

PUT body (all fields required):

```json
{"cooking":"~30 мин","outskirtsLabel":"СОТ и ДНТ","outskirtsMin":200,"outskirtsMax":350}
```

Both success responses have those four fields plus `updatedAt` (RFC 3339), for example:

```json
{"cooking":"~30 мин","outskirtsLabel":"СОТ и ДНТ","outskirtsMin":200,"outskirtsMax":350,"updatedAt":"2026-11-01T12:00:00Z"}
```

- Trim text before validation/storage. Text is plain text; require cooking length 1–100 and label length 1–60 Unicode characters. Accept arbitrary nonempty plain text, not HTML rendering.
- Decode monetary inputs as required pointer integers in rubles, then convert to `money.Kopecks`. This detects missing/null values, allows zero and rejects fractional JSON numbers before money rounding. Keep integers within 0–100000; minimum must not exceed maximum. Equal bounds are valid.
- Use `httpx.DecodeJSON`, `validate.Struct`, existing middleware and standard error envelopes. Missing/null numeric values use required validation. Wrong JSON types, fractions and malformed JSON use the existing decode error; do not add permissive string parsing.
- Validation top message: `Ошибка валидации, пожалуйста проверьте введенные вами данные`.
- Required field error: `поле не должно быть пустым`; text max error: `значение должно быть не больше 100` / `значение должно быть не больше 60`; price bounds: `значение должно быть не меньше 0` / `значение должно быть не больше 100000`.
- Reversed range: 400 with `errors.outskirtsMax` = `Максимальная цена должна быть не меньше минимальной`.
- Reuse existing exact 401, 403, 404, malformed-request and 500 messages. Invalid requests must not change the row. No new cache policy; frontend reads use 60-second revalidation.

## Tests to write
Use real PostgreSQL via `testutil.NewDB(t)`, chi routes and `testutil.DoJSON`; never start a database or use `t.Parallel()`. Each test restores this singleton to seeded values in cleanup because it is not truncated by the shared helper.

- `TestDeliverySettingsDefaults`: migrated database → exact seeded settings and nonzero timestamp; also verify database constraints and migration Up/Down/Up on an isolated database using the existing migration-test pattern.
- `TestDeliverySettingsStore`: replace values → a new store reads the saved row/time; direct SQL confirms kopecks; missing row → `ErrNotFound`; closed DB → propagated errors. Restore the singleton after missing-row cases.
- `TestDeliverySettingsPublicGet`: unauthenticated GET → exact response fields and defaults; no write occurs.
- `TestDeliverySettingsPutAuth`: no token → 401, customer token → 403, admin token → 200; unauthorized attempts preserve the row.
- `TestDeliverySettingsPut`: normal update, surrounding whitespace, zero range, equal bounds and maximum bounds → saved normalized values and timestamp.
- `TestDeliverySettingsValidation`: table cases for empty/whitespace/overlong Unicode text; missing/null numeric inputs; negative/too-large/reversed ranges; fractions, numeric strings, booleans and malformed JSON. Assert field errors where specified and unchanged database state.
- `TestDeliverySettingsErrors`: missing singleton → GET/PUT 404; database failure → GET/PUT 500, without leaked SQL errors.
- Add an app-route registration check and maintain existing OpenAPI contract checks.

## API documentation
Add GET/PUT sections under delivery information in `docs/API.md`. Add `DeliverySettingsInput` and `DeliverySettings` schemas to OpenAPI: all four input fields required, numeric fields integer rubles with bounds, response also requires `updatedAt`. Capture final response/error examples from tests. Explicitly distinguish informational outskirts prices from authoritative town tariffs.

## Done when
Load `.env.development` and run all checks in CONVENTIONS (full tests use `-p 1`); every changed internal package has ≥80% coverage. Migration, route/auth/validation tests and API/OpenAPI checks pass. No live settings writes, orders, payments or notifications are needed.

## Out of scope
Admin UI (task 30), public frontend integration (task 49), working hours, town tariff changes, promotional pricing and order calculations.
