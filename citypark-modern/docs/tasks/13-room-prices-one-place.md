# Task 13: Room prices from one place

## Goal
Every room price on the site comes from `room.pricePerNight`. The home hero's «от 4 000 ₽ / ночь» shows the lowest
price of all rooms, and the home «Двухместный» card shows «от» the lower of the two double rooms. Changing one price in
`rooms.data.ts` (later in the admin) changes every place that shows it.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/home/ui/hero.tsx`, `src/pages/home/ui/rooms-section.tsx`,
`src/pages/home/ui/home-page.tsx`, `src/entities/room/index.ts`, `src/entities/room/model/rooms.data.ts`,
`src/shared/lib/format.ts`.

## Background
Decided by the owner (ADR 0001, `../docs/adr/0001-admin-panel.md`): a price is a number stored once. Today `hero.tsx`
has the string `'от 4 000 ₽'` hardcoded, and the «Двухместный» home card reads only `double-twin`, so a different
`double-euro` price would be wrong there. The rooms page, hotel page and booking total already use `pricePerNight`.

## Files to change
- `src/entities/room/model/` (new file `prices.ts`) and `src/entities/room/index.ts`: two pure functions (Spec 1).
- `src/pages/home/ui/hero.tsx`: the hotel tile caption from the lowest price (Spec 2).
- `src/pages/home/ui/rooms-section.tsx`: teasers name a group of room ids (Spec 3).
- `src/pages/home/ui/home-page.tsx`: pass `rooms` to `Hero`.

## Spec

### 1. Prices
```ts
/** Lowest price per night of the given rooms, or of all rooms when `ids` is omitted. `null` for an empty list. */
export function minPricePerNight(rooms: readonly Room[], ids?: readonly string[]): number | null
```
Export it from `@/entities/room`.

### 2. Hero
`Hero` takes `{ rooms: readonly Room[] }`. Caption: mobile `от ${formatPrice(min)}`, desktop
`от ${formatPrice(min)} / ночь`, where `min = minPricePerNight(rooms)`. If `min` is `null`, the caption is
`Номера` (mobile) / `12 номеров «Люкс»` (desktop). With today's data the output is identical to now (`от 4 000 ₽`).

### 3. Home room cards
`Teaser.roomId: string` becomes `roomIds: readonly string[]`: «Одноместный» → `['single', 'single-balcony']`,
«Двухместный» → `['double-twin', 'double-euro']`, «VIP-номер» → `['vip']`. The price shown is
`minPricePerNight(rooms, teaser.roomIds)`. Show «от» when the group has more than one id **and** their prices differ,
or when `from` was already `true` (keep the current output for today's data: «Одноместный» shows «от 4 000 ₽», the
others show the plain price). Remove the `from` flag if the rule above makes it unnecessary and the output stays the
same; check today's output against the reference first.

## Tests to write
No unit-test runner. On a production build:
- `/` looks exactly as before (hero caption, three room cards) at 390 and 1440 px;
- temporarily set `double-euro` `pricePerNight` to 4200 in `rooms.data.ts` → the «Двухместный» card shows
  «от 4 200 ₽»; set `single` to 3900 → the hero shows «от 3 900 ₽ / ночь». Revert both before committing.

## Done when
CONVENTIONS "Done means" passes and `node scripts/qa/flows/home.mjs` prints no unexpected values.

## Out of scope
Rooms from the backend (task 17), hotel fees and breakfast prices (task 15).
