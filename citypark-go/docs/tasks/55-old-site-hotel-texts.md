# Task 55: Hotel guest information and room texts from the old site

## Goal
A migration fills the five empty «Информация для гостей» sections of the hotel page and the four empty room
descriptions with the texts of the old site. After it,
`GET /content/blocks?page=hotel` and `GET /rooms` return these texts, and the hotel page no longer shows «Скоро добавим.».

## Depends on
48, 54.

## Read first
`docs/tasks/CONVENTIONS.md`, `migrations/00005_rooms.sql`, `migrations/00010_content_blocks_seed.sql` (rows
`hotel.guest-info.*`), `internal/content/handler.go` (`validateRichtext`), `internal/richtext/richtext.go`.

Source texts (old CRA site, read only, copy the wording exactly):
- `/workspace/citypark_frontend/src/features/Hotel/components/Hotel/Hotel.js`: «Приложение № 1» … «Приложение №8».
- `/workspace/citypark_frontend/src/features/Hotel/components/SingleRoom/SingleRoom.js`, `DoubleRoom/DoubleRoom.js`,
  `VipRoom/VipRoom.js`: the `room_text` block.

## Background
Migration 00010 seeded `hotel.guest-info.2.text` … `hotel.guest-info.6.text` as `''`, and 00005 seeded the rooms
`single-balcony`, `single`, `double-twin`, `double-euro` with `description = ''`. The old site has all these texts.
Content block values are checked with the `Short` rich text profile: only `p`, `br`, `strong`, `em`, `ul`, `ol`, `li`,
`a` survive. So the old tables become lists, and the old headings become `<p><strong>…</strong></p>`. The old texts
are longer than the current `max_length` of 1000.

## Files to create
- `migrations/00011_old_site_hotel_texts.sql`
- `internal/content/old_site_texts_test.go`

## Spec

### Content blocks
For each key below: `UPDATE content_blocks SET text_value = '<html>', max_length = 6000, updated_at = now() WHERE key = '<key>';`

| Key | Old source | Content |
|---|---|---|
| `hotel.guest-info.2.text` | Приложение № 1 | One `<ul>`, one `<li>` per category: «Люкс» — 8 000 ₽; «Первая категория» одноместный с двуспальной кроватью — 4 000 ₽; … (all rows of the old table, «₽» instead of a bare number). Then `<p>` with the footnote «* — согласно приложению к свидетельству о присвоении категории № 23/АА-165-2021/0091-2022». |
| `hotel.guest-info.3.text` | Приложения № 2, 3, 4 | `<p><strong>Завтраки</strong></p>`, `<ul>` of every Russian breakfast row as «Омлет (130 гр) — 250 ₽» (drop the English rows), `<p>` «Заказ завтраков принимается до 22:00. Доставка еды в номер — {{breakfastDelivery}}.»; then `<p><strong>Лобби-бар</strong></p>` and its `<ul>`; then `<p><strong>Прачечная</strong></p>` and the two sentences of Приложение 4. |
| `hotel.guest-info.4.text` | Приложение № 5 | The heading sentence and each paragraph as `<p>`. |
| `hotel.guest-info.5.text` | Приложения № 6, 7 | `<p><strong>Правила проживания</strong></p>` and its paragraphs (drop «Утверждаю ___ ген.директор …»); `<p><strong>Пожарная безопасность</strong></p>` and its paragraphs, the two «В случае пожара…» lists as `<ul>`. |
| `hotel.guest-info.6.text` | Приложение №8 | `<ul>`, one `<li>` per item: «Халат — 2 000 ₽, частичная порча — 900 ₽»; items with one price: «Занавески — 5 000 ₽». |

Rules for the HTML:
- Keep the old wording; fix only obvious typos (`Категори` → `Категория`, `двуместный` → `двухместный`,
  `присвоениии` → `присвоении`, `в праве` → `вправе`, `указаную` → `указанную`, `бронирование` → `бронировании` where
  the grammar needs it, `Кпаснодарский` → `Краснодарский`, `пр едмет` → `предмет`).
- Prices as `N NNN ₽` with a normal space (the site adds non-breaking spaces). The breakfast delivery fee is the
  placeholder `{{breakfastDelivery}}`, as in `hotel.extras.breakfast-note`. Do not hard-code other prices that
  exist as placeholders (see `checkPlaceholders` for the allowed names).
- No `&nbsp;`, no inline styles, no `<table>`, no headings.
- Escape `'` as `''`.

### Rooms
`UPDATE rooms SET description = '<html>', updated_at = now() WHERE id = '<id>';`

| id | Text |
|---|---|
| `single-balcony` | `<p>Одноместный номер с двуспальной кроватью и балконом. Площадь 23 м² + балкон 7 м².</p>` + the shared part |
| `single` | `<p>Одноместный номер с двуспальной кроватью. Площадь 19 м².</p>` + the shared part |
| `double-twin` | `<p>Двухместный номер с двумя односпальными кроватями, их можно сдвинуть. Площадь 23–25 м².</p>` + the shared part |
| `double-euro` | `<p>Двухместный номер с двуспальной кроватью «евро». Площадь 23–25 м².</p>` + the shared part |
| `vip` | `<p>Двухкомнатный номер с балконом. Площадь 45 м² + балкон 7 м².</p>`, `<p><strong>Гостиная:</strong></p>` + `<ul>` of its items, `<p><strong>Спальня:</strong></p>` + `<ul>`, `<p><strong>Санузел:</strong> душевая кабинка, фен, туалетные принадлежности.</p>` |

The shared part (old «Общая комплектация номеров»): `<ul>` with сплит-система, Smart TV, сейф, холодильник,
шкаф-гардероб, прикроватные тумбы, журнальный столик, пуф, комод, внутренний телефон (capitalised, no `;`), then
`<p><strong>Санузел:</strong> душевая кабинка, фен, туалетные принадлежности.</p>`.

Do not change `photo_url`: the old site has no own photos of the single and «евро» rooms (the owner will send them
and upload them in the admin).

### Down
Restore the previous values: `text_value = ''`, `max_length` as in 00010 for the five keys; `description = ''` for the
four rooms; the VIP description as in 00005.

## Tests to write
In `internal/content/old_site_texts_test.go`, with `testutil.NewDB`:
- `TestOldSiteGuestInfoFilled`: for keys `hotel.guest-info.2.text` … `6.text`, `text_value` is not empty and
  `validateValue` accepts the stored value unchanged (sanitizing it returns the same string).
- `TestOldSiteGuestInfoContent`: `hotel.guest-info.6.text` contains «Халат — 2 000 ₽»; `3.text` contains
  `{{breakfastDelivery}}` and «Сырники».
- `TestOldSiteRoomDescriptions`: every row of `rooms` has a non-empty `description` that equals
  `richtext.Sanitize(richtext.Short, description)`.
No test for the down step: tests share one database, and running a migration down there would break other packages.
Check the Down section by reading it against 00005 and 00010.

## Done when
The commands in CONVENTIONS "Done means" pass, and with `make run`,
`curl -s 'localhost:3000/api/v1/content/blocks?page=hotel' | grep -c 'Халат'` prints 1.
Write under the task line in PROGRESS.md: «Owner: confirm the guest information is current (rules ПП № 1085 are from
2015, the category certificate in the old text expired in 2019).»

## Out of scope
The site's static fallback `blocks.data.ts` (citypark-modern task 23), room photos, new block keys, the admin.
