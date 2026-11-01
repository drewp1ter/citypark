# Task 30: Editable delivery information

## Goal
Add cooking-time and outskirts delivery information controls to «Доставка». Staff can save one label/range used by the website without editing source files or duplicating town tariffs.

## Depends on
Admin task 29 and [backend task 91](../../../citypark-go/docs/tasks/91-delivery-settings.md), deployed at `VITE_API_URL`. [Website task 49](../../../citypark-modern/docs/tasks/49-delivery-settings.md) consumes the same contract; it is not an admin implementation dependency.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/features/working-hours/ui/working-hours-card.tsx`
- `src/features/delivery-prices/ui/delivery-prices-page.tsx`
- `../citypark-go/docs/tasks/91-delivery-settings.md`

## Files to create
- `src/features/delivery-settings/api.ts`
- `src/features/delivery-settings/model/form.ts`
- `src/features/delivery-settings/model/form.test.ts`
- `src/features/delivery-settings/ui/delivery-settings-card.tsx`
- `src/features/delivery-settings/ui/delivery-settings-form.tsx`
- `e2e/delivery-settings.spec.ts`

## Files to change
- `src/shared/api/schema.d.ts`: regenerate with `npm run api:gen`; never edit manually.
- `src/routes/_app/delivery.tsx`: compose delivery settings, working hours and existing delivery-price screen.
- `src/features/delivery-prices/ui/delivery-prices-page.tsx`: remove the existing direct working-hours feature import/card and allow route-composed content immediately after its page header (optional `children: ReactNode`).
- `docs/tasks/PROGRESS.md`: completion status.

## Spec
Use HeroUI v3, existing shared API/error/toast helpers and TanStack Query. Read the installed component documentation/types before using them. Do not introduce another UI kit or cross-feature imports.

- Route order: existing «Доставка» header/actions, working-hours card, new delivery-settings card, existing city search/list/dialogs. Route passes both settings cards as the delivery-price page's children.
- New card title: `Информация о доставке`.
- Description: `Время приготовления и стоимость доставки в СОТ и ДНТ показываются на сайте. Цены по городам и порог бесплатной доставки редактируются ниже.`
- Fields: `Время приготовления` (`cooking`), `Зона доставки` (`outskirtsLabel`), `Цена от, ₽` (`outskirtsMin`), `Цена до, ₽` (`outskirtsMax`).
- Help text: `Например: ~30 мин`; range help: `Информационный диапазон. Стоимость заказа рассчитывается по тарифу выбранного города.`
- Save button: `Сохранить`; success toast: `Сохранено`; retry button: `Повторить`.
- Use `GET /delivery-settings` and `PUT /delivery-settings` through `api`, with generated `DeliverySettings` / `DeliverySettingsInput` types. Query key `['delivery-settings']`; successful mutation updates/invalidates that query and adopts returned canonical values/time.
- Do not modify `/settings`, `/working-hours` or `/delivery-prices` from this form. No separate free-delivery threshold input; existing town editor is its source.
- Export `deliverySettingsQuery` and `useSaveDeliverySettings` from this feature's `api.ts`. Export `DeliverySettingsCard` from its UI file.
- Form model exports `DeliverySettingsForm` (all four fields as strings), `DeliverySettingsErrors`, `toForm`, `toRequest`, `validateDeliverySettings` and `mapServerFields`. API types come from the generated schema.
- Trim text, count Unicode characters (not UTF-16 units), require cooking 1–100 characters and label 1–60 characters. Numeric fields are required whole rubles, digits only after trimming, 0–100000 inclusive; reject fractions, signs and exponent notation. Require maximum ≥ minimum. Convert to numbers only after validation.
- Required errors: `поле не должно быть пустым`; text maximum errors and numeric bounds match backend task 91. Invalid numeric syntax: `Введите целое число от 0 до 100000`. Reversed range: `Максимальная цена должна быть не меньше минимальной` on `outskirtsMax`.
- Map known backend fields beside inputs; show backend error message in a toast. Never erase typed values on 400, transport failure or a background query refetch while dirty. After save success, reset dirty state from the returned values. Disable saving when unchanged or pending; prevent duplicate submissions.
- Show a skeleton while initially loading, and a visible error/retry state on load failure. Keep town list/working-hours cards independently usable if delivery settings fail.
- Responsive stacked fields at 390 px; compact grid on desktop. Both themes, labeled inputs, visible focus, and accessible pending/error states.

## Tests to write
- Vitest `form.test.ts`: API→form→request roundtrip; trim; Unicode lengths at/beyond bounds; empty/whitespace fields; zero/equal/max prices; reversed bounds; fractions, negative numbers, exponent notation, huge inputs and missing values; known/unknown server-field mapping.
- Playwright `delivery-settings.spec.ts`: at 390/1440 px in light/dark, sign in, load settings, save all fields, reload and confirm persistence, validate required/range errors, confirm unchanged/pending save state and preserved inputs after failed save. Verify loading/retry and dirty-form preservation during a query refetch.
- Verify existing working-hours card, city list/search and city tariff dialog still work after moving composition to the route.
- Read actual settings before live writes and restore them in `finally`; use existing authenticated e2e conventions. Simulated browser transport failures are allowed; never substitute a fake backend for successful persistence checks.
- Backend writes still require explicit session authorization under the workspace rules. If not authorized, report the write-dependent cases as pending; do not claim the task complete. No orders, payment or external notifications are needed.

## API documentation
No endpoint changes here. Regenerate API types from backend task 91 and use its documented contract. If it is unavailable or incompatible, record the blocker; do not invent a local API.

## Done when
`npm run api:gen`, `npm run check`, `npm run build` and `npm run e2e -- e2e/delivery-settings.spec.ts` pass against the existing configured backend. Original delivery settings are restored. Then tick this task and commit per CONVENTIONS.

## Out of scope
Backend implementation, website implementation, editable free-delivery duplicates, working-hours behavior changes and town tariff mutations outside explicitly authorized regression fixtures.
