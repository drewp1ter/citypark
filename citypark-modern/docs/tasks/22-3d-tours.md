# Task 22: 3D tours from the old site

## Goal
The old site's 3D panoramas (Pano2VR, HTML5) are served from `/tour/`, and the site links to them: the «3D-панорама»
tiles of the home and restaurant pages open the restaurant tour, and each room card has a «3D-тур» link.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/shared/ui/app-link.tsx`, `src/pages/restaurant/ui/hero.tsx`,
`src/pages/home/ui/restaurant-section.tsx`, `src/pages/rooms/ui/room-card.tsx`, `src/entities/room/index.ts`,
`src/entities/room/model/types.ts`.

## Background
The old CRA site (`/workspace/citypark_frontend`, read only) has `public/tour/` (about 46 MB): one HTML page and one
`*_out.xml` per panorama, `pano2vr_player.js` and `images/` (cube faces). Panoramas: `restaurant`, `kafe`,
`lodge` (lobby), `bathroom`, `room_1` (single room), `room_2` (double room), `vip_1`, `vip_2`. The old room pages
linked `room_1` (single), `room_2` (double) and `vip_1` (VIP). The pages are self-contained (no Flash, no external
scripts) and are not Next pages, so links to them must be plain `<a>`: `next/link` would try a client navigation.
Today the restaurant tiles link to `#tour` (no such anchor) and the home tile links to `/restaurant`.

## Files to create
- `public/tour/`: a copy of the old `public/tour/` (all files).
- `src/entities/room/model/tours.ts`: `roomTourHref(kind: RoomKind): string`.

## Files to change
- `public/tour/*.html`: replace the favicon `http://cityparkvip.ru/assets/templates/citypark/favicon.png` with
  `/icon.svg` (`type="image/svg+xml"`); change nothing else in the player files.
- `src/entities/room/index.ts`: export `roomTourHref`.
- `src/pages/restaurant/ui/hero.tsx`: both `href='#tour'` tiles → the restaurant tour.
- `src/pages/home/ui/restaurant-section.tsx`: the «3D-панорама зала» tile → the restaurant tour.
- `src/pages/rooms/ui/room-card.tsx`: the «3D-тур» link (Spec 2), on the regular and the VIP card.

## Spec

### 1. Tour links
`/tour/restaurant.html` for the restaurant. Every tour link is a plain `<a href=… target="_blank" rel="noopener">`
with the existing classes (for the tiles, keep `cardClass(...)` and the content). Add a visually hidden
« (откроется в новой вкладке)» to the accessible name (the `sr-only` class), as the tiles already end with «↗».

### 2. Room cards
```ts
// src/entities/room/model/tours.ts
const tours: Record<RoomKind, string> = { single: 'room_1', double: 'room_2', vip: 'vip_1' }
/** Old site's Pano2VR tour of a room type, served from public/tour. */
export function roomTourHref(kind: RoomKind): string
```
Returns `/tour/<name>.html`. In the card, after the «№ … · … м²» line, a text link «3D-тур ↗», `text-sm`, colour
`text-text-muted` with underline on hover, a touch target of at least 44 px high (padding, not a bigger font).
The link is not designed; keep it this small and use only tokens. Both themes.

### 3. Size
Copy the files as they are. Write under the task line in PROGRESS.md the size of `public/tour` (`du -sh`) so the owner
can decide later whether to move it to a CDN.

## Tests to write
No unit-test runner. On a production build (`npm run build && npm start -- -p 3111`):
- `curl -sI localhost:3111/tour/restaurant.html` → `200` and `content-type: text/html`; the same for one file in
  `tour/images/` and `tour/pano2vr_player.js`.
- Open `/tour/restaurant.html`, `/tour/room_1.html`, `/tour/vip_1.html` in Playwright (or a browser): the panorama
  draws, no console errors, no request to `cityparkvip.ru`.
- `/` and `/restaurant` at 1440 and 390 px: the tiles open the tour in a new tab; `/rooms`: each card's «3D-тур»
  opens the matching tour.

## Done when
CONVENTIONS "Done means" passes; `node scripts/qa/flows/rooms.mjs`, `node scripts/qa/flows/restaurant.mjs`,
`node scripts/qa/flows/home.mjs` and `npm run qa:a11y` print no unexpected values (update a flow's expected link if it
checked `#tour` or `/restaurant` for the panorama tile).

## Out of scope
The `kafe`, `lodge`, `bathroom`, `vip_2` tours get no links (the files are copied; the owner may ask for links later),
a new tour player, redirects from old URLs.
