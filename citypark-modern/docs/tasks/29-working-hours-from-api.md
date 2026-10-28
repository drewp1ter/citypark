# Task 29: Working hours from the backend

## Goal
Every place on the site that shows or checks the hours (10:00–23:00 written in the code today) uses
`GET /working-hours` from the backend, which admins edit on the admin's «Доставка» page. The starting value is
9:00–22:00.

## Depends on
28. Needs citypark-go task 63 (`GET /working-hours`, the `deliveryTime` check on `POST /orders`).

## Read first
`docs/tasks/CONVENTIONS.md`, `src/shared/api/get-site-settings.ts` (getter with a fallback),
`src/pages/checkout/api/place-order.ts`, `src/pages/checkout/api/orders-api.ts`, `src/pages/checkout/model/schema.ts`,
`../citypark-go/docs/API.md` (sections `### Working hours` and `#### POST /orders`).

## Background
The backend had its own hours in config (9:00–22:00) while the site said 10:00–23:00. Now there is one value:
`GET /working-hours` → `{ "start": "09:00", "end": "22:00", "updatedAt": "…" }` (`HH:MM`, leading zero, `start` before
`end`). Orders are accepted from `start` until `end` (503 otherwise); a «Ко времени» time must be within
[`start`, `end`], else `POST /orders` answers 400 with `errors.deliveryTime = "Мы работаем с 9:00 до 22:00"`.

## Files to create
- `src/shared/api/get-working-hours.ts`
- `src/shared/lib/working-hours.ts` (pure helpers)

## Files to change
- `src/shared/api/index.ts`, `src/shared/lib/index.ts`: export the new names.
- `src/entities/dish/model/delivery.ts`: remove `hours`.
- Every user of `delivery.hours` (find with `grep -rn "delivery.hours" src`): `src/app/layouts/checkout-layout.tsx`,
  `src/pages/home/ui/delivery-section.tsx`, `src/pages/menu/ui/menu-page.tsx`,
  `src/pages/checkout/ui/checkout-done-page.tsx`.
- `src/pages/contacts/ui/contacts-page.tsx`: the hard-coded `10:00–23:00`.
- `src/pages/checkout/model/schema.ts`, `src/pages/checkout/api/place-order.ts`, `src/pages/checkout/api/orders-api.ts`,
  `src/pages/checkout/ui/checkout-page.tsx`, `src/pages/checkout/ui/checkout-form.tsx`.

## Spec

### `shared/lib/working-hours.ts`
```ts
export type WorkingHours = { start: string; end: string } // "09:00", "22:00"
export const defaultWorkingHours: WorkingHours = { start: '09:00', end: '22:00' }
/** "09:00" → "9:00". */
export function displayTime(hhmm: string): string
/** "9:00–22:00" (en dash), the text shown on the pages. */
export function formatHours(h: WorkingHours): string
/** "Мы работаем с 9:00 до 22:00": the same text the backend sends in errors.deliveryTime. */
export function hoursError(h: WorkingHours): string
```

### `shared/api/get-working-hours.ts`
`export const getWorkingHours = cache(async (): Promise<WorkingHours> => …)`: `fetch(MENU_API_URL + '/working-hours',
{ next: { revalidate: 60 } })`, returns `{ start, end }`. On any error (network, non-200, a value that is not `HH:MM`)
it logs `Working hours: using the built-in hours.` with the error and returns `defaultWorkingHours`, like
`getSiteSettings`. The page keeps working when the backend is down.

### Pages
- Each Server Component that showed `delivery.hours` calls `await getWorkingHours()` and shows `formatHours(hours)`.
  If a component is not async today, make the nearest Server Component load the hours and pass the text down as a
  prop. Keep the text around it unchanged («Доставка …», «Доставляем …», «Ресторан и доставка · …»).
- `contacts-page.tsx`: the `hours` span gets its text from `formatHours` (still `whitespace-nowrap`).

### Checkout
- `conditionalErrors(order, hours: WorkingHours)`: the range check uses `hours.start`/`hours.end` (both ends
  included) and the message `hoursError(hours)`. `place-order.ts` loads `getWorkingHours()` before calling it.
- `checkout-page.tsx` loads the hours (with the menu and the towns, in the same `Promise.all`) and passes them to
  `CheckoutForm`; the time input's `min`/`max` are `hours.start`/`hours.end`.
- `orders-api.ts`: a 400 result also carries `fields?: Record<string, string>` from the body's `errors`.
  `place-order.ts`: when `result.fields?.deliveryTime` is set, put it in `errors.time` (not `errors.form`), so the
  message appears under the time field. Everything else stays as it is (503 → `errors.form`).

## Tests to write
No unit test runner in this project. Check by hand on a production build with the backend running:
- `/menu`, `/` (delivery section), `/contacts`, `/checkout` (header) show `9:00–22:00`.
- In admin set 10:00–23:00 (or `curl -X PUT .../working-hours` with the admin token); within 60 s all those pages show
  `10:00–23:00`; set it back.
- Checkout «Ко времени» `08:30` → «Мы работаем с 9:00 до 22:00» under the time field.
- Stop the backend: the pages above still render with `9:00–22:00`.
Then run `node scripts/qa/flows/menu.mjs` and the checkout flow (`scripts/qa/flows/checkout-and-contacts.mjs`, without
`QA_PLACE_ORDER`) and fix every difference.

## Done when
CONVENTIONS "Done means" pass, `grep -rn "10:00\|23:00" src/pages src/app src/entities/dish` finds nothing about the
working hours (hotel texts in `blocks.data.ts` stay), and the checks above were done.

## Out of scope
- The backend (citypark-go task 63) and the admin panel (citypark-admin task 21).
- Showing "closed now" banners or disabling the checkout outside the hours.
- Room-service and breakfast hours in the hotel texts.
