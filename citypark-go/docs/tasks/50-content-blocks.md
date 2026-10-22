# Task 50: Content blocks

## Goal
Editable page copy is stored as **content blocks**: rows addressed by a block key (`home.hero.title`) with a type
(`text`, `lines`, `richtext`, `image`). The site reads them by page, and an admin changes a block's value. This task
creates the table and the endpoints; the rows come in task 54.

## Depends on
45, 47.

## Read first
`docs/tasks/CONVENTIONS.md`, `../GLOSSARY.md` (Content block, Block key, Price placeholder, Rich text profile),
`internal/upload/storage.go`, `internal/richtext/richtext.go`.

## Background
Decided by the owner (`../docs/adr/0001-admin-panel.md`): keys are created only by migrations, each with a Russian
label for the admin; the admin edits values and never adds or deletes keys. Prices are never typed into a text: a text
uses a price placeholder such as `{{smokingFine}}`, and the site fills it in.

## Files to create
- `migrations/00007_content_blocks.sql`, `internal/content/store.go`, `internal/content/handler.go`,
  `internal/content/content_test.go`

## Files to change
- `internal/app/app.go`: `content.NewHandler(db, uploads, tokens).Routes(r)`.
- `docs/API.md` (new area `### Content`), `internal/apidocs/openapi.yaml`.

## Spec

### Migration
```sql
-- +goose Up
CREATE TABLE content_blocks (
    key            text PRIMARY KEY CHECK (key ~ '^[a-z0-9-]+(\.[a-z0-9-]+)+$'),
    page           text NOT NULL,
    section        text NOT NULL DEFAULT '',
    sort           integer NOT NULL DEFAULT 0,
    label          text NOT NULL,
    type           text NOT NULL CHECK (type IN ('text', 'lines', 'richtext', 'image')),
    max_length     integer NOT NULL DEFAULT 0,
    text_value     text NOT NULL DEFAULT '',
    image_url      text NOT NULL DEFAULT '',
    image_alt      text NOT NULL DEFAULT '',
    image_position text NOT NULL DEFAULT '',
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX content_blocks_page_idx ON content_blocks (page, sort);

-- +goose Down
DROP TABLE content_blocks;
```
`page` is the first segment of the key; `section` is a Russian heading the admin groups blocks under («Первый экран»).
`lines` are stored in `text_value` joined with `\n`. `max_length` `0` means the default of the type.

### Value rules (`func validateValue(b Block, in valueInput) (stored string, err error)`)
| Type | Request value | Rules (field `value` unless said otherwise) |
|---|---|---|
| `text` | `"value": "строка"` | trimmed; no `\n`; ≤ `max_length` (default 300): `не более N символов` |
| `lines` | `"value": ["a", "b"]` | each trimmed, empty ones dropped; 1–50 lines: `от 1 до 50 строк`; each ≤ 200 |
| `richtext` | `"value": "<p>…</p>"` | `richtext.Sanitize(Short, …)`; `TextLength` ≤ `max_length` (default 2000) |
| `image` | `"image": {"url", "alt", "position"}` | `url` required, `uploads.Stored` (field `image.url`); `alt` ≤ 200; `position` as in task 48 |

Placeholders: in `text`, `lines` and `richtext` every `{{…}}` (regexp `\{\{\s*([a-zA-Z]+)\s*\}\}`) must name one of
`minRoomPrice`, `lateCheckoutPerHour`, `smokingFine`, `breakfastDelivery` (export them as
`var Placeholders = []string{...}`), else 400 field `value`: `неизвестная подстановка {{name}}`. Store them as typed.

### Routes
| Method | Path | Role | Behaviour | Success |
|---|---|---|---|---|
| GET | `/content/blocks` | public | all blocks, or `?page=home` only; by `page, sort, key` | 200 |
| PUT | `/content/blocks/{key}` | admin | set the value of an existing block | 200 |

Block JSON (`value` depends on the type; `image` is `null` except for `image` blocks):
```json
{"key": "home.hero.title", "page": "home", "section": "Первый экран", "label": "Заголовок", "type": "text",
 "maxLength": 300, "value": "Отель, ресторан и доставка", "image": null, "updatedAt": "..."}
{"key": "hotel.amenities", "type": "lines", "value": ["Сплит-система", "Smart TV"], "image": null, ...}
{"key": "home.hero.photo", "type": "image", "value": null,
 "image": {"url": "/images/hotel-building.jpg", "alt": "Здание отеля", "position": ""}, ...}
```
`maxLength` is the effective limit (the default when the column is 0; `0` for `image`). `image.url` through
`uploads.Public`. PUT body: `{"value": ...}` or `{"image": {...}}` as in the table; a `value` of the wrong JSON type for
the block → 400 field `value`: `неверный тип значения`. Unknown key → 404. After an `image` PUT commits, remove the
replaced uploaded file. PUT response: the block.

## Tests to write
Insert test blocks with SQL in each test (the table has no rows yet): one of each type, `max_length` 20 on the text
one.
- `TestListBlocks`: all → 4 blocks with values of the right JSON type; `?page=hotel` → only hotel blocks;
  `?page=none` → `[]`.
- `TestPutText`: `"  Новый заголовок  "` → stored trimmed; 21 characters → 400 `не более 20 символов`; a value with
  `\n` → 400.
- `TestPutLines`: `["a", " ", "b "]` → `["a", "b"]`; `[]` → 400; 51 lines → 400.
- `TestPutRichtext`: `<h2>x</h2><p>y</p>` → `x<p>y</p>`.
- `TestPutPlaceholders`: `Штраф {{smokingFine}}` → ok, stored unchanged; `{{price}}` → 400
  `неизвестная подстановка {{price}}`.
- `TestPutImage`: new `/uploads/...` → stored; the old uploaded file is removed; `https://x.y/a.jpg` → 400 field
  `image.url`.
- `TestPutWrongType`: `{"value": ["a"]}` to a `text` block → 400 `неверный тип значения`; `{"value": "a"}` to an
  `image` block → 400.
- `TestPutUnknownKey` → 404; `TestPutAuth`: no token → 401, role `user` → 403.

## API documentation
`docs/API.md`, `### Content`: both routes, the type table, the placeholder list with what the site puts there
(`minRoomPrice`: lowest `pricePerNight` of the rooms; the others: the hotel prices of task 52), and the note "keys are
created by the backend; there is no endpoint to add or delete one".

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/content` is at least 80%, docs updated.

## Out of scope
The block rows (task 54), drafts, history of changes.
