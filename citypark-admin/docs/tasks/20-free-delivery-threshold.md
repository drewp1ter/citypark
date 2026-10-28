# Task 20: Free delivery threshold

## Goal
On «Доставка» an admin sees and edits, for each town, the order total from which delivery is free (the old site's
`MIN_AMOUNT_TO_FREE_DELIVERY`, 500 ₽ for Белореченск).

## Depends on
04, and citypark-go task 59.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/features/delivery-prices/` (all files), `e2e/delivery-prices.spec.ts`,
`../citypark-go/docs/API.md` (section `### Delivery prices`, field `freeFrom`).

## Background
The old site had one hard-coded constant. citypark-go task 59 made it a per-town field: `freeFrom` (rubles, `0` = never
free) in `GET /delivery-prices`, `POST /delivery-prices` and `PATCH /delivery-prices/{id}`. `POST /orders` sets the
delivery price to 0 when the order total (without delivery) is at least `freeFrom` and `freeFrom` is above 0. In the
input `freeFrom` is optional and a missing value is stored as 0, so the edit dialog must always send it, or saving a
town clears its threshold. The admin's `schema.d.ts` was generated before task 59 and has no `freeFrom` yet.

## Files to change
- `src/shared/api/schema.d.ts`: regenerate with `npm run api:gen` (do not edit by hand).
- `src/features/delivery-prices/model/form.ts`, `model/form.test.ts`.
- `src/features/delivery-prices/ui/delivery-price-dialog.tsx`.
- `src/features/delivery-prices/ui/delivery-prices-page.tsx`.
- `e2e/delivery-prices.spec.ts`.

`api.ts` needs no change: it already sends `toRequest(form)`.

## Spec

### `model/form.ts`
```ts
export type DeliveryPriceForm = { name: string; value: string; price: string; freeFrom: string }
export function toRequest(f: DeliveryPriceForm): { name: string; value: string; price: number; freeFrom: number }
```
- `validateDeliveryPrice`: `freeFrom` is optional. When it is not empty after trimming, it uses the same normalisation
  (comma → dot) and `PRICE_PATTERN` as `price`, with the same message for a bad number; `0` is allowed. A negative
  number fails the pattern.
- `toRequest`: empty `freeFrom` → `0`; `"500,5"` → `500.5`.
- `SERVER_FIELD` also matches `freeFrom` (`deliveryPrices[0].freeFrom` and `freeFrom`).

### Dialog
- The initial form gets `freeFrom: item === undefined || item.freeFrom === 0 ? '' : String(item.freeFrom)`.
- After «Цена доставки, ₽» add a `TextField` `name='freeFrom'`, `inputMode='decimal'`, label «Бесплатная доставка от, ₽»,
  `Description` «Сумма заказа без доставки. Пусто или 0 — доставка всегда платная», `FieldError` from
  `errors.freeFrom`. Same structure as the price field.

### Page
- Table: a column «Бесплатно от» after «Цена», cell `formatRub(item.freeFrom)`, or «—» (muted) when it is 0.
- Cards (< 768 px): under the price, muted text «Бесплатно от 500 ₽» only when `freeFrom > 0`.

## Tests to write
- `form.test.ts`: update `valid` with `freeFrom: ''`; empty and `'0'` → no error; `'-1'`, `'abc'`, `'1.555'` → error
  on `freeFrom`; `toRequest` with `''` → `freeFrom: 0`, with `'500,5'` → `500.5`;
  `mapServerFields({ 'deliveryPrices[0].freeFrom': 'x' })` → `{ freeFrom: 'x' }`.
- `e2e/delivery-prices.spec.ts`: extend the existing flow: add «Тест <timestamp>» 123 ₽ with «Бесплатная доставка от»
  500 → the row shows «500 ₽» in «Бесплатно от»; edit and clear the field → «—»; editing only the price keeps a set
  threshold (reopen the dialog and see the value). The flow still deletes its town at the end.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/delivery-prices.spec.ts`. After setting Белореченск to 600 in
the UI, `curl -s "$VITE_API_URL/delivery-prices" | jq '.[] | select(.name=="Белореченск").freeFrom'` (URL from `.env`)
prints `600`. Set it back to 500 afterwards.

## Out of scope
- The backend (citypark-go task 59 is done).
- One global threshold for all towns.
- The website: citypark-modern still has `freeFrom: 500` in `src/entities/dish/model/delivery.ts`; reading it from
  `GET /delivery-prices` is a citypark-modern task.
