# Task 63: Working hours from the database

## Goal
The hours when orders are accepted are one row in the database. The site reads them from `GET /working-hours`, an
admin changes them with `PUT /working-hours`, and `POST /orders` uses the current row instead of the
`WORKING_HOURS_*` config, which is removed. A «Ко времени» `deliveryTime` outside the hours is rejected.

## Depends on
27, 61.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/settings/store.go` and `internal/settings/handler.go` (single-row table
pattern), `internal/order/workinghours.go`.

## Background
Decided by the owner (2026-10-28). The hours start as 09:00–22:00 (the old config values). They are edited on the
admin's «Доставка» page (citypark-admin task 21) and shown by the site (citypark-modern task 29). No overnight hours:
`start` is before `end`. Task 61 said «No check against working hours»; this task reverses that.

## Files to create
- `migrations/00017_working_hours.sql`
- `internal/workinghours/store.go`, `internal/workinghours/handler.go`, `internal/workinghours/workinghours_test.go`

## Files to change
- `internal/order/workinghours.go`, `internal/order/handler.go`, `internal/order/create_handler_test.go`,
  `internal/order/callback_test.go` (it builds `HandlerOptions`).
- `internal/config/config.go`, `internal/config/config_test.go`: remove `WorkingHours`, its type, its `Load` lines,
  `TestLoadEmptyWorkingHours` and the `WORKING_HOURS_*` rows.
- `internal/app/app.go`: `hours := workinghours.NewStore(db)`, `workinghours.NewHandler(hours, tokens).Routes(r)`,
  `order.HandlerOptions{Hours: hours, DefaultPageSize: cfg.DefaultPageSize}`. `internal/app/app_test.go`: drop the
  `WORKING_HOURS_*` env lines.
- `.env.example` (remove the «Working hours» block), `README.md` (remove the two rows; the `APP_TIMEZONE` row stays).
- `docs/API.md`, `internal/apidocs/openapi.yaml`.

## Spec

### Migration
```sql
-- +goose Up
CREATE TABLE working_hours (
    id         smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    start_time text NOT NULL CHECK (start_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    end_time   text NOT NULL CHECK (end_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (start_time < end_time)
);
INSERT INTO working_hours (start_time, end_time) VALUES ('09:00', '22:00');

-- +goose Down
DROP TABLE working_hours;
```

### Package `workinghours`
```go
// Hours are "HH:MM" with a leading zero, start before end, the same day.
type Hours struct {
	Start string `json:"start" validate:"required,hhmm"`
	End   string `json:"end" validate:"required,hhmm"`
}

// Contains reports whether hhmm ("18:30") is within [Start, End], both ends included (used for deliveryTime).
func (h Hours) Contains(hhmm string) bool
// OpenAt reports whether t (in its own location) is within [Start, End): the order window, end excluded.
func (h Hours) OpenAt(t time.Time) bool
// Display drops a leading zero: "09:00" -> "9:00", "22:00" -> "22:00".
func Display(hhmm string) string

func NewStore(db *sqlx.DB) *Store
// Get returns store.ErrNotFound when the row is missing.
func (s *Store) Get(ctx context.Context) (Hours, time.Time, error)
func (s *Store) Put(ctx context.Context, h Hours) (time.Time, error) // updated_at = now(), RETURNING updated_at
```
`OpenAt` reuses `salewindow.ParseClock` and `salewindow.Window{Start, End}.OpenAtTime` (the same rule the middleware
has today). `Contains` compares the strings (both are zero-padded).

### Routes
| Method | Path | Role | Behaviour | Success |
|---|---|---|---|---|
| GET | `/working-hours` | public | the row | 200 |
| PUT | `/working-hours` | admin | replace both times | 200 |

Response: `{"start": "09:00", "end": "22:00", "updatedAt": "2026-10-28T12:00:00Z"}`. PUT body:
`{"start": "10:00", "end": "23:00"}` (an `updatedAt` in the body is ignored). Trim both strings, then
`validate.Struct` (`hhmm` message «Укажите время в формате ЧЧ:ММ»). When both are valid and `start >= end`: 400 with
`errors.end = "Время окончания должно быть позже начала"` and the usual top message. Missing row: GET and PUT → 404.

### order
```go
// HoursSource is satisfied by *workinghours.Store; tests use a fake.
type HoursSource interface {
	Get(ctx context.Context) (workinghours.Hours, time.Time, error)
}
type HandlerOptions struct {
	Hours           HoursSource
	DefaultPageSize int
}
func WorkingHours(c clock.Clock, hours HoursSource) func(http.Handler) http.Handler
```
- The middleware reads the hours on every request (it only wraps `POST /orders`). A read error is returned through
  `httpx.WriteError` (500); it never lets the order through. Closed → 503, message
  `"Извините, но мы не можем принять ваш заказ.\nЗаказы принимаются с " + Display(start) + " до " + Display(end)`
  (unchanged text).
- `Handler.create`, after `validate.Struct(in)`: when `in.DeliveryTime != ""`, read the hours; if
  `!hours.Contains(in.DeliveryTime)`, return 400 with the validation top message and
  `errors.deliveryTime = "Мы работаем с " + Display(start) + " до " + Display(end)` (the same text the site shows).
- The "unparsable or empty config means always open" behaviour is gone; delete those test rows.

## Tests to write
- `workinghours_test.go` (restore the row after each test, like `settings_test.go`'s fixture):
  - `TestSeed`: GET → `start` `09:00`, `end` `22:00`, `updatedAt` set.
  - `TestPut`: PUT `10:00`/`23:00` → 200 with the new values; GET shows them; `updatedAt` changed.
  - `TestPutTrims`: `" 10:00 "` → stored `10:00`.
  - `TestPutInvalid` (table): `start` `9:00` → `errors.start` hhmm message; `end` `25:00` → `errors.end` hhmm message;
    `start` empty → `errors.start`; `12:00`/`12:00` and `13:00`/`12:00` → `errors.end` «Время окончания должно быть
    позже начала».
  - `TestPutAuth`: no token → 401; role `user` → 403.
  - `TestHours` (table, no DB): `Contains` for `08:59` false, `09:00` true, `22:00` true, `22:01` false; `OpenAt` at
    8:59 false, 9:00 true, 21:59 true, 22:00 false; `Display` `09:00` → `9:00`, `00:30` → `0:30`, `22:00` → `22:00`.
- `create_handler_test.go`: a `fakeHours{h workinghours.Hours; err error}`; `defaultOpts()` uses `09:00`/`22:00`.
  - `TestWorkingHoursBoundaries`: keep the four time rows, drop the «unparsable»/«empty» rows; add «read error» → 500.
  - `TestCreateDeliveryTimeOutsideHours`: `deliveryTime` `08:30` and `22:30` → 400
    `errors.deliveryTime = "Мы работаем с 9:00 до 22:00"`; `22:00` → 201.
  - `TestCreateOrderNoWorkingHours` is removed (there is no "no hours" state any more).
- `config_test.go`: the remaining tests pass without the `WorkingHours` rows.

## API documentation
- New `### Working hours` area: both routes, the JSON above, the 400 errors, note «times are `HH:MM` with a leading
  zero; show them with `9:00`-style formatting; orders are accepted from `start` until `end` (not at `end`), a
  «Ко времени» time may equal `end`».
- `#### POST /orders`: the 503 row says «the hours from `GET /working-hours`»; the `deliveryTime` row no longer says
  «not checked»; add the 400 `errors.deliveryTime: «Мы работаем с 9:00 до 22:00»` (built from the current hours).
- `openapi.yaml`: `WorkingHours` schema and both paths; the same changes on `POST /orders`.

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/workinghours`, `internal/order` and
`internal/config` is at least 80%, docs updated, and `grep -rn WORKING_HOURS .` finds only `docs/tasks/`.

## Out of scope
- Weekday schedules, holidays, overnight hours.
- Caching the row.
- The admin panel (citypark-admin task 21) and the site (citypark-modern task 29).
- The owner removes `WORKING_HOURS_*` from local `.env` files; note it in PROGRESS.md.
