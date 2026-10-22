# Task 11: Room availability from the backend

## Goal
The status badges on `/rooms` («Свободно 2», «Свободен», «Нет на эти даты») and the «Только свободные» filter use
citypark-go `GET /hotel/rooms` instead of sample data, when the guest's check-in is today.

## Depends on
09, and citypark-go task 41.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/09-free-rooms-from-api.md`, `src/entities/room/api/get-rooms.ts`,
`src/entities/room/api/hotel-api.ts`, `src/entities/room/model/types.ts`, `src/pages/rooms/ui/rooms-page.tsx`,
`src/pages/rooms/ui/room-card.tsx`, `src/pages/home/ui/home-page.tsx` (error pattern), `scripts/qa/flows/rooms.mjs`,
`../citypark-go/docs/API.md` (section `### Hotel`).

## Background
- `fetchHotelRooms()` (task 09) returns `{ rooms: { vip, singleBalcony, single, doubleTwin, doubleEuro }, free, updatedAt }`
  or `null` on 404. The numbers are rooms free **today**, not for a date range.
- `getRoomAvailability(query)` returns sample data today. `RoomsPage` passes its result to `RoomCard`; `statusLabel`
  turns a missing entry into «Нет на эти даты».
- Decided by the product owner:
  - check-in is today (`todayInMoscow()`) → badges from the backend;
  - check-in is any other day → we do not know, so no badge, the card looks available («Выбрать»), and
    «Только свободные» filters nothing;
  - backend 404 or error → the same as "do not know". An error is logged once on the server; 404 is not an error.
- Room types with one room (`vip`, `single-balcony`) say «Свободен», not «Свободно 1» (as in the design).

## Files to change
- `src/entities/room/api/get-rooms.ts`: `getRoomAvailability` (Spec 1).
- `src/pages/rooms/ui/rooms-page.tsx`: error handling and the filter (Spec 2).
- `src/pages/rooms/ui/room-card.tsx`: neutral state (Spec 3).
- `scripts/qa/flows/rooms.mjs`: setup comment, new expected values, a future-date check (Spec 4).

## Spec

### 1. `getRoomAvailability`
```ts
/**
 * Availability by room id for the given stay, or `null` when it is not known.
 * The backend only knows today's counts, so any check-in other than today is `null`. Errors are left to the caller.
 */
export async function getRoomAvailability(query: StayQuery): Promise<Record<string, RoomAvailability> | null>
```
- `query.checkIn !== todayInMoscow()` → `null` without calling the backend.
- Otherwise `fetchHotelRooms()`; `null` → `null`.
- Map with one table in this file (website room id → API key → rooms of that type):
  `vip`→`vip` (1), `single-balcony`→`singleBalcony` (1), `single`→`single` (2), `double-twin`→`doubleTwin` (4),
  `double-euro`→`doubleEuro` (4).
- count `0` → `{ state: 'unavailable' }`; count > 0 and the type has 1 room → `{ state: 'available' }`;
  otherwise `{ state: 'available', left: count }`.
- Remove the sample numbers and the old TODO/comment about sample data. `query` is now required.
- Export name stays the same; `index.ts` does not change.

### 2. `RoomsPage`
- Wrap the call like `HomePage`: `.catch((error: unknown) => { console.error(error); return null })`.
- Filter: `!query.freeOnly || availability === null || availability[room.id]?.state === 'available'`.
- Pass `availability ? availability[room.id] : null` to `RoomCard` (see Spec 3 for the type).
- The «Только свободные» chip stays visible in every state.

### 3. `RoomCard`
- Prop type becomes `availability: RoomAvailability | null | undefined`:
  `null` = not known, `undefined` = known map without this room (stays «Нет на эти даты», as now).
- `null` → no `StatusBadge` (neither the mobile nor the desktop one on the VIP card), photo not greyscale, price not
  muted, button «Выбрать» (`variant='inverse'` on normal cards; VIP card unchanged apart from the badge).
- Simplest way: `const known = availability !== null` and `available = !known || availability?.state === 'available'`;
  render the badge only when `known`. `statusLabel` keeps its texts.

### 4. QA (`scripts/qa/flows/rooms.mjs`)
Before running, set known numbers on the backend; write this command in a comment at the top of the file:
```
curl -s -X POST localhost:3000/api/v1/hotel/rooms -H "X-Webhook-Secret: $HOTEL_WEBHOOK_SECRET" \
  -H 'Content-Type: application/json' -d '{"rooms":{"vip":1,"singleBalcony":0,"single":2,"doubleTwin":1,"doubleEuro":0}}'
```
Put the expected values in the `out(...)` labels, like the other checks:
- `statuses` (today) → `["Свободен","Нет на эти даты","Свободно 2","Свободно 1","Нет на эти даты"]`
  (VIP badge appears once on desktop; check the real output and keep the order of the page);
- `+ only free (balcony)` → only the VIP room; `double + free` → only the twin room;
- `bad params: cards (free only)` → 3;
- new check: `/rooms?checkIn=<plus(2)>&free=1` → 5 cards and no status badges.

## Tests to write
No unit-test runner. Check by hand on a production build and report:
- numbers above, `/rooms` → badges as in Spec 4;
- `/rooms` with check-in in 2 days → no badges, every card has «Выбрать»;
- backend stopped → page renders without badges, one error in the server log;
- empty `hotel_rooms` table (404) → no badges, no error in the log.

## Done when
CONVENTIONS "Done means" passes, and with citypark-go running (task 41 done, `HOTEL_WEBHOOK_SECRET` set) and a
production build on port 3111: `node scripts/qa/flows/rooms.mjs`, `node scripts/qa/flows/home.mjs` and
`npm run qa:a11y` print no unexpected values.

## Out of scope
Availability per date range, a booking engine, the «Выбрать» booking flow, the hotel page, and any change in
`../citypark-go/`.
