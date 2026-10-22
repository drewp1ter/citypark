# Task 08: Free rooms count on the home page

## Goal
The «Номера» section of the home page shows how many hotel rooms are free today, like the old site
(«4 свободных номера»), in the design language of the new site.

## Depends on
None.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/home/ui/rooms-section.tsx`, `src/entities/room/api/get-rooms.ts`,
`src/shared/lib/index.ts` (look for an existing plural helper before writing one).

## Background
- The old site printed a big number and «свободных номера» under the room list. The new site has no such element.
- There is no booking engine yet. `getRoomAvailability()` in `entities/room/api/get-rooms.ts` returns sample data and
  is the seam where the real call goes later. The count must come from this seam, not from a second hardcoded number.
- Sample data today: `vip` available (no `left`), `single-balcony` available (no `left`), `single` left 2,
  `double-twin` left 1, `double-euro` unavailable. An available room type without `left` counts as 1.
- The hotel has 12 rooms in total (hero stat «12 номеров»); do not touch that stat.

## Files to create
- `src/pages/home/ui/free-rooms.tsx`: `FreeRooms` (Spec 2). Used only by the home page, so it stays in the page slice.

## Files to change
- `src/entities/room/api/get-rooms.ts`: `getFreeRoomsCount` (Spec 1).
- `src/entities/room/index.ts`: export `getFreeRoomsCount`.
- `src/pages/home/ui/home-page.tsx`: fetch the count with the other data and pass it down.
- `src/pages/home/ui/rooms-section.tsx`: accept `free` and render `FreeRooms` under the grid.
- `scripts/qa/flows/home.mjs`: Spec 4.

## Spec

### 1. `getFreeRoomsCount`
```ts
/** Rooms free tonight, summed over room types. `null` when availability is unknown. */
export async function getFreeRoomsCount(): Promise<number | null>;
```
- Calls `getRoomAvailability()` and sums: `state: "available"` adds `left ?? 1`, `"unavailable"` adds 0.
  With the sample data the result is **5**.
- Wrap nothing in try/catch here. In `HomePage` add `getFreeRoomsCount().catch(...)` to the existing `Promise.all`,
  logging with `console.error` and returning `null`, the same way `getMenu()` is handled there.

### 2. `FreeRooms({ count }: { count: number })`
A Server Component. One memorable element, everything else quiet.
- Layout: one row, left aligned, under the room cards, with the same gap as between the cards
  (`mt-3 md:mt-4`). No card, no border, no shadow.
- Left: the number in a square tile, `bg-accent text-on-accent`, `rounded-sm` (20 px) on desktop and `rounded-xs`
  (16 px) on mobile, `font-display font-medium` with `tracking-[-0.03em]`, 88 × 88 px on desktop and 64 × 64 px on
  mobile, content centred, number size 56 px / 40 px. Use `tabular-nums`. The tile is `aria-hidden`.
- Right of the tile, two lines of text, left aligned:
  - line 1 (Onest 700, 22 px desktop / 18 px mobile, `text-text`): the phrase from Spec 3, for example
    «свободных номера».
  - line 2 (`text-text-muted`, 14 px): «на сегодня из 12» (text exactly like this; «12» is the hotel's room count).
- Accessibility: the whole row is one `<p>` with an `sr-only` full sentence, e.g. «Сегодня свободно 4 номера из 12»,
  and the visible parts are `aria-hidden`. Do not announce the number twice.
- Both themes: only tokens. Lime tile with `--on-accent` text works in light and dark; check both.
- No animation, no counter roll-up, no pulsing dot. The tile is the single accent.
- `count` is never 0 here (Spec 3); the zero case is a different element.

### 3. Wording and plurals (Russian, verbatim)
Plural of «номер» with the adjective «свободный»:

| count | phrase |
|---|---|
| ends in 1, but not 11 (1, 21 …) | `свободный номер` |
| ends in 2–4, but not 12–14 | `свободных номера` |
| everything else (5–20, 25 …) | `свободных номеров` |

Put the plural function in `shared/lib` only if no plural helper exists there; otherwise reuse it. Keep it a pure
function with its own name, for example `pluralize(count, [one, few, many])`.

Zero rooms (`count === 0`): render no tile. Render a single muted line in the same place:
«Сегодня свободных номеров нет. Позвоните: 8 918 311-97-10» where the phone is a `tel:+79189 …` link built the same
way the contacts block builds it (use `typo()`/non-breaking spaces as in `contacts-block.tsx`; look there for the
helper or constant, do not retype the number if a constant exists in `shared/config/site.ts`).

`count === null` (availability unknown): `RoomsSection` renders nothing for this element.

### 4. QA in `scripts/qa/flows/home.mjs`
Add after the rooms checks (follow the style of the file):
- `out("rooms: free count tile / phrase", …)` → text of the tile and the visible phrase → `5 / свободных номеров`.
- `out("rooms: screen reader sentence", …)` → `Сегодня свободно 5 номеров из 12`.
Print a screenshot only if the file already does so for other checks.

## Tests to write
There is no unit-test runner. Verify by hand at 390 px and 1440 px, light and dark, and report what you saw:
- the row sits under the three room cards and does not push the horizontal rail on mobile;
- temporarily return `count` 1, 2, 4, 11, 21 and 0 from the page and check each phrase; revert afterwards;
- the contrast of `--on-accent` on `--accent` in both themes (the existing buttons already use this pair).

## Done when
CONVENTIONS "Done means" passes, and with a production build on port 3111: `node scripts/qa/flows/home.mjs` and
`npm run qa:a11y` print no unexpected values.

## Out of scope
A real booking-engine call, date-aware counts (the date bar keeps working as it does today), the free-rooms count on
`/rooms` and `/hotel`, changes to the hero stat «12 номеров», and any change in `../citypark-go/`.
