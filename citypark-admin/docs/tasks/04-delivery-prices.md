# Task 04: Delivery prices

## Goal
On «Доставка» an admin sees every delivery price (town and price), adds one, edits one and deletes one.

## Depends on
03.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/api/` (all files), `src/shared/ui/page-header.tsx`,
`src/shared/ui/toast.ts`, `../citypark-go/docs/API.md` (section `### Delivery prices`).

## Background
An order gets its delivery price by comparing the address `city` with `name` exactly, so the name must be written the
way customers type their town. `value` is an optional longer description («Россия, Краснодарский край, Сочи»).
`POST` takes an array; this screen always sends one item. A 409 means the name is taken.

## Files to create
- `src/features/delivery-prices/api.ts`, `src/features/delivery-prices/model/form.ts` (+ `form.test.ts`),
  `src/features/delivery-prices/ui/delivery-prices-page.tsx`, `ui/delivery-price-dialog.tsx`, `ui/delete-dialog.tsx`
- `e2e/delivery-prices.spec.ts`

## Files to change
- `src/routes/_app/delivery.tsx`: render the page instead of the placeholder.

## Spec

### Page
`PageHeader` «Доставка» with the button «Добавить город». Under it a HeroUI table (on < 768 px a list of cards with the
same data): «Город» (`name`, with `value` in muted text under it when set), «Цена» (`formatRub`), and per row icon
buttons «Изменить» and «Удалить» (`aria-label` with the name: «Изменить Сочи»). Sorted by name (`localeCompare`
`ru`). Search field «Поиск города» above the table filters by name or value on the client (the list is small).
Empty state: «Городов пока нет. Добавьте первый.». Loading: skeleton rows. Load error: the message and «Повторить».

### Dialog (add and edit)
Fields: «Название города» (required, max 255, help text «Как клиенты пишут город в адресе, например «Белореченск».
Цена подставится, только если название совпадает»), «Описание» (optional, max 255), «Цена доставки, ₽» (number,
required, > 0, max 2 decimals). Buttons «Сохранить» and «Отмена». Client validation (`model/form.ts`) mirrors the
backend rules and returns Russian messages; the backend's field errors are mapped too
(`deliveryPrices[0].name` → `name`). 409 → field error on name «Такой город уже есть». On success: close, toast
«Сохранено», invalidate the list.

### Delete
Confirm dialog «Удалить «Сочи»? Заказы с этим городом станут без цены доставки.» with «Удалить» (danger) and «Отмена».
On success: toast «Удалено».

### `model/form.ts`
```ts
export type DeliveryPriceForm = { name: string; value: string; price: string }
export function validateDeliveryPrice(f: DeliveryPriceForm): Partial<Record<keyof DeliveryPriceForm, string>>
export function toRequest(f: DeliveryPriceForm): { name: string; value: string; price: number } // trims; "150,5" → 150.5
export function mapServerFields(fields: Record<string, string>): Partial<Record<keyof DeliveryPriceForm, string>>
```

## Tests to write
- `form.test.ts`: empty name → error; 256 characters → error; price `0`, `-1`, `abc`, `1.555` → error; `150,5` → 150.5;
  spaces trimmed; `deliveryPrices[0].price` → `price`.
- `e2e/delivery-prices.spec.ts`: add «Тест <timestamp>» 123 ₽ → row appears with «123 ₽»; add it again → «Такой город
  уже есть»; edit price to 150 → «150 ₽»; delete → row gone. At 390 px the cards show the same data.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/delivery-prices.spec.ts`.

## Out of scope
Bulk import, sorting by column, pagination.
