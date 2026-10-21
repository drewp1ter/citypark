# Task 01: Menu from the backend API

## Goal
`getMenu()` returns the real menu from citypark-go (`GET /categories` and `GET /products`) instead of the static
`menu.data.ts`. The `/menu` page, header cart, CartBar, checkout and the order action use it without other changes.
When the backend is unreachable, `/menu` and `/checkout` show an error page and the home page still renders.

## Depends on
Nothing. The backend must have task 39 done (it is).

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/dish/api/get-menu.ts`, `src/entities/dish/model/types.ts`,
`../citypark-go/docs/API.md` (sections `GET /categories` and `GET /products` only).
Next docs: `node_modules/next/dist/docs/01-app/03-api-reference/03-file-conventions/error.md`,
`01-app/01-getting-started/06-fetching-data.md`, `01-app/03-api-reference/02-components/image.md` (`remotePatterns`).

## Files to create
- `.env.example`
- `src/shared/config/env.ts`
- `src/entities/dish/api/menu-api.ts`
- `src/entities/dish/api/to-dish.ts`
- `src/pages/menu/ui/menu-error.tsx`
- `app/(menu)/menu/error.tsx`, `app/(checkout)/checkout/error.tsx`

## Files to change
- `src/shared/config/index.ts`: export `serverEnv`.
- `src/entities/dish/api/get-menu.ts`: read from the API (Spec 4).
- `src/entities/dish/index.ts`: no new exports except what the Spec names.
- `src/pages/menu/index.ts`: export `MenuError`.
- `src/pages/menu/ui/dish-card.tsx`: show ` · ` between composition and weight only when the weight is not empty.
- `src/app/layouts/menu-layout.tsx`: survive a failed `getMenu()` (Spec 6).
- `src/pages/home/ui/home-page.tsx`: survive a failed `getMenu()`; choose dishes and categories without slugs (Spec 7).
- `src/pages/checkout/api/place-order.ts`: form error when the menu cannot be read (Spec 6).
- `next.config.ts`: `images.remotePatterns` (Spec 5).
- `scripts/qa/flows/menu.mjs`, `scripts/qa/flows/home.mjs`: no hard-coded dish names or category slugs (Spec 8).
- `scripts/qa/flows/checkout-and-contacts.mjs`, `scripts/qa/a11y.mjs`, `scripts/qa/responsive.mjs`,
  `scripts/qa/keyboard.mjs`, `scripts/screenshots.mjs`: the seeded cart uses real product ids (Spec 8).
- `CLAUDE.md` ("Content" bullet) and `README.md`: the menu comes from citypark-go; how to set `MENU_API_URL`.
- Delete `src/entities/dish/model/menu.data.ts` and the 11 photos `public/images/dish-*.jpg` (Spec 9).

## Spec

### 1. Env
`.env.example`:
```
# citypark-go API, including /api/v1. The backend runs on :3000, so run this site on another port (npm run dev -- -p 3001).
MENU_API_URL=http://localhost:3000/api/v1
# Optional: host of the menu photos (from iiko), allowed for next/image. Empty = no remote photos.
MENU_IMAGE_HOST=
```
`src/shared/config/env.ts` exports `serverEnv(): { MENU_API_URL: string }`, parsed with zod (`z.url()`) on each call,
not at import time, so a build without the variable fails only where the menu is read. The error message names the
variable. The trailing `/` is removed.

### 2. `menu-api.ts` (server only)
```ts
export type ApiImage = { id: number; url: string; type: string };
export type ApiCategory = { id: number; name: string; description: string; images: ApiImage[]; isAvailableForOrder: boolean };
export type ApiProduct = {
  id: number; categoryId: number; name: string; description: string;
  measureUnit: string; weight: number; price: number; images: ApiImage[]; isAvailableForOrder: boolean;
};
export async function fetchCategories(): Promise<ApiCategory[]>; // GET /categories
export async function fetchProducts(): Promise<ApiProduct[]>;     // GET /products?pageSize=500&page=1,2,… 
```
- One private `fetchJson<T>(path)`: `fetch(serverEnv().MENU_API_URL + path, { next: { revalidate: 60 } })`.
  Cache Components is off in `next.config.ts`, so this is the right caching API; 60 s because availability depends on
  the time of day. A non-2xx status throws `Error("GET <path>: <status>")`.
- `fetchProducts` asks page 1, 2, … and stops after the first page with fewer than 500 items.
- Types list only the fields we use; the API sends more, and that is fine.

### 3. `to-dish.ts`
```ts
export function toCategory(c: ApiCategory): MenuCategory; // { id: String(c.id), name: c.name }
export function toDish(p: ApiProduct): Dish;
export function weightLabel(kg: number): string;
```
`toDish`:
- `id` and `externalId`: `String(p.id)`; `categoryId`: `String(p.categoryId)`; `name`: `p.name`.
- `composition`: `p.description.trim()` or `undefined` when empty.
- `photo`: `{ src: p.images[0]?.url ?? PLACEHOLDER_PHOTO, alt: p.name }`. When `MENU_IMAGE_HOST` is empty, remote
  URLs would fail in `next/image`, so use `PLACEHOLDER_PHOTO` unless the URL's host equals `MENU_IMAGE_HOST`.
- one variant: `{ id: "default", label: weightLabel(p.weight), weight: weightLabel(p.weight), price: p.price > 0 ? p.price : null }`.
- `available`: `p.isAvailableForOrder`.

`weightLabel` (the API's `weight` is in **kilograms**; `measureUnit` is ignored): `0` or less → `""`; below 1 →
grams, rounded: `0.2` → `"200 г"`; 1 and more → kilograms with a comma and at most two decimals: `1` → `"1 кг"`,
`1.5` → `"1,5 кг"`. Wrap the result in `typo()` (non-breaking space before the unit).

### 4. `getMenu()`
```ts
export const getMenu = cache(async (): Promise<Menu> => { … }); // cache from "react"
```
- `Promise.all([fetchCategories(), fetchProducts()])`.
- Keep only products whose `categoryId` is in the category list (`GET /products` also returns products of hidden
  categories). Keep the API order of both lists.
- Errors are not caught here; they reach the caller.
- `getDishesByIds(ids)` filters `(await getMenu()).dishes`.
- Remove `hiddenCategoriesCount` with `menu.data.ts` (nothing else uses it).

### 5. Remote photos
`next.config.ts`: when `process.env.MENU_IMAGE_HOST` is set, `images.remotePatterns = [{ protocol: "https", hostname: <it> }]`;
otherwise no `remotePatterns`. Keep the existing options.

### 6. Backend failures
- `MenuError` (`src/pages/menu/ui/menu-error.tsx`, `"use client"`), props `{ error: unknown; retry: () => void }`
  as in the Next 16 `error.js` docs (the prop is `retry`, not `reset`). Logs `error` with `console.error` in an effect.
  Layout like `src/pages/not-found/ui/not-found-page.tsx`, tokens only:
  - `h1`: «Меню временно недоступно»
  - text: «Не получилось загрузить меню. Попробуйте обновить страницу или закажите по телефону.»
  - `Button` «Обновить» → `retry()`; outline `Button` «8 918 311-97-91» → `tel:+79183119791` (take both from `site` config if they are there).
- `app/(menu)/menu/error.tsx` and `app/(checkout)/checkout/error.tsx`: `export { MenuError as default } from "@/pages/menu";`
  If Next refuses a re-export as an error boundary, add `"use client"` to the route file; if it then pulls server code
  into the client bundle, stop and write that in PROGRESS.md.
- `MenuLayout`: `getMenu()` failure → log it, use `dishes = []` (header and CartBar show an empty cart); the page
  throws into its own boundary. A layout error would not be caught by the sibling `error.tsx`.
- `place-order.ts`: wrap only `getMenu()` in try/catch; on failure log it and return
  `{ status: "error", errors: { form: "Не удалось проверить меню. Попробуйте ещё раз через минуту." }, values }`.
  `redirect()` stays outside any try/catch.

### 7. Home page
- `getMenu().catch(…)` inside the existing `Promise.all`: log and return `null`. When `null`, do not render `DeliverySection`.
- Remove `HOME_DISH_IDS` and the slug filter. Categories: the first 4. Dishes: the first 3 that are `available`
  and whose photo is not `PLACEHOLDER_PHOTO`; if fewer than 3, fill up with other available dishes.

### 8. QA flows
`menu.mjs` and `home.mjs` click «Добавить в корзину: салат свекольный», «…крем-суп из тыквы», «…салат овощной» and wait for
`/menu?category=steaks`. Rewrite those steps to use what the page shows: the first/second «Добавить в корзину: …»
buttons, and the `href` of the second category chip. Keep the `?category=nope` check (it must fall back to the first
category). Keep every other check and the output format.

The other scripts put a cart into `localStorage` with slug ids (`"beet-salad"`, `"pumpkin-soup"`, `"mushroom-risotto"`),
which now match nothing. Add `scripts/qa/menu-cart.mjs` exporting `async function seedCart(quantities)`. It fetches
`${process.env.MENU_API_URL}/products?pageSize=500` (default `http://localhost:3000/api/v1`), takes the first products
with `isAvailableForOrder` and `price > 0`, and returns `[{ dishId: String(id), variantId: "default", quantity }]`, one
line per given quantity (`seedCart([1, 2])` → two lines). Use it in those scripts instead of the literal carts.

### 9. Remove the static menu
Delete `menu.data.ts` and `public/images/dish-{beet-salad,borscht,cabbage-pie,kurze,mushroom-risotto,pumpkin-soup,thai-rice,tofu-poke,vegetable-pan,vegetable-salad,warm-bean-salad}.jpg`.
Before deleting, `grep -rn "dish-" src app scripts` must find nothing else. `design/reference/*.html` also point at
these photos; leave the references as they are (their photos will be broken, the screenshots stay).

## Tests to write
The project has no unit-test runner; do not add one. Check by running:
- Backend: `cd ../citypark-go && make db-up && make seed && make run` (API on :3000). `curl localhost:3000/api/v1/categories`
  must return a non-empty list; if it is empty, stop and say so.
- `weightLabel` by hand in `node` or the dev server: `0 → ""`, `0.2 → "200 г"`, `0.25 → "250 г"`, `1 → "1 кг"`, `1.5 → "1,5 кг"`.
- `/menu`: the tabs are the backend categories; each tab shows its products; add to cart → header total and the
  cart panel change; `/checkout` shows the same prices; submitting an order reaches `/checkout/done`.
- Backend stopped: `/menu` and `/checkout` show «Меню временно недоступно», `/` renders without the delivery section,
  and «Обновить» works after the backend is started again.

## Done when
CONVENTIONS "Done means" passes, and with the backend running and a production build on port 3111:
`node scripts/qa/flows/menu.mjs`, `node scripts/qa/flows/home.mjs` and
`node scripts/qa/flows/checkout-and-contacts.mjs` print no unexpected values.

## Out of scope
Sending orders to `POST /orders` (orders still go to `submitLead`), online payment, showing category sale windows,
product search through the API (search stays in the browser), size variants from iiko modifiers, and any change in
`../citypark-go/`.
