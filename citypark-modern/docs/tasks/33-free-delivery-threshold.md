# Task 33: Free delivery from the town's threshold on checkout

## Goal
On `/checkout` the «Доставка» line shows `0 ₽` when the cart subtotal reaches the selected town's free-delivery
threshold (`freeFrom`, set in the admin panel), and «К оплате» equals the amount the backend charges through Moneta.

Bug today: admin has «Белореченск — 100 ₽, бесплатно от 500 ₽»; a 780 ₽ cart shows «Доставка 100 ₽», «К оплате 880 ₽»,
but the backend creates the payment for 780 ₽.

## Depends on
26. No backend change: `GET /delivery-prices` already returns `freeFrom` (see `../citypark-go/docs/API.md`,
section «Delivery prices»).

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/dish/api/get-delivery-prices.ts`, `src/entities/dish/model/types.ts`,
`src/pages/checkout/ui/checkout-form.tsx`, and the backend rule to copy: `deliveryCost` in
`../citypark-go/internal/order/service.go` (read only).

## Background
- Root cause: `getDeliveryTowns()` maps the API response to `{ name, price }` and drops `freeFrom`;
  `checkout-form.tsx` (`const deliveryPrice = …`) always adds the town price.
- Backend rule (`deliveryCost`): delivery is free when `freeFrom > 0 && subtotal >= freeFrom`; `freeFrom = 0` means
  never free. `subtotal` is the items total without delivery. The frontend must use exactly this rule, so the page
  and the payment never disagree.
- API values are rubles (numbers), as `price` already is.

## Files to create
- `src/entities/dish/lib/town-delivery-price.ts`

## Files to change
- `src/entities/dish/model/types.ts`: add `freeFrom` to `DeliveryTown` (spec below).
- `src/entities/dish/api/get-delivery-prices.ts`: add `freeFrom: number` to `ApiDeliveryPrice`; map it as
  `freeFrom: freeFrom ?? 0` (an older backend may not send it).
- `src/entities/dish/index.ts`: export `townDeliveryPrice`.
- `src/pages/checkout/ui/checkout-form.tsx`:
  - `deliveryPrice` = `townDeliveryPrice(town, priced.subtotal)` for the selected town; `0` for pickup or no town.
  - Town option label: `Белореченск (+100 ₽)` when `freeFrom` is 0 (as now);
    `Белореченск (+100 ₽, бесплатно от 500 ₽)` when `freeFrom > 0`. Keep `textValue: town.name`.
  - The summary lines that already print `formatPrice(deliveryPrice)` (mobile `<summary>` and desktop «Доставка»
    row) need no other change: they show `0 ₽` when free.

`place-order.ts` needs no change: it does not send a delivery price; the backend computes it.

## Spec
```ts
// model/types.ts
export type DeliveryTown = {
	name: string
	/** Rubles. */
	price: number
	/** Order subtotal (rubles, without delivery) from which delivery is free; 0 = never free. */
	freeFrom: number
}

// lib/town-delivery-price.ts
/** Delivery price for a town; same rule as the backend's `deliveryCost` (citypark-go/internal/order/service.go). */
export function townDeliveryPrice(town: DeliveryTown, subtotal: number): number
```
Rules: `freeFrom > 0 && subtotal >= freeFrom` → `0`; otherwise `town.price`.

Examples (`price` 100):

| `freeFrom` | `subtotal` | result |
|---|---|---|
| 500 | 499 | 100 |
| 500 | 500 | 0 |
| 500 | 780 | 0 |
| 0 | 10 000 | 100 |

## Out of scope
`quoteDelivery` and the static `delivery.freeFrom = 500` in `src/entities/dish/model/delivery.ts` (cart panel, cart
sheet, home hero, home delivery section, menu page) still use hardcoded values. Do not touch them; a later task may
read them from the backend.

## Done when
```
npm run check
npm run build
npm start -- -p 3111
node scripts/qa/flows/checkout-and-contacts.mjs
```
- Add to `scripts/qa/flows/checkout-and-contacts.mjs`, right after the «towns from the backend» line, an `out()` that
  prints the selected town's «Доставка» line and «К оплате» for the default cart, with the expected value next to it
  (expected: `0 ₽` if the cart subtotal ≥ the town's threshold in the option label, else the town price).
- Manual check on `/checkout` with the backend's current data (Белореченск 100 ₽, free from 500 ₽):
  - cart 780 ₽ → «Доставка 0 ₽», «К оплате 780 ₽», mobile summary «доставка 0 ₽»;
  - cart below 500 ₽ → «Доставка 100 ₽», total = subtotal + 100;
  - a town with `freeFrom` 0 → its price at any subtotal;
  - change the quantity in the cart across 500 ₽ (back to the menu and return) → the line updates.
- If the backend at `MENU_API_URL` does not return `freeFrom`, stop and write that under the task in `PROGRESS.md`.
