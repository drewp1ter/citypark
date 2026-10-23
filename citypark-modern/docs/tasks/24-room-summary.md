# Task 24: Short room text on the rooms page

## Goal
The VIP card on `/rooms` shows the room's short `summary` instead of its full description, so it is as tall as in the
design again and the grid has no empty space.

## Depends on
17, 23, and citypark-go task 56.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/rooms/ui/room-card.tsx`, `src/entities/room/api/get-rooms.ts`,
`../citypark-go/docs/API.md` (section `### Rooms`, field `summary`).

## Background
citypark-go task 55 filled the room descriptions with the old site's full equipment text (two lists, about 20 items for
VIP). `FeaturedRoomCard` renders `descriptionHtml` on desktop, so the VIP card grew to about 1000 px. The grid row
stretched with it, and so did the «Одноместный с балконом» card beside it: a large empty space, with the price and
«Забронировать» pushed far down. citypark-go task 56 added a plain-text `summary` (the VIP one is the design's
«Гостиная и спальня, расширенная комплектация, балкон над кронами парка.»; the others are empty).

## Files to change
- `src/entities/room/api/rooms-api.ts`: `summary: string` in the backend room type.
- `src/entities/room/model/types.ts`: `summary?: string` on `Room`, with a comment «Одна-две фразы для карточки».
- `src/entities/room/api/get-rooms.ts`: `summary: typo(room.summary) || undefined`.
- `src/pages/rooms/ui/room-card.tsx`.

## Spec
- In `FeaturedRoomCard`, replace the `RichText` block with
  `{room.summary && <p className='hidden flex-1 md:block'>{room.summary}</p>}` (the same place and classes, so the price
  and button stay at the bottom). Remove the `RichText` import if nothing else in the file uses it.
- Small cards stay as in the design, with no text.
- Keep `descriptionHtml` in the entity. Nothing on `/rooms` renders it now; a room page may use it later.
- If any other place renders `descriptionHtml` on a card (`grep -rn descriptionHtml src`), switch it to `summary` the
  same way and note it in PROGRESS.

## Tests to write
No unit-test runner. On a production build with the backend running:
- `/rooms` at 1440 px, light and dark: the first row (VIP + «Одноместный с балконом») is about 400 px tall, as in
  `design/screenshots/{light,dark}/desktop/rooms.jpg`; the VIP card shows the teaser, «8 000 ₽ / ночь» and
  «Забронировать · …» without scrolling; the second row starts right below.
- `/rooms` at 390 px: unchanged (the teaser is hidden on mobile, as before).
- Change the VIP summary in the admin (or ask the user to), revalidate, reload → the new text shows.

## Done when
`npm run check` is clean, `npm run build` passes, and `npm run qa:flows` and `npm run qa:responsive` pass on the
build. Then the checks above.

## Out of scope
Showing the full description (a room page or a disclosure is the owner's decision), the backend and the admin.
