# Task 28: Checkout sends the delivery time as its own field

## Goal
For «Ко времени…» the checkout sends `deliveryTime: "18:30"` in `POST /orders` and no longer writes «Ко времени 18:30.»
into `comment`. The backend shows the time in Telegram and iiko on its own line.

## Depends on
26. Needs citypark-go task 61 (`deliveryTime` on `POST /orders`).

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/checkout/api/orders-api.ts`, `src/pages/checkout/api/place-order.ts`,
`../citypark-go/docs/API.md` (section `#### POST /orders`).

## Files to change
- `src/pages/checkout/api/orders-api.ts`: `CreateOrderBody` gets `deliveryTime?: string // "HH:MM"`.
- `src/pages/checkout/api/place-order.ts`: builds the body.

## Spec
- `comment: comment` (only the customer's text, no «Ко времени» prefix).
- After the body is built: `if (when === 'scheduled') body.deliveryTime = time`. For `asap`, the field is not sent.
- Validation stays as it is (`schema.ts` already checks `HH:MM` and 10:00–23:00). A backend 400 is shown as today,
  through `result.message`.

## Tests to write
No unit test runner covers this server action. Checking is manual (below), plus `npm run check`.

## Done when
`npm run check` is clean. Owner check (a note under the PROGRESS line): with a local citypark-go that has task 61, place
a «Ко времени» order. In Telegram it has the «Ко времени» line, and the comment has only the customer's text.

## Out of scope
The checkout UI, other fields, the backend.
