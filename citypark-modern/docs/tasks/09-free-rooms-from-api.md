# Task 09: Free rooms from the backend

## Goal
The free rooms count on the home page (task 08) comes from citypark-go `GET /hotel/rooms` instead of sample data.

## Depends on
08, and citypark-go task 41 (the endpoint must exist on the backend you run against).

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/room/api/get-rooms.ts`, `src/entities/dish/api/menu-api.ts` (fetch
pattern), `../citypark-go/docs/API.md` (section `### Hotel`).

## Background
- Task 08 added `getFreeRoomsCount(): Promise<number | null>`, which sums the sample `getRoomAvailability()`.
  `HomePage` calls it with `.catch(...)` that logs and returns `null`; `null` hides the counter.
- The backend stores free rooms for each of the five website room types, set by the hotel's webhook.
  `GET /hotel/rooms` returns
  `{"rooms": {"vip": 1, "singleBalcony": 0, "single": 2, "doubleTwin": 1, "doubleEuro": 0}, "free": 4, "updatedAt": "…"}`,
  or **404** while no webhook has arrived yet. The keys are the website room ids in camelCase
  (`single-balcony` → `singleBalcony`).
- The counts are for today, not for a date range. `/rooms` asks `getRoomAvailability()` for the dates the guest chose
  and labels a missing room «Нет на эти даты», so it stays sample data in this task (it needs date-aware data).
- `MENU_API_URL` is the citypark-go base URL (`…/api/v1`). Use it for this call too; do not add a second variable.

## Files to create
- `src/entities/room/api/hotel-api.ts`: `ApiHotelRooms`, `fetchHotelRooms` (Spec 1).

## Files to change
- `src/entities/room/api/get-rooms.ts`: `getFreeRoomsCount` reads the API (Spec 2); a comment on
  `getRoomAvailability` saying why it is still sample data.
- `scripts/qa/flows/home.mjs`: new expected values (Spec 3).

## Spec

### 1. `hotel-api.ts`
```ts
export type ApiHotelRooms = {
  rooms: { vip: number; singleBalcony: number; single: number; doubleTwin: number; doubleEuro: number };
  free: number;
  updatedAt: string;
};

/** GET /hotel/rooms. `null` when the backend has no data yet (404). */
export async function fetchHotelRooms(): Promise<ApiHotelRooms | null>;
```
- `fetch(serverEnv().MENU_API_URL + "/hotel/rooms", { next: { revalidate: 60 } })`, same as `menu-api.ts`.
- 404 → `null`. Any other non-OK status → `throw new Error(\`GET /hotel/rooms: ${status}\`)`.
- Do not export it from the slice's `index.ts`; only the getter is public.

### 2. `getFreeRoomsCount`
- Returns `(await fetchHotelRooms())?.free ?? null`. Errors are not caught here (the page decides, as for the menu).
- Signature and export name stay the same, so `HomePage`, `RoomsSection` and `FreeRooms` do not change.
- Result: backend down → `HomePage` logs and hides the counter; 404 → hidden; `free: 0` → the zero-state line from
  task 08; otherwise the tile.

### 3. QA
Before `node scripts/qa/flows/home.mjs`, set known numbers on the backend:
```
curl -s -X POST localhost:3000/api/v1/hotel/rooms -H "X-Webhook-Secret: $HOTEL_WEBHOOK_SECRET" \
  -H 'Content-Type: application/json' -d '{"rooms":{"vip":1,"singleBalcony":0,"single":2,"doubleTwin":1,"doubleEuro":0}}'
```
Write this command in a comment at the top of the free-rooms checks in `home.mjs`, and change the expected values from
task 08 to `4 / свободных номера` and `Сегодня свободно 4 номера из 12`.

## Tests to write
There is no unit-test runner. Check by hand on a production build and report the results:
- the numbers above → tile «4», «свободных номера»;
- webhook with all zeros, wait 60 s or restart → zero-state line with the hotel phone;
- backend stopped → the home page renders, no counter, one error in the server log;
- empty `hotel_rooms` table (404) → no counter, no error in the log.

## Done when
CONVENTIONS "Done means" passes, and with citypark-go running (task 41 done, `HOTEL_WEBHOOK_SECRET` set) and a
production build on port 3111: `node scripts/qa/flows/home.mjs` and `npm run qa:a11y` print no unexpected values.

## Out of scope
The «Только свободные» filter and per-type availability on `/rooms`, availability per date, the webhook sender, and any
change in `../citypark-go/`.
