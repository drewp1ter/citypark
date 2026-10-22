# Task 14: Hotel prices as numbers

## Goal
The hotel's fees and the breakfast menu are one typed data object read through a getter (`getHotelPrices()`), and every
text that shows one of these prices builds it from the number. No price of the hotel is written inside a string any
more.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`, `../GLOSSARY.md` (Hotel prices, Price placeholder), `src/pages/hotel/model/content.ts`,
`src/pages/hotel/ui/extras.tsx`, `src/entities/hotel/model/hotel.data.ts`, `src/entities/hotel/api/get-hotel-info.ts`,
`src/entities/hotel/index.ts`, `src/shared/lib/format.ts`, `../citypark-go/docs/tasks/52-hotel-prices.md` (the API
shape this mirrors).

## Background
Decided by the owner (ADR 0001): a price is a number stored once. Today: «доставка 100 ₽» (breakfast note), «500 ₽/час»
(late checkout, in `rules` and again in `mobileCheckInRule`), «штраф 2 000 ₽» (in `quickFacts` and in `rules`), and
the breakfast prices in `content.ts`. The backend will serve them from `GET /hotel/prices` (citypark-go task 52);
task 20 switches the getter to it. Texts that mention a price will use placeholders (`{{smokingFine}}`) from task 15
on; this task only introduces the numbers and the function that fills placeholders.

## Files to create
- `src/entities/hotel/model/prices.data.ts`, `src/entities/hotel/api/get-hotel-prices.ts`
- `src/shared/lib/placeholders.ts`

## Files to change
- `src/entities/hotel/model/types.ts`: `HotelPrices`, `BreakfastItem` (Spec 1).
- `src/entities/hotel/index.ts`: export `getHotelPrices` and the types.
- `src/entities/hotel/model/hotel.data.ts`: price texts use placeholders (Spec 3).
- `src/pages/hotel/model/content.ts`: remove `breakfast.items` and the price from `breakfast.note` and
  `mobileCheckInRule` (Spec 3).
- `src/pages/hotel/ui/extras.tsx`, the components that render `rules`, `quickFacts` and `mobileCheckInRule`: fill
  placeholders (Spec 3). Find them with `grep -rn "getHotelRules\|getQuickFacts\|mobileCheckInRule" src`.
- `src/shared/lib/index.ts`: export `fillPlaceholders`.

## Spec

### 1. Types and data
```ts
export type BreakfastItem = { name: string; shortName: string; price: number; desktopOnly: boolean }
export type HotelPrices = {
	breakfastDelivery: number
	lateCheckoutPerHour: number
	smokingFine: number
	breakfast: readonly BreakfastItem[]
}
```
`prices.data.ts`: `breakfastDelivery: 100, lateCheckoutPerHour: 500, smokingFine: 2000`, and the five breakfast items
from `content.ts` (`shortName: ''` when there is none, `desktopOnly: false` when not set).
`getHotelPrices(): Promise<HotelPrices>` returns it (comment: "swap the body for the API in task 20").

### 2. Placeholders
```ts
/** Replaces {{name}} with formatPrice(values[name]). Unknown names are left as they are and logged once (console.error). */
export function fillPlaceholders(text: string, values: Readonly<Record<string, number>>): string
```
Match `\{\{\s*([a-zA-Z]+)\s*\}\}`. Apply `typo()` to the result where the original text used `typo()`.

### 3. Texts
- `quickFacts[2]`: `Без животных · не курить (штраф {{smokingFine}})`.
- `rules` «Поздний выезд»: `До 6 ч — {{lateCheckoutPerHour}}/час · 6–12 ч — половина суток · 12–24 ч — сутки.`
- `rules` «Нельзя»: `С животными. Курить в помещениях — штраф {{smokingFine}}.`
- `mobileCheckInRule`: `Заезд круглосуточно, выезд до 12:00. Поздний выезд: до 6 ч — {{lateCheckoutPerHour}}/час.`
- `breakfast.note`: `Заказ до 22:00 · доставка {{breakfastDelivery}}`.
- The components call `getHotelPrices()` (pages are Server Components; pass the values down) and render
  `fillPlaceholders(text, prices)`; the breakfast list renders `prices.breakfast`.
The rendered pages must be **identical** to now: «500 ₽/час», «2 000 ₽», «100 ₽» with the same non-breaking spaces.

## Tests to write
No unit-test runner. On a production build, `/hotel` and `/rooms` at 390 and 1440 px look exactly as before; then
temporarily set `smokingFine: 3000` → both places show «3 000 ₽»; revert.

## Done when
CONVENTIONS "Done means" passes; `node scripts/qa/flows/hotel.mjs` and `node scripts/qa/flows/rooms.mjs` print no
unexpected values; `grep -rn "₽" src/entities/hotel src/pages/hotel/model` finds nothing.

## Out of scope
Content blocks (task 15), the API (task 20), room prices (task 13).
