# Task 16: Content blocks, part 2: the hotel and rooms pages

## Goal
The hotel page's copy and the hotel texts shown on `/rooms` («Коротко», «В каждом номере») come from content blocks
(page `hotel`). After this task `blocks.data.ts` is complete for v1, and citypark-go task 54 can copy it into a
migration.

## Depends on
15.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/15-content-blocks-home.md` (rules for keys, types and labels),
`src/entities/content/` (all files), `src/entities/hotel/` (all files), `src/pages/hotel/ui/*.tsx`,
`src/pages/hotel/model/content.ts`, `src/pages/rooms/ui/rooms-page.tsx`.

## Background
Task 14 already turned the prices in these texts into placeholders. The hotel texts are rich text (*short* profile),
headings and titles are `text`, plain lists are `lines`. Guest documents (the PDFs with their meta) are not blocks:
they will be files, which is out of scope.

## Files to change
- `src/entities/content/model/blocks.data.ts`: hotel blocks after the home ones (Spec 1).
- `src/entities/hotel/model/hotel.data.ts`, `src/entities/hotel/api/get-hotel-info.ts`, `src/entities/hotel/index.ts`:
  remove `roomAmenities`, `quickFacts`, `rules` and the texts of `guestInfo` (keep `guestInfo.documents`) and their
  getters; remove the types nothing uses any more.
- `src/pages/hotel/model/content.ts`: remove what became blocks (`extras`, `breakfast.note`, `mobileCheckInRule`, …);
  delete the file if nothing is left.
- `src/pages/hotel/ui/*.tsx`, `src/pages/rooms/ui/rooms-page.tsx` (and the component that renders «Коротко» and
  «В каждом номере»): read from `getPageContent('hotel')`.

## Spec

### 1. Blocks (page `hotel`)
Sections and what they hold (names are a guide; derive the full list from the components):
- «Первый экран»: hero heading, lead, badges, photo(s);
- «Номерной фонд»: heading and lead (room names and prices stay in the rooms entity);
- «В каждом номере»: `hotel.amenities` (`lines`, the 12 items);
- «Коротко»: `hotel.quick-facts` (`lines`, with `{{smokingFine}}` in the last item);
- «Правила»: one `text` block for each rule title and one `richtext` for each rule text, keys
  `hotel.rules.<n>.title` / `.text` with `n` = 1…6 in the current order (placeholders kept);
- «Информация для гостей»: the intro (`text`) and, for each section, `hotel.guest-info.<n>.title` (`text`) and
  `.text` (`richtext`, `''` when the reference has no text yet);
- «Дополнительно»: the park photo (`image`), balconies and free extras titles (`text`) and texts (`richtext`),
  the breakfast heading (`text`) and note (`text`, `Заказ до 22:00 · доставка {{breakfastDelivery}}`);
- the mobile check-in rule as `hotel.rules.check-in-mobile` (`text`, with `{{lateCheckoutPerHour}}`).
A rule or guest-info section whose `.text` is empty renders as it does today for a section without text.

### 2. Rendering
Pages pass `values` built from `getHotelPrices()` (`breakfastDelivery`, `lateCheckoutPerHour`, `smokingFine`) to
`text()`/`html()`. `lines` items with placeholders: fill each item with `fillPlaceholders`.

## Tests to write
No unit-test runner. On a production build `/hotel` and `/rooms` at 390 and 1440 px, light and dark, look exactly as
before (screenshots before/after).

## Done when
CONVENTIONS "Done means" passes; `node scripts/qa/flows/hotel.mjs`, `node scripts/qa/flows/rooms.mjs` and
`npm run qa:a11y` print no unexpected values. Write the final number of entries in `blocks.data.ts` under the task
line in `PROGRESS.md` (citypark-go task 54 checks it).

## Out of scope
Guest documents (PDF files), the restaurant, contacts and news pages, reading blocks from the backend (task 20).
