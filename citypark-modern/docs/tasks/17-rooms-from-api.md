# Task 17: Rooms from the backend

## Goal
`getRooms()` reads the five rooms from citypark-go `GET /rooms` instead of `rooms.data.ts`, so a price, text or photo
changed in the admin shows on `/rooms`, `/hotel` and `/` within a minute. Uploaded photos load through `next/image`.

## Depends on
13, 15, and citypark-go tasks 45 and 48.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/room/api/get-rooms.ts`, `src/entities/room/api/hotel-api.ts`,
`src/entities/room/model/types.ts`, `src/entities/room/model/rooms.data.ts`, `src/shared/config/env.ts`,
`next.config.ts`, `src/pages/rooms/ui/room-card.tsx`, `src/shared/ui/rich-text.tsx`,
`../citypark-go/docs/API.md` (sections `### Rooms` and `### Uploads`).

## Background
The backend seeded the rooms with the values of `rooms.data.ts` (citypark-go task 48), with two differences: plain
texts use normal spaces (the site adds non-breaking ones with `typo()`), and `description` is HTML (`<p>…</p>`).
Photo URLs are either site paths (`/images/room-vip.jpg`, from `public/`) or uploaded images
(`http://<backend>/uploads/2026/10/<name>.jpg`).

## Files to create
- `src/entities/room/api/rooms-api.ts`: `fetchRooms()` and the API type.

## Files to change
- `src/entities/room/api/get-rooms.ts`: `getRooms()` maps the API (Spec 1).
- `src/entities/room/model/types.ts`: `description` is HTML (`descriptionHtml: string`, `''` when empty); update the
  doc comment.
- `src/entities/room/model/rooms.data.ts`: delete it.
- `src/pages/rooms/ui/room-card.tsx` (and any other place that renders `description`): `<RichText variant='short'>`.
- `src/shared/config/env.ts`, `.env.example`, `next.config.ts`: `UPLOADS_ORIGIN` (Spec 2).
- Pages that call `getRooms()` (`grep -rn "getRooms()" src`): wrap the call like `HomePage` does for the menu
  (`.catch((error: unknown) => { console.error(error); return null })`) and handle `null` (Spec 3).

## Spec

### 1. Mapping
`GET /rooms` with `{ next: { revalidate: 60 } }`; non-OK → throw `GET /rooms: <status>`.
`Room` fields: `id`, `name`, `hotelName` (`''` → `undefined`), `kind`, `balcony`, `numbers`, `area`, `areaDetailed`,
`capacity` (`''` → `undefined`), `descriptionHtml`, `pricePerNight`, `photo` (`{ src: url, alt, position: position ||
undefined }`), `hotelNote`. Apply `typo()` to `numbers`, `area`, `areaDetailed`, `hotelNote` (as the data file did).
Keep the API's order.

### 2. Uploaded photos
`UPLOADS_ORIGIN` (optional, e.g. `http://localhost:3000`): the backend's `PUBLIC_BASE_URL`. In `next.config.ts` add a
`remotePatterns` entry `{ protocol, hostname, port, pathname: '/uploads/**' }` built from it, next to the existing
`MENU_IMAGE_HOST` entry (keep that one). Empty → no entry. Read the `next/image` docs in `node_modules/next/dist/docs/`
for the current `remotePatterns` format.

### 3. Backend down
`/rooms`: show the page heading and the message used by the menu page for an unavailable backend (reuse
`menu-error.tsx`'s wording or component if it is generic; otherwise «Не удалось загрузить номера. Позвоните нам:» and
the hotel phone). `/hotel` «Номерной фонд» and the home rooms section: hide the room list, keep the rest of the page.
The hero «от … ₽» caption falls back as task 13 says.

## Tests to write
No unit-test runner. With citypark-go running (tasks 45 and 48 done):
- `/rooms`, `/hotel`, `/` look the same as before at 390 and 1440 px;
- ask the user to change the VIP price to 9000 in the database or the admin, wait 60 s → all three pages show
  «9 000 ₽» (VIP card and hero «от» unchanged since VIP is not the cheapest);
- ask the user to upload a photo for a room → it loads (no `next/image` host error);
- backend stopped → the pages render as in Spec 3, one error per request in the server log.

## Done when
CONVENTIONS "Done means" passes; on a production build `node scripts/qa/flows/rooms.mjs`,
`node scripts/qa/flows/hotel.mjs`, `node scripts/qa/flows/home.mjs` and `npm run qa:a11y` print no unexpected values.

## Out of scope
Room availability (already from the backend), booking, the admin.
