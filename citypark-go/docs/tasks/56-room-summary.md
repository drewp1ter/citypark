# Task 56: Room summary

## Goal
Every room has a short plain-text `summary` beside its rich-text `description`. `GET /rooms` returns it and
`PUT /rooms/{id}` stores it, so the website's room cards can show one or two sentences while `description` keeps the
full text from the old site.

## Depends on
55.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/room/handler.go`, `internal/room/store.go`, `migrations/00005_rooms.sql`.

## Background
Migration 00011 (task 55) replaced the VIP description «Гостиная и спальня, расширенная комплектация, балкон над
кронами парка.» with the old site's full equipment text (two lists, about 20 items). The website's VIP card renders the
description, so it became about 1000 px tall and stretched its grid row: a large empty space on `/rooms`. One field
cannot be both a card teaser and a full description, so the teaser gets its own field.

## Files to create
- `migrations/00012_room_summary.sql`

## Files to change
- `internal/room/store.go`: `Summary string \`db:"summary"\`` on `Room`; add `summary` to the select column list and
  to the `UPDATE` (a new `$N`, keep the order of the other arguments).
- `internal/room/handler.go`: `Summary` in `updateRequest`, `response`, `h.response` and the `Room` built in the
  validation function.
- `internal/room/room_test.go`: the tests below; add `"summary"` to `validBody()` only where a test needs it.
- `internal/apidocs/openapi.yaml`: `summary` in the room response schema and in the `PUT /rooms/{id}` request schema.
- `docs/API.md`: section `### Rooms`.

## Spec

### Migration
```sql
-- +goose Up
ALTER TABLE rooms ADD COLUMN summary text NOT NULL DEFAULT '';
UPDATE rooms SET summary = 'Гостиная и спальня, расширенная комплектация, балкон над кронами парка.', updated_at = now()
WHERE id = 'vip';

-- +goose Down
ALTER TABLE rooms DROP COLUMN summary;
```
Other rooms keep `''`: the design shows no text on their cards.

### API
- Response field `summary` (string, plain text, may be `""`), placed after `capacity` and before `description` in the JSON.
- Request field `summary`: optional, `validate:"max=200"` (the same style as `hotelName`, so the message comes from
  the validator, key `summary`). It is stored as sent: no rich-text sanitizing, no HTML. A missing field is stored as `""`
  (the PUT replaces all editable fields, like `hotelNote`).

## Tests to write
In `internal/room/room_test.go`, with the existing fixture (`testutil.NewDB`, `testutil.DoJSON`):
- `TestRoomSummarySeed`: `GET /rooms` → the `vip` item has `summary` equal to the teaser above; `single` has `""`.
- `TestUpdateRoomSummary`: `PUT /rooms/vip` with `summary` «Кратко о номере» → 200, response `summary` is the same,
  `SELECT summary FROM rooms WHERE id = 'vip'` reads it back.
- `TestUpdateRoomSummaryTooLong`: `summary` of 201 characters (Cyrillic, to check that runes are counted) → 400 with
  `errors.summary` set; the stored value is unchanged.
- `TestUpdateRoomSummaryMissing`: a body without `summary` → 200 and the stored summary is `""`.
Keep the existing tests green. If an existing assertion lists the response keys, add `summary` to it.

## API documentation
`docs/API.md`, `### Rooms`: add `"summary"` to the `GET /rooms` example (the VIP teaser) and to the `PUT /rooms/{id}`
request and response examples. Add a one-line note: «`summary` — одна-две фразы для карточки на сайте, простой текст,
до 200 символов; `description` — полный текст (HTML)».

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/room` is at least 80%, and with `make run`
`curl -s localhost:3000/api/v1/rooms | jq -r '.[0].summary'` prints the teaser.

## Out of scope
The admin form (citypark-admin task 18) and the website card (citypark-modern task 24). Do not change
`description` values or migration 00011.
