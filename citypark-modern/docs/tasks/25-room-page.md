# Task 25: Room pages with a gallery

## Goal
Each room type has its own page `/rooms/{id}` with the main photo, a photo gallery, the full description, the 3D tour,
the price and a booking button, as the old site had (`/hotel/katalog-nomerov/vip` and others). The room cards on
`/rooms` and the room list on `/hotel` link to these pages.

## Depends on
17, 22, 24, and citypark-go task 57.

## Read first
`docs/tasks/CONVENTIONS.md`, `CLAUDE.md`, `src/entities/room/` (all files), `src/pages/rooms/ui/room-card.tsx`,
`src/pages/rooms/ui/rooms-page.tsx` (error handling), `src/pages/hotel/ui/room-fund.tsx`,
`app/(site)/news/[slug]/page.tsx`, `src/pages/news-article/ui/article-page.tsx` (detail page pattern),
`src/shared/ui/breadcrumbs.tsx`, `src/shared/ui/rich-text.tsx`, `scripts/qa/flows/rooms.mjs`,
`../citypark-go/docs/API.md` (section `### Rooms`, field `gallery`).
For Next.js: dynamic route params, `generateMetadata` and `notFound` in `node_modules/next/dist/docs/`.

## Background
The backend already has the old site's full equipment text in each room's `description` (citypark-go task 55), but
nothing on the site shows it since task 24 (cards show `summary`). citypark-go task 57 added `gallery` and seeded the
VIP room with three photos whose files this task adds. The old site's room pages (`/workspace/citypark_frontend/src/
features/Hotel/components/{Single,Double,Vip}Room/*.js`, read only) are the content reference: every room type has
a description and a 3D tour (`room_1`, `room_2`, `vip_1`, already in `public/tour`; `vip_2`, `lodge` and `bathroom`
are reached through hotspots inside the tours), and only VIP has a gallery; there is no design mock for this
page, so build it from tokens and existing `shared/ui` parts, in the style of the article page.

## Files to create
- `public/images/room-vip-living.jpg`, `room-vip-bathroom.jpg`, `room-vip-balcony.jpg`: copy byte for byte (`cp`)
  from `/workspace/citypark_frontend/src/features/Hotel/components/VipRoom/images/image1.jpg`, `image2.jpg`,
  `image3.jpg` (living room, bathroom, balcony).
- `app/(site)/rooms/[id]/page.tsx`: re-exports only, like the news article route.
- `src/pages/room/index.ts`, `src/pages/room/ui/room-page.tsx` (`RoomPage`, `generateRoomMetadata`),
  `src/pages/room/ui/room-gallery.tsx` (client component, the only one in the slice).

## Files to change
- `src/entities/room/api/rooms-api.ts`: `gallery: { url: string; alt: string }[]` in `ApiRoom`.
- `src/entities/room/model/types.ts`: `gallery: Photo[]` on `Room`, comment «Дополнительные фото для страницы номера».
- `src/entities/room/api/get-rooms.ts`: map `gallery` (`{ src: url, alt }`, `?? []` for safety); add
  `getRoom(id: string): Promise<Room | undefined>` built on `getRooms()`.
- `src/entities/room/index.ts`: export `getRoom`.
- `src/pages/rooms/ui/room-card.tsx`, `src/pages/hotel/ui/room-fund.tsx`.
- `scripts/qa/flows/rooms.mjs`.

## Spec

### Route and data
- `app/(site)/rooms/[id]/page.tsx`: `generateMetadata` → `generateRoomMetadata(id)`, page → `<RoomPage id={id} />`.
- `generateRoomMetadata`: title `${room.name} — Сити Парк`, description `room.summary` when present; unknown room →
  «Страница не найдена».
- `RoomPage`: `getRoom(id)`; `undefined` → `notFound()`. If the backend fails, handle it the same way `rooms-page.tsx`
  does (log, then the same fallback or error the rooms page shows); do not invent a new pattern.

### Page content (top to bottom), `max-w-page` container like the article page
1. `Breadcrumbs` with «Номера» (`/rooms`) → room name.
2. `h1` room name (`heading` class); line `numbers · (areaDetailed ?? area) · capacity` in `text-text-muted`.
3. Two columns on desktop (photo left, details right), one column on mobile:
   - main photo with `next/image` (`fill`, eager, `fetchPriority='high'`, `object-position` from the room), rounded
     like the cards;
   - details: price `formatPrice(pricePerNight)` «/ ночь» (same type styles as the VIP card), button «Забронировать»
     → `/rooms#search`, link «3D-тур ↗» (`roomTourHref(room.kind)`, new tab, the sr-only note as in `TourLink`).
4. Gallery (only when `gallery.length > 0`): heading «Фото», a grid of thumbnails (2 columns on mobile, 3 on desktop,
   3:2 aspect, `next/image` with `sizes`). Each thumbnail is a `<button>` with the photo's alt as its accessible name.
   Clicking opens `RoomGallery`'s lightbox: a native `<dialog>` (`showModal`) with the full photo, «Закрыть», «Назад»,
   «Вперёд» buttons and the counter «N из M»; Esc closes, ← and → switch photos, focus goes back to the thumbnail on
   close. The main photo is item 0 of the lightbox, followed by the gallery.
5. Description: `RichText html={descriptionHtml} variant='article'`; if empty, nothing.
6. A short paragraph: «Правила проживания, дополнительные услуги и документы — на странице отеля» with a link to
   `/hotel#info` (the guest information section).
Colours only from tokens; check light and dark.

### Links in
- `room-card.tsx`: in both card types the room name (`h2`) becomes an `AppLink` to `/rooms/${room.id}` (keep the text
  styles, add a hover underline). The VIP card's photo also links there. Do not add a separate «Подробнее» button:
  the cards must keep their current height (task 24).
- `room-fund.tsx`: each list item links to `/rooms/${room.id}` instead of `/rooms`. The header link «Фото и бронь ↗»
  stays `/rooms`.

## Tests to write
No unit-test runner. Extend `scripts/qa/flows/rooms.mjs` (same output style: expected next to real):
- `/rooms`: clicking the VIP name opens `/rooms/vip`.
- `/rooms/vip`: `h1` «VIP-номер»; 3 gallery thumbnails; the description shows «Камин электрический»; clicking the
  first thumbnail opens the dialog with «2 из 4»; → shows «3 из 4»; Esc closes it and focus is on the thumbnail.
- `/rooms/single`: no «Фото» section; the description shows «Сплит-система» (the old site's text from citypark-go
  migration 00011); the «3D-тур» link points to `/tour/room_1.html`.
- `/rooms/double-twin`: the description renders; the «3D-тур» link points to `/tour/room_2.html`.
- `/rooms/nope`: status 404.
Then by hand on a production build: `/rooms/vip` at 390 px and 1440 px, light and dark; `/hotel` list items open their
room pages.

## Done when
CONVENTIONS "Done means" passes; `node scripts/qa/flows/rooms.mjs`, `npm run qa:a11y`, `npm run qa:keyboard` and
`npm run qa:responsive` pass on the build with the backend running (add `/rooms/vip` to their page lists if they
use a fixed list).

## Out of scope
The backend and the admin (citypark-go task 57, citypark-admin task 19), 301 redirects from the old URLs, embedding the
3D tour in the page, booking engine integration, social share buttons.
