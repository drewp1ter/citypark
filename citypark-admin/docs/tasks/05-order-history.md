# Task 05: Order history

## Goal
«Заказы» lists orders newest first with server-side search (number, phone, name), a date range, status and payment
type filters, and pages. Opening an order shows its details. Nothing can be changed (owner's decision).

## Depends on
03, and citypark-go task 42.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/api/`, `src/shared/lib/format.ts`,
`../citypark-go/docs/API.md` (sections `GET /orders` and `GET /orders/{id}`), the TanStack Router guide on search
params (`validateSearch`) for the installed version.

## Background
Statuses: `new` «Новый», `pending` «Ожидает оплаты», `paid` «Оплачен», `canceled` «Отменён», `confirmed`
«Подтверждён». Payment types: `cash` «Наличными», `online` «Онлайн». `total` is the items' sum; `deliveryPrice` is
separate. `address` is `null` for pickup («Самовывоз»). Dates in filters are local days (Moscow), both inclusive.

## Files to create
- `src/features/orders/api.ts`, `src/features/orders/model/search.ts` (+ test), `src/features/orders/model/labels.ts`
  (+ test), `src/features/orders/ui/orders-page.tsx`, `ui/order-filters.tsx`, `ui/orders-table.tsx`,
  `ui/order-details.tsx`
- `src/routes/_app/orders.$orderId.tsx`
- `e2e/orders.spec.ts`

## Files to change
- `src/routes/_app/orders.tsx`: the page, with `validateSearch` (Spec 1).
- `npm run api:gen` (the schema gained the parameters).

## Spec

### 1. Search params
```ts
export type OrderSearch = { q?: string; from?: string; to?: string; status?: OrderStatus; paymentType?: 'cash' | 'online'; page?: number }
export function parseOrderSearch(raw: Record<string, unknown>): OrderSearch // drops invalid values silently
export function toApiQuery(s: OrderSearch, pageSize: number): Record<string, string | number> // omits empty values
```
`page` ≥ 1, default 1; dates must be `YYYY-MM-DD`; `q` trimmed, max 100. Page size 20.

### 2. Filters
A search field «Номер, телефон или имя» (applies 400 ms after typing stops, and on Enter), a date range picker «Период»
(HeroUI date range; quick buttons «Сегодня», «Вчера», «7 дней», «30 дней»), selects «Статус» and «Оплата» (each with
«Все»), and «Сбросить» when any filter is set. Changing a filter resets `page` to 1. Filters live in the URL, so a
link reproduces the view and Back works.

### 3. List
Columns: «№» (`id`), «Дата» (`formatDateTime(createdAt)`), «Клиент» (name and phone; phone as a `tel:` link),
«Адрес» (city, address, or «Самовывоз»), «Сумма» (`formatRub(total + deliveryPrice)`), «Оплата», «Статус» (HeroUI
chip; colour by status: new/pending neutral or warning, paid/confirmed success, canceled danger). A row is a link to
`/orders/<id>` keeping the search params. On < 768 px: cards with №, date, client, sum and status. Under the list:
«Найдено: N» (`count`) and pagination. Empty: «Заказов не найдено» (+ «Сбросить фильтры» when filtered). Loading keeps
the previous page visible (`placeholderData: keepPreviousData`).

### 4. Details (`/orders/$orderId`)
A side panel (drawer) over the list on desktop, a full page on phones (choose one HeroUI pattern and use it for both
sizes if simpler), loaded with `GET /orders/{id}`: number and date, status and payment chips, customer name and phone,
address or «Самовывоз», comment (if any), items table («Блюдо», «Категория», «Кол-во», «Цена», «Сумма»), «Доставка»,
«Итого». «Назад к списку» returns with the same filters. 404 → «Заказ не найден».

## Tests to write
- `search.test.ts`: `{page: '0'}` → page dropped; `{from: '21.10.2026'}` → dropped; `{status: 'done'}` → dropped;
  `toApiQuery` omits `undefined` and `''`, adds `pageSize: 20`.
- `labels.test.ts`: every status and payment type has a Russian label; an unknown status shows the raw value.
- `e2e/orders.spec.ts`: needs at least one order in the dev database. Create one in `beforeAll` through the API
  (`POST /orders` as the backend docs describe, within working hours; if outside working hours or no product exists,
  `test.skip` with the reason). Then: the order is in the list; search by its number finds only it; search by a random
  string → «Заказов не найдено»; filter by today → present; opening it shows the items; the URL keeps the filters after
  going back.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/orders.spec.ts`.

## Out of scope
Confirming, deleting or editing orders, CSV export, live updates.
