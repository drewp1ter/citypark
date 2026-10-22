# Task 48: Hotel rooms content

## Goal
The five hotel room types (texts, price per night, photo) live in the database. The site reads them from
`GET /rooms`, and an admin edits one with `PUT /rooms/{id}`. The room price is edited only here.

## Depends on
45, 47.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/hotel/store.go` and `internal/hotel/handler.go` (the five room ids),
`internal/upload/storage.go`, `internal/richtext/richtext.go`,
`../citypark-modern/src/entities/room/model/rooms.data.ts` and `types.ts` (the seed values and field meanings).

## Background
Decided by the owner (`../docs/adr/0001-admin-panel.md`): the five room types are fixed (they match the free-rooms
webhook of task 41), so there is no create or delete. One price per night all year. The description is rich text
(*short* profile). Plain-text fields are stored with normal spaces; the site adds non-breaking spaces itself.

## Files to create
- `migrations/00005_rooms.sql` (table + seed), `internal/room/store.go`, `internal/room/handler.go`,
  `internal/room/room_test.go`

## Files to change
- `internal/app/app.go`: `room.NewHandler(room.NewStore(db), uploads, tokens).Routes(r)`.
- `docs/API.md` (new area `### Rooms` after `### Hotel`), `internal/apidocs/openapi.yaml`.

## Spec

### Migration
```sql
-- +goose Up
CREATE TABLE rooms (
    id             text PRIMARY KEY CHECK (id IN ('vip', 'single-balcony', 'single', 'double-twin', 'double-euro')),
    sort           integer NOT NULL,
    kind           text NOT NULL CHECK (kind IN ('single', 'double', 'vip')),
    balcony        boolean NOT NULL,
    name           text NOT NULL,
    hotel_name     text NOT NULL DEFAULT '',
    numbers        text NOT NULL,
    area           text NOT NULL,
    area_detailed  text NOT NULL DEFAULT '',
    capacity       text NOT NULL DEFAULT '',
    description    text NOT NULL DEFAULT '',
    price_kopecks  bigint NOT NULL CHECK (price_kopecks > 0),
    photo_url      text NOT NULL DEFAULT '',
    photo_alt      text NOT NULL DEFAULT '',
    photo_position text NOT NULL DEFAULT '',
    hotel_note     text NOT NULL DEFAULT '',
    updated_at     timestamptz NOT NULL DEFAULT now()
);
INSERT INTO rooms (...) VALUES (...);  -- five rows, see below

-- +goose Down
DROP TABLE rooms;
```
Seed: one row per room of `rooms.data.ts`, in its order (`sort` 1–5), with the values copied exactly, except:
`typo(...)` is dropped (store the plain string with normal spaces); `description` becomes `<p>…</p>` (empty when the
room has none); `price_kopecks` = `pricePerNight * 100`; `photo_url` = `photo.src` (`/images/...`), `photo_alt` =
`photo.alt`, `photo_position` = `photo.position` or `''`. Missing optional fields are `''`.

### Store
```go
type Room struct {
	ID            string        `db:"id"`
	Sort          int           `db:"sort"`
	Kind          string        `db:"kind"`
	Balcony       bool          `db:"balcony"`
	Name          string        `db:"name"`
	HotelName     string        `db:"hotel_name"`
	Numbers       string        `db:"numbers"`
	Area          string        `db:"area"`
	AreaDetailed  string        `db:"area_detailed"`
	Capacity      string        `db:"capacity"`
	Description   string        `db:"description"`
	Price         money.Kopecks `db:"price_kopecks"`
	PhotoURL      string        `db:"photo_url"`
	PhotoAlt      string        `db:"photo_alt"`
	PhotoPosition string        `db:"photo_position"`
	HotelNote     string        `db:"hotel_note"`
	UpdatedAt     time.Time     `db:"updated_at"`
}

func (s *Store) List(ctx context.Context) ([]Room, error)          // ORDER BY sort; never nil
func (s *Store) Get(ctx context.Context, id string) (Room, error)  // store.ErrNotFound
// Update writes the editable fields of r (everything except ID, Sort, Kind, Balcony, UpdatedAt) and returns the
// stored row and the photo path it replaced. store.ErrNotFound for an unknown id.
func (s *Store) Update(ctx context.Context, r Room) (Room, string, error)
```
`Update` runs `SELECT photo_url ... FOR UPDATE` and the `UPDATE ... RETURNING` in one transaction (the handler opens
it with `store.InTx`).

### Routes
| Method | Path | Role | Behaviour | Success |
|---|---|---|---|---|
| GET | `/rooms` | public | all five, by `sort` | 200 |
| PUT | `/rooms/{id}` | admin | replace the editable fields | 200 |

JSON of a room (GET list items and PUT response):
```json
{"id": "vip", "kind": "vip", "balcony": true, "name": "VIP-номер", "hotelName": "VIP, 2 комнаты",
 "numbers": "№ 301", "area": "45 + 7 м²", "areaDetailed": "45 м² + балкон 7 м²", "capacity": "до 4 гостей",
 "description": "<p>Гостиная и спальня, …</p>", "pricePerNight": 8000,
 "photo": {"url": "/images/room-vip.jpg", "alt": "VIP-номер", "position": ""},
 "hotelNote": "№ 301 · до 4 гостей", "updatedAt": "2026-10-22T12:00:00Z"}
```
`pricePerNight` is rubles (`money.Kopecks` JSON). `photo.url` goes through `uploads.Public`.

PUT body: the same fields without `id`, `kind`, `balcony`, `updatedAt`. Rules (trim plain strings first):
`name` required ≤ 100; `hotelName`, `numbers`, `area`, `areaDetailed`, `capacity`, `hotelNote` ≤ 100 (`numbers` and
`area` required); `pricePerNight` required, > 0, ≤ 1 000 000, at most 2 decimals (reuse the delivery price rule);
`photo.url` required, through `uploads.Stored` (400 field `photo.url`: `недопустимый адрес фото`); `photo.alt` ≤ 200;
`photo.position` `""` or `^\d{1,3}% \d{1,3}%$` ≤ 100 each (field `photo.position`: `формат «50% 30%»`);
`description` → `richtext.Sanitize(richtext.Short, …)`, then `richtext.TextLength` ≤ 1000 (field `description`:
`не более 1000 символов`). Unknown id → 404. After commit, remove the replaced photo file when the path changed
(`uploads.Remove`, log errors).

## Tests to write
- `TestSeed`: after migrations, `GET /rooms` → 5 rooms in the order `vip`, `single-balcony`, `single`,
  `double-twin`, `double-euro`, VIP `pricePerNight` 8000 and description wrapped in `<p>`.
- `TestUpdateRoom`: PUT a full valid body → 200, response and a following GET show the new values; `kind`,
  `balcony`, `sort` unchanged.
- `TestUpdateRoomSanitizes`: description `<h2>x</h2><p onclick="a">y</p>` → stored `x<p>y</p>`.
- `TestUpdateRoomPhotoReplaced`: real file saved through storage as the current photo, PUT another `/uploads/...`
  → old file removed; PUT the same URL again → file kept.
- `TestUpdateRoomInvalid`: table → 400 with the field: empty `name`; `pricePerNight` 0; `pricePerNight` 10.555;
  `photo.url` `https://evil.example/a.jpg`; `photo.position` `top`; description of 1001 letters.
- `TestUpdateRoomNotFound`: `/rooms/penthouse` → 404.
- `TestUpdateRoomAuth`: no token → 401; role `user` → 403.

## API documentation
`docs/API.md`, `### Rooms`: both routes; notes "the five room ids are fixed (the same as in `GET /hotel/rooms`)",
"`description` is HTML with `p`, `br`, `strong`, `em`, `a`, `ul`, `ol`, `li`; anything else is removed on save",
"plain texts use normal spaces; `photo.url` is either a site path (`/images/...`) or an uploaded image".

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/room` is at least 80%, docs updated, and
`curl -s localhost:3000/api/v1/rooms` after `make run` prints the five seeded rooms.

## Out of scope
Seasonal prices, adding or removing room types, galleries, the site (citypark-modern task 17).
