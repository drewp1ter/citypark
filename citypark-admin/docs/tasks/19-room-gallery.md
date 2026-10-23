# Task 19: Room photo gallery

## Goal
The room edit page has a «Галерея» section where the owner adds, removes and reorders the room's extra photos (with
alt text). The website shows them on the room page.

## Depends on
11, 18, and citypark-go task 57.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/features/rooms/model/form.ts`, `src/features/rooms/ui/room-form.tsx`,
`src/shared/ui/image-field.tsx`, `e2e/rooms.spec.ts`, `e2e/image-field.spec.ts`,
`../citypark-go/docs/API.md` (section `### Rooms`, field `gallery`).

## Background
citypark-go task 57 added `gallery: { url, alt }[]` (at most 12 items, alt at most 200 characters) to `GET /rooms` and
`PUT /rooms/{id}`. `PUT` replaces all editable fields, so the admin must always send `gallery`, or saving a room
clears it. The VIP room is seeded with three photos from the old site (site paths `/images/room-vip-*.jpg`); the
`ImageField` already previews site paths for the main photo. The backend deletes uploads that are no longer used after
a save.

## Files to change
- `src/shared/api/schema.d.ts`: regenerate with `npm run api:gen`.
- `src/features/rooms/model/form.ts`: `gallery` in `RoomForm`, `RoomData`, `toForm`, `toRequest`, `validateRoom`,
  `RoomErrors`, `mapServerFields`.
- `src/features/rooms/model/form.test.ts`.
- `src/shared/ui/image-field.tsx`: only if needed for the prop below.
- `src/features/rooms/ui/room-form.tsx`.
- `e2e/rooms.spec.ts`.

## Spec
- `form.ts`: `const MAX_GALLERY = 12`. `RoomForm.gallery: { url: string; alt: string }[]`. `toRequest` trims alts and
  keeps the order. `validateRoom`: more than 12 → `errors.gallery = 'Не больше 12 фото'`; per item, empty url →
  `'Загрузите фото'`, alt longer than 200 → `'Не больше 200 символов'`, stored as
  `errors.galleryItems: Record<number, { url?: string; alt?: string }>` (or a similar shape that the form can read by
  index). `mapServerFields` maps backend keys `gallery` and `gallery.N.url|alt` into the same shape.
- `image-field.tsx`: add an optional prop `withPosition` (default `true`). When `false`, the focal-point picker is not
  shown and `onChange` always sends `position: ''`. Existing callers do not change.
- `room-form.tsx`: a section «Галерея» right after the main «Фото» field, with help «Дополнительные фото для страницы
  номера на сайте, в этом порядке». One row per photo: `ImageField` with `withPosition={false}`, `aspect={3 / 2}`
  (the old photos are 1280×853), label «Фото N», and three buttons «Выше», «Ниже», «Удалить» (HeroUI v3 `Button`, read
  its docs; «Выше» is disabled on the first row, «Ниже» on the last). Under the list a «Добавить фото» button adds an
  empty row; it is disabled at 12 rows. Gallery-level error text is shown under the list. All changes go through the
  same form state, so the unsaved-changes blocker sees them.
- The value sent to the backend for a gallery item is `{ url, alt }` (no `position`).

## Tests to write
- `form.test.ts`: round trip `toRequest(toForm(room))` keeps gallery items and order; 13 items → `'Не больше 12 фото'`,
  12 → no error; empty url in item 1 → error at index 1; alts trimmed; `mapServerFields({ 'gallery.0.url': 'x' })`
  → the error at index 0.
- `e2e/rooms.spec.ts`: open «VIP-номер» → the gallery shows 3 photos; open «Одноместный», add a photo (use the upload
  fixture from `image-field.spec.ts`), save, reload → 1 photo, and «Выше» and «Ниже» are
  disabled; remove it, save, reload → 0 photos. Leave the data as it was.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/rooms.spec.ts e2e/image-field.spec.ts`.

## Out of scope
The backend (citypark-go task 57), the website room page (citypark-modern task 25), drag-and-drop reordering,
uploading several files at once.
