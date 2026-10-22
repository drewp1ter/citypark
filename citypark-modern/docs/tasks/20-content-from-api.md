# Task 20: Content blocks and hotel prices from the backend

## Goal
`getPageContent(page)` reads citypark-go `GET /content/blocks?page=…` and `getHotelPrices()` reads
`GET /hotel/prices`, so texts, photos and hotel prices edited in the admin show on the site within a minute. When the
backend fails, the pages fall back to the static data and stay readable.

## Depends on
16, 17, and citypark-go tasks 50, 52 and 54.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/content/` (all files), `src/entities/hotel/api/get-hotel-prices.ts`,
`src/entities/hotel/model/prices.data.ts`, `src/entities/room/api/rooms-api.ts` (fetch pattern),
`../citypark-go/docs/API.md` (sections `### Content` and `### Hotel`).

## Background
The backend's blocks are a copy of `blocks.data.ts` (citypark-go task 54), so with an untouched database the site
must look exactly as before. Unlike rooms and news, the static files stay as the **fallback**: a page with missing copy
is worse than slightly old copy.

## Files to create
- `src/entities/content/api/content-api.ts`, `src/entities/hotel/api/prices-api.ts`

## Files to change
- `src/entities/content/api/get-page-content.ts`: fetch, map, fall back (Spec 1).
- `src/entities/hotel/api/get-hotel-prices.ts`: fetch, fall back (Spec 2).

## Spec

### 1. Content
`GET /content/blocks?page=<page>` with `{ next: { revalidate: 60 } }`. Map each API block to the `Block` type:
`text`/`richtext` → `value` string; `lines` → `value` array; `image` → `image: { src: url, alt, position: position ||
undefined }`. Build `PageContent` over the API blocks. A key the site asks for that the API does not have → use the
static block with that key (log `content block missing in API: <key>` once). The whole request fails → log once and use
the static blocks for that page.

### 2. Hotel prices
`GET /hotel/prices` (revalidate 60) → `HotelPrices` (`shortName`, `desktopOnly` as sent). On error → log once, return
the static `prices.data.ts`.

## Tests to write
No unit-test runner. With citypark-go running and tasks 50, 52, 54 done:
- `/`, `/hotel`, `/rooms` at 390 and 1440 px look exactly as before;
- ask the user to change, through the admin or `PUT` with an admin token: the home hero lead, the smoking fine
  (3000), and the hotel park photo → after 60 s the new text, «3 000 ₽» in both places, and the new photo;
- backend stopped → the pages show the static copy; the log has one error per page render.

## Done when
CONVENTIONS "Done means" passes; on a production build `node scripts/qa/flows/home.mjs`,
`node scripts/qa/flows/hotel.mjs`, `node scripts/qa/flows/rooms.mjs` and `npm run qa:a11y` print no unexpected values.

## Out of scope
New blocks for other pages, on-demand revalidation, removing the static fallback files.
