# Task 11: Rooms

## Goal
«Номера» lists the five hotel room types with their prices and photos; an admin edits a room's texts, price per night,
description and photo. This is the only place room prices are edited.

## Depends on
08, 10, and citypark-go task 48.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/ui/image-field.tsx`, `src/shared/ui/rich-text-editor.tsx`,
`../citypark-go/docs/API.md` (section `### Rooms`).

## Background
The five room types are fixed (they match the hotel's free-room counts), so there is no add or delete. `PUT /rooms/{id}`
replaces all editable fields. The site shows the room price on the rooms page, the hotel page and the home page, and
computes «от N ₽» from the lowest price.

## Files to create
- `src/features/rooms/api.ts`, `src/features/rooms/model/form.ts` (+ test), `src/features/rooms/ui/rooms-page.tsx`,
  `ui/room-form.tsx`, `src/routes/_app/content.rooms.$roomId.tsx`, `e2e/rooms.spec.ts`

## Files to change
- `src/routes/_app/content.rooms.tsx`

## Spec
List: one card per room in API order: photo (4:3), name, `numbers`, `formatRub(pricePerNight)` «/ ночь», «Изменить».
Edit page `/content/rooms/$roomId` («← Номера» back link), sections:
- «Основное»: «Название» (`name`, required, ≤ 100), «Название в списке на странице отеля» (`hotelName`, ≤ 100, help
  «Если отличается»), «Номера комнат» (`numbers`, required, e.g. «№ 204, 304»), «Площадь» (`area`, required, «23 + 7 м²»),
  «Площадь подробно» (`areaDetailed`, help «Для десктопа: «45 м² + балкон 7 м²»»), «Вместимость» (`capacity`,
  «до 4 гостей»), «Строка на странице отеля» (`hotelNote`).
- «Цена»: «Цена за ночь, ₽» (number, required, > 0, ≤ 1 000 000, 2 decimals) with help «Одна цена на весь год. На
  главной «от» считается автоматически.».
- «Описание»: `RichTextEditor` profile `short`, `maxLength` 1000.
- «Фото»: `ImageField`, aspect 4/3, alt required in the UI.
Read-only info: type (`kind`), balcony yes/no. Save → `PUT`; field errors mapped (`photo.url` → the image field's
`url`, `description` → the editor); success → toast «Сохранено», stay on the page. Leaving with unsaved changes asks
«Есть несохранённые изменения. Уйти без сохранения?» (TanStack Router's blocker).
`form.ts`: `toForm(room)`, `toRequest(form)` (trims, price string → number), `validateRoom(form)`, `mapServerFields`.

## Tests to write
- `form.test.ts`: round trip `toRequest(toForm(room))` equals the editable fields; price `4 500` and `4500,5` parse;
  empty `name` / `numbers` / `area` → errors; price 0 → error.
- `e2e/rooms.spec.ts`: open «Одноместный», change the price to a test value and save → the list shows it; reload →
  kept; put the original price back at the end. Bold a word in the description, save, reload → still bold. Navigate
  away with a change → the confirm dialog appears.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/rooms.spec.ts`.

## Out of scope
Adding or removing room types, seasonal prices, galleries, free-room counts (task 06).
