# Task 07: Menu (read only) and the sync button

## Goal
«Меню» shows the categories and dishes as they came from iiko (name, price, weight, availability, photo) and has the
button «Обновить меню из iiko», which starts a sync and reports the result.

## Depends on
03, and citypark-go task 43.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/api/`, `src/shared/ui/toast.ts`, `../citypark-go/docs/API.md`
(sections `GET /categories`, `GET /products`, `POST /iiko/sync`).

## Background
iiko is the source of names, prices, descriptions and photos; the admin cannot change them (owner's decision). `GET /categories` hides categories hidden in iiko, but `GET /products`
returns their products too; group those under «Скрытые в iiko». A sync takes a few seconds; 409 means another sync is
already running (the server also syncs on a timer).

## Files to create
- `src/features/menu/api.ts`, `src/features/menu/model/group.ts` (+ test), `src/features/menu/ui/menu-page.tsx`,
  `ui/category-list.tsx`, `ui/dish-row.tsx`, `ui/sync-button.tsx`, `e2e/menu.spec.ts`

## Files to change
- `src/routes/_app/menu.tsx` (search params: `category?: number`, `q?: string`)

## Spec

### Data
Load `GET /categories` and every page of `GET /products?pageSize=500` (loop until a page is shorter than 500).
```ts
export type MenuGroup = { id: number | 'hidden'; name: string; available: boolean; dishes: ApiProduct[] }
/** Categories in API order, then one 'hidden' group for products whose category is not in the list (if any). */
export function groupMenu(categories: ApiCategory[], products: ApiProduct[]): MenuGroup[]
```

### Page
`PageHeader` «Меню» with `SyncButton`. Left (desktop) / top chips (phone): the groups with dish counts; selecting one
filters the list (`category` in the URL). A search field «Поиск блюда» filters by name on the client. Dish rows: photo
thumbnail 56×56 (`images[0].url`, an iiko URL shown as is; or a neutral placeholder icon with «Нет фото»), name, weight (`weight` kg →
«350 г», `measureUnit` as given if not kg), `formatRub(price)`, and a chip «Не в продаже сейчас» when
`isAvailableForOrder` is false. A note at the top: «Названия, цены, состав и фото меняются в iiko, затем нажмите «Обновить меню из iiko».»

### Sync button
«Обновить меню из iiko» → `POST /iiko/sync`, spinner and disabled while running (it may take ~10 s). 200 → toast
«Меню обновлено: категорий N, блюд M» and invalidate the menu queries. 409 → toast «Обновление уже идёт, попробуйте
через минуту». Other errors → `toastError`.

## Tests to write
- `group.test.ts`: products of a listed category go to it in category order; a product of an unknown category goes to
  «Скрытые в iiko»; no hidden group when there are none; empty input → `[]`.
- `e2e/menu.spec.ts`: the page lists the groups of the dev database (skip with a reason if the menu is empty); search
  narrows the list; the sync button: if iiko is not configured in dev the request fails, so assert only that clicking
  shows either the success toast or an error toast and the button becomes enabled again.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/menu.spec.ts`.

## Out of scope
Editing anything (iiko is the source), category photos.
