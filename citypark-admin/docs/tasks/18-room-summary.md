# Task 18: Room summary field

## Goal
The room edit page has a «Кратко» field for the room's short plain-text `summary`, which the website shows on the room
card. The full rich-text description stays as it is.

## Depends on
11, and citypark-go task 56.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/features/rooms/model/form.ts`, `src/features/rooms/ui/room-form.tsx`,
`../citypark-go/docs/API.md` (section `### Rooms`, field `summary`).

## Background
citypark-go task 56 added `summary` (plain text, optional, at most 200 characters) to `GET /rooms` and `PUT /rooms/{id}`.
`PUT` replaces all editable fields, so the admin must send `summary`, or saving a room clears it.

## Files to change
- `src/shared/api/schema.d.ts`: regenerate with `npm run api:gen`.
- `src/features/rooms/model/form.ts`: `summary` in `RoomForm`, `RoomData`, `toForm`, `toRequest` (trimmed),
  `validateRoom`, `TEXT_FIELDS`.
- `src/features/rooms/model/form.test.ts`.
- `src/features/rooms/ui/room-form.tsx`.

## Spec
- `form.ts`: `const MAX_SUMMARY = 200`. `validateRoom`: if `length(f.summary.trim()) > MAX_SUMMARY`, then
  `errors.summary = 'Не больше 200 символов'`. Add `'summary'` to the `TextField` union only if that keeps the existing
  `MAX_TEXT` loop unchanged; otherwise widen `RoomErrors` with `'summary'` explicitly. `mapServerFields` maps the backend key
  `summary` to `errors.summary`.
- `room-form.tsx`: in the «Описание» section, before the rich-text editor, add a multi-line text field (HeroUI v3
  `TextField` with a `TextArea`; read its docs) labelled «Кратко», 2 rows, with help «Одна-две фразы для карточки на
  странице «Номера». Если пусто — на карточке нет текста.» and the error from `errors.summary`. It uses the same
  `set('summary')` pattern as the other fields, so the unsaved-changes blocker sees it.

## Tests to write
- `form.test.ts`: the round trip `toRequest(toForm(room))` includes `summary`; `summary` of 201 characters → error
  «Не больше 200 символов», 200 → no error; surrounding spaces are trimmed by `toRequest`; `mapServerFields({ summary: 'x' })`
  → `{ summary: 'x' }`.
- `e2e/rooms.spec.ts`: open «VIP-номер», change «Кратко», save, reload → kept. Put the original text back at the end.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/rooms.spec.ts`.

## Out of scope
The backend (citypark-go task 56), the website card (citypark-modern task 24), changes to the description editor.
