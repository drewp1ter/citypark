# Task 26: Checkout sends the order to the backend and opens the payment page

## Goal
«Оплатить заказ» on `/checkout` places a real order through the backend's `POST /orders` (online payment) and sends
the customer to the Moneta payment page whose URL the backend returns. The delivery price on the checkout comes from
the backend's `GET /delivery-prices`. After paying, Moneta brings the customer back to `/checkout/done`; after a failed
payment, to `/checkout?payment=failed`.

## Depends on
01, and citypark-go task 58.

## Read first
`docs/tasks/CONVENTIONS.md`, `CLAUDE.md`, `src/pages/checkout/` (all files), `src/entities/dish/api/menu-api.ts`
(fetch pattern), `src/entities/dish/lib/quote-delivery.ts`, `src/entities/dish/index.ts`,
`scripts/qa/flows/checkout-and-contacts.mjs`, `../citypark-go/docs/API.md` (`#### GET /delivery-prices`,
`#### POST /orders`). For Next.js: `redirect` to an external URL from a Server Action, in `node_modules/next/dist/docs/`.

## Background
Until now `placeOrder` priced the order on the site and sent it to `submitLead`, which only logs. The owner decided:
- **online payment only**: the payment choice was removed from the form (the `RadioPills name='payment'` block is
  already deleted in the working tree; finish the removal);
- **the backend price wins**: the town list and delivery price come from `GET /delivery-prices` (rubles, flat price
  per town, including СОТ/ДНТ). The site's «бесплатно от 500 ₽» and «200–350 ₽» rules no longer apply on the checkout;
- the backend builds the payment URL (task 58) and returns it as `redirectUrl`.

## Files to create
- `src/entities/dish/api/get-delivery-prices.ts`: `getDeliveryTowns(): Promise<DeliveryTown[]>`.
- `src/pages/checkout/api/orders-api.ts`: `createOrder(body)`, server-only, used by `place-order.ts`.

## Files to change
- `src/entities/dish/model/types.ts` (or the file where `Dish` lives): `DeliveryTown`.
- `src/entities/dish/index.ts`: export `getDeliveryTowns`, `DeliveryTown`.
- `src/pages/checkout/model/schema.ts`, `src/pages/checkout/api/place-order.ts`,
  `src/pages/checkout/ui/checkout-form.tsx`, `src/pages/checkout/ui/checkout-page.tsx`.
- `scripts/qa/flows/checkout-and-contacts.mjs`.
- `CLAUDE.md`, section «Forms (how they work)»: the order action now posts to the backend's `POST /orders` and
  redirects to the Moneta page from `redirectUrl`; Moneta returns the customer to `/checkout/done`, where the cart is
  cleared. The other forms still use `submitLead`.

## Spec

### Delivery towns
```ts
/** A town the restaurant delivers to, with its flat delivery price (rubles). From GET /delivery-prices. */
export type DeliveryTown = { name: string; price: number }
```
`getDeliveryTowns()` fetches `serverEnv().MENU_API_URL + '/delivery-prices'` with `{ next: { revalidate: 60 } }` and
maps `{ name, price }`. A non-2xx response throws (same as `menu-api.ts`). `CheckoutPage` loads it next to `getMenu()`
with `Promise.all` and passes `towns` to `CheckoutForm`; a failure reaches `error.tsx` the same way as the menu.

### Schema (`model/schema.ts`)
- Remove `payments`, `payment`, `zones`, `zone`. Add `city: z.string().trim().max(80)`.
- `conditionalErrors`: for `method === 'delivery'` an empty `city` gives `city: 'Выберите населённый пункт'` (next to the
  street rule).

### Form (`checkout-form.tsx`)
- The «Населённый пункт» `Select` gets `name='city'`, its options are the towns (`value` and `label` = `name`, label
  `${name} (+${formatPrice(price)})`, `textValue` = `name`), the first town is selected by default, `error={errors.city}`.
  No towns → no select, and `errors.form` text «Доставка сейчас недоступна, выберите самовывоз.» under the method
  pills, with the submit disabled for delivery.
- Delivery price: pickup → 0; delivery → the price of the selected town. Total = `priced.subtotal + delivery`. The
  summary line «Доставка» shows `formatPrice(delivery)`. Remove the «уточним» / «не входит в сумму» case and the
  `quoteDelivery` import. Remove the leftover `payment` state.
- Buttons: desktop «Оплатить заказ», mobile «Оплатить» (the total stays on the right); pending text
  «Переходим к оплате…». The hint under the desktop button: «Оплата картой или через СБП на странице MONETA.RU».
- `?payment=failed` in the URL: `CheckoutPage` reads `searchParams` and passes `paymentFailed` to the form. The form shows,
  above the sections, a `role='alert'` card: «Оплата не прошла. Заказ сохранён в корзине: попробуйте ещё раз или
  позвоните нам.» The cart is not cleared.

### Server Action (`place-order.ts`)
Keep the field validation, the cart parsing and the menu repricing as they are. Then:
1. Delivery: load `getDeliveryTowns()`; an unknown `city` → `errors.city = 'Выберите населённый пункт'` (error state).
2. Body for `POST /orders`:
   ```json
   {
     "phone": "+79991234567",
     "name": "Анна",
     "comment": "Ко времени 19:30. Без лука",
     "paymentType": "online",
     "address": { "city": "Белореченск", "address": "Гоголя 10, кв. 5, подъезд 2" },
     "items": [{ "productId": 101, "amount": 2 }]
   }
   ```
   - `productId` = `Number(item.dish.externalId)`, `amount` = `item.quantity`.
   - `address` only for delivery; `address.address` = street, then `кв. {apartment}` and `подъезд {entrance}` when
     filled, joined with `, `.
   - `comment` = `Ко времени {time}.` when `when === 'scheduled'`, then the customer's comment, joined with a space;
     empty when both are empty.
3. `createOrder` posts JSON to `serverEnv().MENU_API_URL + '/orders'` with `cache: 'no-store'`. Result:
   - 201 → `{ orderId, redirectUrl }`;
   - 400 or 503 with a JSON `{ message }` → `errors.form = message` (the working-hours message has a real `\n`: render
     `errors.form` with `whitespace-pre-line` so it shows as two lines);
   - a network error or any other status → `console.error`, `errors.form = 'Не удалось отправить заказ. Попробуйте
     ещё раз через минуту.'`.
   An error returns the error state with the typed values, as now.
4. On success `redirect(redirectUrl)` outside any try/catch (as the current `redirect('/checkout/done')`).
   `submitLead('order', …)` is no longer called; leave `submitLead` itself for the other forms.

### Done page
`/checkout/done` is the Moneta success URL. It already clears the cart. Change the text to «Оплата прошла, заказ
принят. Мы перезвоним, если что-то нужно уточнить. Доставляем {delivery.hours}.» (keep the rest).

## Tests to write
No unit tests in this project; `scripts/qa/flows/checkout-and-contacts.mjs` is the check:
- button names `Оплатить заказ`; the «outskirts» step becomes «second town from the backend → delivery line shows its
  price and the total includes it»;
- `?payment=failed` → the alert text above;
- a valid order: this **creates an order on the backend**, so put it behind `QA_PLACE_ORDER=1` (off by default) and do
  not run it yourself (CLAUDE.md: no POSTs without asking). With the flag: the page URL host is the Moneta assistant
  host (task 58's `MONETA_URL`); print it. A 503 outside working hours prints the alert text instead.

## Done when
`npm run check` is clean, `npm run build` passes, `node scripts/qa/flows/checkout-and-contacts.mjs` prints the expected
lines, and a manual order on a local backend with `MONETA_TEST_MODE=true` opens the Moneta demo payment form (not a
signature error). Ask the owner to run the `QA_PLACE_ORDER=1` flow and the manual order, and write under the PROGRESS
line what is left for them.

## Out of scope
- The cart panel and sheet (`widgets/cart-summary`), the menu, home and done-page texts that show «0 ₽ от 500 ₽» and
  «СОТ и ДНТ 200–350 ₽», and `quoteDelivery` used by them. They still describe the old rules; the owner decides later.
- Cash payment, the account page, order status pages, the Moneta widget.
- Any change to citypark-go.
