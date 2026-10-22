# Task 41: Hotel free rooms webhook

## Goal
The backend stores how many hotel rooms of each of the five room types are free. An external sender updates the
numbers through a webhook, and the website reads them from a public endpoint.

## Depends on
31.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/handler.go` (shared-secret pattern), `internal/deliveryprice/store.go`
(store pattern), `internal/config/config.go`.

## Background
The old Rails backend kept three counters in process memory (double, single, vip), so the numbers were lost on every
restart. The website now has five room types, so the old body is not kept. The sender can be reconfigured: it gets
the new URL, the new header and the new body.

Room types and how many rooms each has (these are the maximums; 12 rooms in total). The JSON keys are the website's
room ids in camelCase:

| JSON key | Website room id | Rooms | Max |
|---|---|---|---|
| `vip` | `vip` | № 301 | 1 |
| `singleBalcony` | `single-balcony` | № 201 | 1 |
| `single` | `single` | № 204, 304 | 2 |
| `doubleTwin` | `double-twin` | № 203, 206, 302, 303 | 4 |
| `doubleEuro` | `double-euro` | № 202, 205, 305, 306 | 4 |

## Files to create
- `migrations/00004_hotel_rooms.sql`
- `internal/hotel/store.go`
- `internal/hotel/handler.go`
- `internal/hotel/hotel_test.go`

## Files to change
- `internal/config/config.go`: `Hotel struct{ WebhookSecret string }`, field `Hotel Hotel` on `Config`, read from
  `HOTEL_WEBHOOK_SECRET`, default `""`.
- `.env.example`: `HOTEL_WEBHOOK_SECRET=` with a one-line comment (`TestEnvExampleIsComplete` checks it).
- `internal/app/app.go`: register `hotel.NewHandler(hotel.NewStore(db), cfg.Hotel.WebhookSecret).Routes(r)` next to the
  iiko handler; add `X-Webhook-Secret` to `Access-Control-Allow-Headers`.
- `docs/API.md`: new area `### Hotel` after `### Catalog`, with both endpoints.
- `internal/apidocs/openapi.yaml`: both paths (`TestSpecCoversAllRoutes` and `TestAPIMarkdownCoversAllRoutes` fail
  without them).

## Spec

### Migration
```sql
-- +goose Up
CREATE TABLE hotel_rooms (
    id                  smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    vip_free            integer NOT NULL CHECK (vip_free BETWEEN 0 AND 1),
    single_balcony_free integer NOT NULL CHECK (single_balcony_free BETWEEN 0 AND 1),
    single_free         integer NOT NULL CHECK (single_free BETWEEN 0 AND 2),
    double_twin_free    integer NOT NULL CHECK (double_twin_free BETWEEN 0 AND 4),
    double_euro_free    integer NOT NULL CHECK (double_euro_free BETWEEN 0 AND 4),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE hotel_rooms;
```
The table has at most one row. It is empty until the first webhook arrives.

### Store
```go
// Rooms is the number of free rooms of each type.
type Rooms struct {
	VIP           int `db:"vip_free" json:"vip"`
	SingleBalcony int `db:"single_balcony_free" json:"singleBalcony"`
	Single        int `db:"single_free" json:"single"`
	DoubleTwin    int `db:"double_twin_free" json:"doubleTwin"`
	DoubleEuro    int `db:"double_euro_free" json:"doubleEuro"`
}

// Free is the total number of free rooms.
func (r Rooms) Free() int

// State is the stored row.
type State struct {
	Rooms
	UpdatedAt time.Time `db:"updated_at"`
}

func NewStore(db store.DBTX) *Store
// Get returns store.ErrNotFound while the table is empty.
func (s *Store) Get(ctx context.Context) (State, error)
// Set replaces all five counts and returns the stored row.
func (s *Store) Set(ctx context.Context, rooms Rooms) (State, error)
```
SQL:
```sql
SELECT vip_free, single_balcony_free, single_free, double_twin_free, double_euro_free, updated_at
FROM hotel_rooms WHERE id = 1;

INSERT INTO hotel_rooms (id, vip_free, single_balcony_free, single_free, double_twin_free, double_euro_free)
VALUES (1, $1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET vip_free = EXCLUDED.vip_free, single_balcony_free = EXCLUDED.single_balcony_free,
    single_free = EXCLUDED.single_free, double_twin_free = EXCLUDED.double_twin_free,
    double_euro_free = EXCLUDED.double_euro_free, updated_at = now()
RETURNING vip_free, single_balcony_free, single_free, double_twin_free, double_euro_free, updated_at;
```

### Routes
| Method | Path | Role | Behaviour | Success |
|---|---|---|---|---|
| GET | `/hotel/rooms` | public | the current counts; 404 while the table is empty | 200 |
| POST | `/hotel/rooms` | `X-Webhook-Secret` header | validates, upserts all five counts, returns the same body as GET | 200 |

`NewHandler(s *Store, secret string) *Handler`. POST checks the secret exactly like `iiko.Handler`: `secret == ""` →
`httpx.NotFound()` (the endpoint is disabled); `crypto/subtle.ConstantTimeCompare` mismatch, including a missing header →
`httpx.Unauthorized()`. The secret is checked before the body is read.

Request body. Every type is required, so the sender always sends the full picture; a missing type is an error, not
"unchanged":
```go
type updateRequest struct {
	Rooms *roomsInput `json:"rooms" validate:"required"`
}
type roomsInput struct {
	VIP           *int `json:"vip" validate:"required,min=0,max=1"`
	SingleBalcony *int `json:"singleBalcony" validate:"required,min=0,max=1"`
	Single        *int `json:"single" validate:"required,min=0,max=2"`
	DoubleTwin    *int `json:"doubleTwin" validate:"required,min=0,max=4"`
	DoubleEuro    *int `json:"doubleEuro" validate:"required,min=0,max=4"`
}
```
Pointers make `0` valid and a missing field an error. Decode with `httpx.DecodeJSON` (unknown fields such as the old
`double_rooms` are rejected or ignored, whatever `DecodeJSON` already does; the required check catches the old body
either way), check with `validate.Struct`.

```json
POST /api/v1/hotel/rooms
X-Webhook-Secret: <HOTEL_WEBHOOK_SECRET>
{"rooms": {"vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": 0}}
```
Response for both GET and POST. `rooms` has the same keys as the request; `free` is `Rooms.Free()`, computed in the
response, not stored:
```json
{"rooms": {"vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": 0}, "free": 4,
 "updatedAt": "2026-10-22T12:00:00Z"}
```
Errors: 401 `Требуется авторизация`, 404 `Не найдено`, 400 with the standard `validate` messages
(`поле не должно быть пустым`, `значение должно быть не больше 4`, `значение должно быть не меньше 0`). Copy the
exact `errors` keys from test output into `docs/API.md`.

Log one `slog` info line on a successful POST with the five numbers.

## Tests to write
Store (`testutil.NewDB`):
- `TestStoreGetEmpty`: empty table → `store.ErrNotFound`.
- `TestStoreSetThenGet`: `Set({1,0,2,1,0})` → `Get` returns the same five values and a non-zero `UpdatedAt`.
- `TestStoreSetTwice`: two `Set` calls → `SELECT count(*) FROM hotel_rooms` is 1, values are from the second call.
- `TestRoomsFree`: table test, `{1,1,2,4,4}` → 12, `{0,0,0,0,0}` → 0, `{1,0,2,1,0}` → 4.

Handler (chi router with `Routes`; build POST requests with `httptest.NewRequest` to set `X-Webhook-Secret`,
because `testutil.DoJSON` only sets `Authorization`):
- `TestGetRoomsEmpty` → 404 `Не найдено`.
- `TestGetRooms`: after `Set({1,0,2,1,0})` → 200 and the body above with `free` 4.
- `TestPostDisabled`: secret `""` → 404.
- `TestPostUnauthorized`: no header → 401; wrong header → 401; the table stays empty.
- `TestPostInvalid`: table test → 400 with the field in `errors`: missing `doubleEuro`; `vip: 2`; `singleBalcony: 2`;
  `single: 3`; `doubleTwin: 5`; `doubleEuro: -1`; body without `rooms`; the old body
  `{"rooms": {"double_rooms": 9, "single_rooms": 2, "vip_room": 1}}`. The table stays empty after each.
- `TestPostValid`: `{1,0,2,1,0}` → 200, body `free` 4, and the row read back with `Get` has the same values.
- `TestPostMax`: `{1,1,2,4,4}` → 200, `free` 12.
- `TestPostZero`: all zeros → 200, `free` 0.

Config: extend the existing config test so `HOTEL_WEBHOOK_SECRET` is read.

## API documentation
`docs/API.md`, new `### Hotel` area: `GET /hotel/rooms` (public; notes: "404 means no data yet; the site hides the
counter"; the table of keys, website room ids and maximums from Background) and `POST /hotel/rooms` (role: public,
protected by `X-Webhook-Secret`; notes: "meant for the hotel's external sender, not the frontend; send all five types
every time").

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/hotel` and `internal/config` is at least 80%,
`docs/API.md` and `openapi.yaml` cover both endpoints, and with `make run` and `HOTEL_WEBHOOK_SECRET=test`:
```
curl -s localhost:3000/api/v1/hotel/rooms                         # 404
curl -s -X POST localhost:3000/api/v1/hotel/rooms -H 'X-Webhook-Secret: test' -H 'Content-Type: application/json' \
  -d '{"rooms":{"vip":1,"singleBalcony":0,"single":2,"doubleTwin":1,"doubleEuro":0}}'   # free 4
curl -s localhost:3000/api/v1/hotel/rooms                         # same body
```

## Out of scope
The old path `PATCH /hotels/update` and the old three-counter body, role-based editing, availability per date, a
booking engine, the Telegram bot, and the frontend (citypark-modern task 09).
