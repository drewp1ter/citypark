# Task 06: Free rooms (read only)

## Goal
«Свободные номера» shows how many rooms of each type the hotel's sender last reported as free, and when.

## Depends on
03.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/api/`, `../citypark-go/docs/API.md` (section `GET /hotel/rooms`).

## Background
The counts come from the hotel's external sender (webhook); the admin only displays them (owner's decision). 404 means
the sender has not posted yet.

## Files to create
- `src/features/free-rooms/api.ts`, `src/features/free-rooms/model/room-types.ts` (+ test),
  `src/features/free-rooms/ui/free-rooms-page.tsx`, `e2e/free-rooms.spec.ts`

## Files to change
- `src/routes/_app/free-rooms.tsx`

## Spec
`room-types.ts`: the five types in this order with label and total: `vip` «VIP-номер» 1, `singleBalcony`
«Одноместный с балконом» 1, `single` «Одноместный» 2, `doubleTwin` «Двухместный, 2 кровати» 4, `doubleEuro`
«Двухместный, кровать евро» 4.
Page: `PageHeader` «Свободные номера», a large number «Свободно: N из 12», a card per type «свободно X из Y» with a
progress bar, and «Обновлено: <formatDateTime(updatedAt)>». If `updatedAt` is older than 24 hours, a warning
«Данные давно не обновлялись — проверьте отправку из гостиницы». 404 → «Данных пока нет: гостиница ещё не присылала
количество свободных номеров.». Refetch every 60 s while the page is open.

## Tests to write
- `room-types.test.ts`: totals add up to 12; every API key has a label.
- `e2e/free-rooms.spec.ts`: the page renders either the counts or the «Данных пока нет» message (both are valid in a
  dev database); if counts are shown, the five labels are visible.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/free-rooms.spec.ts`.

## Out of scope
Editing the counts, history.
