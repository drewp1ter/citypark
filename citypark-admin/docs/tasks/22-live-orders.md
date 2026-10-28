# Task 22: Live orders

## Goal
A new page «Заказы онлайн» (`/orders/live`) shows today's orders in three columns and updates without reloading,
from the backend's WebSocket. An order that becomes paid shows a toast; when the tab is in the background, it also
adds a counter to the tab title and shows a browser notification. Other events (a new cash or unpaid order, a
confirmation) only update the columns. Nothing can be changed on this page.

## Depends on
05, and citypark-go task 64.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/features/orders/` (`api.ts`, `model/labels.ts`, `ui/order-chips.tsx`,
`ui/orders-table.tsx`), `src/shared/api/client.ts`, `src/shared/ui/nav-items.ts`, `../citypark-go/docs/API.md`
(sections `GET /orders` and `GET /orders/live (WebSocket)`).

## Background
Decided by the owner (2026-10-28). The socket is `GET /orders/live` on `VITE_API_URL` with `http` → `ws` (`https` →
`wss`). The JWT goes as a subprotocol: `new WebSocket(url, ['bearer', token])`. Messages are JSON `LiveOrderEvent`
(from the generated schema): `hello` first, then `order.pending` (online, not paid), `order.created` (cash),
`order.paid`, `order.confirmed`, each with the full order. There is no replay: after every `hello` reload the list.
A browser cannot read the HTTP status of a failed handshake; after a failed connect, the list reload (a normal
request through `api`) hits the existing 401 → sign-in redirect in `client.ts` when the token has expired.
A feature never imports another feature: move what both pages need (`statusLabel`, `paymentLabel`, the status and
payment chips) to `src/shared/` and update the imports in `features/orders`.

## Files to create
- `src/routes/_app/orders_.live.tsx` (the `_` after `orders` keeps it out of the `orders.tsx` layout)
- `src/features/live-orders/api.ts`
- `src/features/live-orders/model/socket.ts` (+ `socket.test.ts`)
- `src/features/live-orders/model/board.ts` (+ `board.test.ts`)
- `src/features/live-orders/model/alerts.ts` (+ `alerts.test.ts`)
- `src/features/live-orders/ui/live-orders-page.tsx`, `ui/order-column.tsx`, `ui/live-order-card.tsx`,
  `ui/connection-status.tsx`
- `e2e/live-orders.spec.ts`

## Files to change
- `src/shared/api/schema.d.ts`: `npm run api:gen`.
- `src/shared/ui/nav-items.ts`: `{ label: 'Заказы онлайн', to: '/orders/live' }` right after «Заказы».
- `src/features/orders/model/labels.ts`, `ui/order-chips.tsx` → moved to `src/shared/lib/order-labels.ts` (+ its
  test) and `src/shared/ui/order-chips.tsx`; `features/orders` imports them from there.

## Spec

### 1. `model/socket.ts`
```ts
export type LiveOrderEvent = components['schemas']['LiveOrderEvent']
export type LiveStatus = 'connecting' | 'online' | 'offline'
export function liveUrl(apiUrl: string): string // 'http://h/api/v1' → 'ws://h/api/v1/orders/live'; https → wss
export function nextDelay(attempt: number, random: () => number = Math.random): number
export function connectLiveOrders(opts: {
	url: string
	getToken: () => string | null
	onEvent: (e: LiveOrderEvent) => void
	onStatus: (s: LiveStatus) => void
	onFailedConnect: () => void
}): () => void // returns stop()
```
- `nextDelay`: `min(30000, 1000 * 2 ** attempt)`, plus jitter 0–20 % (`random()`); attempt starts at 0.
- Status `connecting` before each `new WebSocket`; `online` on `hello`; `offline` on close. On close, schedule a
  reconnect with `nextDelay(attempt++)`; `hello` resets `attempt` to 0. A close that happened before `open` calls
  `onFailedConnect`. No token → `offline` and no socket (the app shell already redirects).
- Messages: `JSON.parse` in try/catch; anything that is not an object with a known `type` is ignored.
- `stop()` cancels the timer, closes the socket with 1000 and stops reconnecting.

### 2. `model/board.ts`
```ts
export type LiveOrder = components['schemas']['Order'] // the GET /orders/{id} shape (use the real schema name)
export function applyEvent(orders: LiveOrder[], e: LiveOrderEvent, today: string): LiveOrder[]
export type Board = { pending: LiveOrder[]; fresh: LiveOrder[]; confirmed: LiveOrder[] }
export function groupOrders(orders: LiveOrder[]): Board
```
- `applyEvent`: `hello` → same array. An order event upserts by `id`; an existing order is replaced only when the
  incoming `updatedAt` is not older. Orders whose `createdAt` is not `today` (Moscow day, `moscowToday`) are ignored.
- `groupOrders`: `pending` → «Ждут оплаты»; `new` and `paid` → «Новые»; `confirmed` → «Подтверждены»; `canceled`
  left out. Each column newest first by `createdAt`.

### 3. `model/alerts.ts`
```ts
export function isPaidAlert(e: LiveOrderEvent): boolean // true for order.paid only (owner's decision, 2026-10-28)
export function titleWithCount(base: string, count: number): string // 0 → base; 3 → '(3) ' + base
export function alertText(o: LiveOrder): string // 'Оплачен заказ №123 — 1 450 ₽' (formatRub(total + deliveryPrice))
```

### 4. `api.ts`
```ts
export function todayOrdersQuery(today: string) // queryOptions, key ['live-orders', today],
// GET /orders?from=today&to=today&pageSize=100, returns the items array
export function useLiveOrders(): { board: Board; status: LiveStatus; isLoading: boolean; error: unknown }
```
`useLiveOrders` runs the query and, in one `useEffect`, `connectLiveOrders` with `getSession()?.token`. `onEvent`:
`hello` → `invalidateQueries(['live-orders'])`; an order event → `setQueryData` with `applyEvent`, also
`invalidateQueries(['orders'])` so the history page is fresh, and calls the page's alert callback when
`isPaidAlert`. `onFailedConnect` → `invalidateQueries(['live-orders'])`. Stop the socket on unmount.

### 5. UI
- `live-orders-page.tsx`: `PageHeader` «Заказы онлайн» with `<ConnectionStatus>` and the button «Включить
  уведомления» on the right. The button is shown only when `'Notification' in window` and the permission is
  `default`; a click calls `Notification.requestPermission()`. Three `<OrderColumn>`s side by side from 1024 px,
  stacked below. Loading: three column skeletons. Load error: `role='alert'` with the message and «Повторить».
- On a paid event (`isPaidAlert`), and only then: `toast` (`toast.success(alertText(o))` or the HeroUI v3 equivalent with a link «Открыть»);
  when `document.hidden`, add 1 to a counter, set `document.title = titleWithCount('Заказы онлайн', n)` and, if
  `Notification.permission === 'granted'`, `new Notification(alertText(o))` whose click focuses the window and opens
  the order. On `visibilitychange` to visible reset the counter. Restore the old title on unmount.
- `order-column.tsx`: heading with the count («Новые · 3»); empty: muted «Пусто».
- `live-order-card.tsx`: a link to `/orders/<id>`: «№123», time `HH:MM` of `createdAt`, «Ко времени HH:MM» when
  `deliveryTime` is set, name, phone as `tel:` link, address or «Самовывоз», comment (2 lines max), sum
  `formatRub(total + deliveryPrice)`, payment chip. A card that arrived by an event in the last 10 s has an accent
  ring (keep the set of fresh ids in component state; no colour hex values).
- `connection-status.tsx`: a chip: `online` «Онлайн» (success), `connecting` «Подключение…» (warning), `offline`
  «Нет связи» (danger); `aria-live='polite'`.

### 6. Deployment
Add to the deployment notes from task 17: the proxy in front of the API must pass `Upgrade` and `Connection` for
`/api/v1/orders/live` and allow idle connections for more than 30 s (the server pings every 30 s).

## Tests to write
- `socket.test.ts` (a fake `WebSocket` class on `globalThis`, `vi.useFakeTimers()`): `liveUrl` for http and https;
  `nextDelay(0, () => 0)` = 1000, `nextDelay(10, () => 0)` = 30000, `nextDelay(0, () => 1)` = 1200; the socket is
  created with `['bearer', token]`; `hello` → status `online`; close → `offline`, a new socket after 1000 ms; a close
  before open calls `onFailedConnect`; after `hello` the delay is back to 1000; bad JSON and an unknown type are
  ignored; `stop()` closes and no new socket appears after the delay.
- `board.test.ts`: insert, update with a newer `updatedAt`, an older event ignored, another day ignored, `hello`
  unchanged; `groupOrders` puts each status in its column, `canceled` nowhere, newest first.
- `alerts.test.ts`: `isPaidAlert` is true for `order.paid` and false for the other four types; `titleWithCount` for 0 and 3; `alertText` uses
  non-breaking spaces from `formatRub`.
- `e2e/live-orders.spec.ts`: sign in, open «Заказы онлайн» from the menu, see «Онлайн». Create a cash order through
  the API as in `e2e/orders.spec.ts` (skip with the reason outside working hours or without products) → without a
  reload its card is in «Новые», and no toast shows (only paid orders alert; the paid toast is covered by
  Vitest, since paying needs a signed Moneta callback). Create an online order → its card is in «Ждут
  оплаты». Clicking a card opens the order details.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/live-orders.spec.ts e2e/orders.spec.ts`.

## Out of scope
- The backend (citypark-go task 64).
- Confirming, editing or deleting orders (the owner's decision in task 05 stands), sounds, orders of other days.
