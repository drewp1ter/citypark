# Task 15: Hotel prices

## Goal
«Цены отеля» edits the hotel's fees (breakfast delivery, late checkout per hour, smoking fine) and the breakfast menu.

## Depends on
14, and citypark-go task 52.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/features/settings/` (form pattern from task 14),
`../citypark-go/docs/API.md` (section `GET /hotel/prices`, `PUT /hotel/prices`), `../GLOSSARY.md` (Price placeholder).

## Background
These numbers fill the placeholders in the hotel texts (`{{breakfastDelivery}}`, `{{lateCheckoutPerHour}}`,
`{{smokingFine}}`), so a change here updates every sentence that mentions the price. `PUT` replaces the fees and the
whole breakfast list.

## Files to create
- `src/features/hotel-prices/api.ts`, `src/features/hotel-prices/model/form.ts` (+ test),
  `src/features/hotel-prices/ui/hotel-prices-page.tsx`, `ui/breakfast-list.tsx`, `e2e/hotel-prices.spec.ts`

## Files to change
- `src/routes/_app/content.hotel-prices.tsx`

## Spec
Card «Сборы»: «Доставка завтрака в номер, ₽», «Поздний выезд, ₽ в час», «Штраф за курение, ₽» (numbers ≥ 0,
≤ 100 000, 2 decimals); under each, muted text «Используется в текстах как {{name}}». Card «Меню завтраков»: rows with
«Название» (required, ≤ 80), «Короткое название для телефона» (≤ 40), «Цена, ₽», checkbox «Только на компьютере»,
up/down and «Удалить»; «Добавить блюдо»; max 30. One «Сохранить», unsaved-changes blocker, field errors mapped
(`breakfast[2].price`).

## Tests to write
- `form.test.ts`: validation of fees and items; `toRequest` keeps item order and parses `250,5`.
- `e2e/hotel-prices.spec.ts`: change the smoking fine, save, reload → kept; restore; add a breakfast item, move it up,
  save → order kept; delete it, save.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/hotel-prices.spec.ts`.

## Out of scope
Room prices (task 11).
