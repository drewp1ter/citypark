# Task 57: Room photo gallery

## Goal
Every room has an ordered `gallery` of extra photos beside its main `photo`. `GET /rooms` returns it, `PUT /rooms/{id}`
stores it, and the VIP room is seeded with the old site's three gallery photos (living room, bathroom, balcony), so
the website can show a room page with a gallery like the old site had.

## Depends on
56.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/room/handler.go`, `internal/room/store.go`, `internal/room/room_test.go`
(the photo replace/remove tests), `internal/upload/` (`Stored`, `Public`, `Remove`), `migrations/00005_rooms.sql`,
`migrations/00012_room_summary.sql`.

## Background
The old site (`/workspace/citypark_frontend/src/features/Hotel/components/VipRoom/VipRoom.js`) showed a gallery of
three 1280×853 photos under the VIP text. The owner wants galleries editable in the admin (citypark-admin task 19), so
the list lives in the database. The image files for the seed are site paths that citypark-modern task 25 adds to
`public/images/`; until then the website does not render the gallery, so nothing shows a broken image.

## Files to create
- `migrations/00013_room_gallery.sql`

## Files to change
- `internal/room/store.go`: `GalleryPhoto{URL, Alt}` and `Gallery` on `Room` (jsonb), select column list, `UPDATE`;
  `Update` also returns the old gallery so the handler can remove files that are no longer used.
- `internal/room/handler.go`: request, response, `trim`, `check`, and file removal after save.
- `internal/room/room_test.go`: the tests below.
- `internal/apidocs/openapi.yaml`: `gallery` in the room response schema and in the `PUT /rooms/{id}` request schema.
- `docs/API.md`: section `### Rooms`.

## Spec

### Migration
```sql
-- +goose Up
ALTER TABLE rooms ADD COLUMN gallery jsonb NOT NULL DEFAULT '[]';
UPDATE rooms SET updated_at = now(), gallery = '[
  {"url": "/images/room-vip-living.jpg", "alt": "VIP-номер: гостиная"},
  {"url": "/images/room-vip-bathroom.jpg", "alt": "VIP-номер: санузел"},
  {"url": "/images/room-vip-balcony.jpg", "alt": "VIP-номер: балкон"}
]'
WHERE id = 'vip';

-- +goose Down
ALTER TABLE rooms DROP COLUMN gallery;
```
Other rooms keep `[]`: the old site had no gallery for them.

### Store
- `type GalleryPhoto struct { URL string \`json:"url"\`; Alt string \`json:"alt"\` }` and a `Gallery` type
  (`[]GalleryPhoto`) that implements `sql.Scanner` and `driver.Valuer` through `encoding/json`; `nil` is written as
  `[]`, never `null`.
- `Room.Gallery Gallery \`db:"gallery"\``, added to the select column list and to the `UPDATE` as a new `$N` (keep the
  order of the other arguments).
- `Update` reads the old `photo_url` and `gallery` in the same `SELECT … FOR UPDATE` and returns both
  (change the signature to return a struct or a second value; update the callers and tests).

### API
- Response field `gallery`: array of `{ "url", "alt" }`, placed right after `photo`, `url` through
  `h.uploads.Public` (like `photo.url`). Always an array, `[]` when empty.
- Request field `gallery`: optional; a missing field or `null` is stored as `[]` (the PUT replaces all editable fields).
  `validate:"max=12,dive"`; per item `url` `validate:"required"`, `alt` `validate:"max=200"`. Both are trimmed in
  `trim` before validation. Each `url` goes through `h.uploads.Stored` like `photo.url`; a bad one gives
  `fields["gallery.N.url"] = "недопустимый адрес фото"` (N is the 0-based index, the same key style the validator
  uses for `dive`; check what key `validate.Struct` produces and use the same one). All invalid fields are reported
  together, as now.
- After a successful save, remove every old upload (old `photo_url` and old gallery URLs) that is in neither the new
  `photo_url` nor the new gallery. Use `h.uploads.Remove` with the same `slog.Error` on failure as the photo; `Remove`
  already ignores site paths (`/images/…`) — check, and keep it that way.

## Tests to write
In `internal/room/room_test.go`, with the existing fixture:
- `TestRoomGallerySeed`: `GET /rooms` → `vip` has 3 items with the URLs and alts above; `single` has `[]` (not `null`:
  check the raw JSON).
- `TestUpdateRoomGallery`: `PUT /rooms/single` with two uploaded files in `gallery` → 200, the same order in the
  response and in `SELECT gallery`.
- `TestUpdateRoomGalleryMissing`: a body without `gallery` → 200, stored `[]`.
- `TestUpdateRoomGalleryTooMany`: 13 items → 400 with `errors.gallery`; stored value unchanged.
- `TestUpdateRoomGalleryBadURL`: item 0 `https://evil.example/a.jpg` → 400, `errors["gallery.0.url"]`; item with
  201-character Cyrillic alt → its alt key.
- `TestUpdateRoomGalleryRemovesFiles`: upload a, b; save gallery [a, b]; save [b] → a's file removed, b kept; save
  photo = b and gallery [] → b kept (moved to the photo, not deleted).
Keep the existing tests green. If an existing assertion lists the response keys, add `gallery` to it.

## API documentation
`docs/API.md`, `### Rooms`: add `"gallery"` to the `GET /rooms` example (the VIP items) and to the `PUT /rooms/{id}`
request table (`gallery` | array | no | at most 12 items; `url` as in `photo.url`; `alt` at most 200 characters; a
missing field is stored as `[]`) and examples. One-line note: «`gallery` — дополнительные фото для страницы номера,
в заданном порядке; `photo` остаётся главным фото карточки».

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/room` is at least 80%, and with `make run`
`curl -s localhost:3000/api/v1/rooms | jq '.[0].gallery | length'` prints `3`.

## Out of scope
The admin form (citypark-admin task 19), the website room page and the image files (citypark-modern task 25),
captions or positions for gallery photos.
