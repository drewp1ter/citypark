# Task 21: Working hours on «Доставка»

## Goal
On «Доставка», above the towns, an admin sees and changes the hours when the site accepts orders
(«Часы приёма заказов», 9:00–22:00 at the start). The backend and the site use the saved value.

## Depends on
20, and citypark-go task 63.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/features/delivery-prices/` (`api.ts`, `ui/delivery-prices-page.tsx`),
`src/features/settings/ui/settings-page.tsx` (save, toasts, server errors), `../citypark-go/docs/API.md`
(section `### Working hours`).

## Background
Until citypark-go task 63 the hours were `WORKING_HOURS_START/END` in the backend config. Now they are one row:
`GET /working-hours` (public) → `{ start: "09:00", end: "22:00", updatedAt }`, `PUT /working-hours` (admin) with
`{ start, end }`. Times are `HH:MM` with a leading zero, `start` before `end` (no overnight hours). Errors come back in
`fields.start` / `fields.end`. Orders are accepted from `start` until `end`; a «Ко времени» time may equal `end`.

## Files to create
- `src/features/working-hours/api.ts`
- `src/features/working-hours/model/form.ts`, `src/features/working-hours/model/form.test.ts`
- `src/features/working-hours/ui/working-hours-card.tsx`
- `e2e/working-hours.spec.ts`

## Files to change
- `src/shared/api/schema.d.ts`: regenerate with `npm run api:gen` (do not edit by hand).
- `src/features/delivery-prices/ui/delivery-prices-page.tsx`: render `<WorkingHoursCard />` between `PageHeader` and
  the search field.

## Spec

### `api.ts`
```ts
export const workingHoursQuery // queryOptions, key ['working-hours'], GET /working-hours
export function useSaveWorkingHours() // useMutation, PUT /working-hours with toRequest(form), invalidates ['working-hours']
```

### `model/form.ts`
```ts
export type WorkingHoursForm = { start: string; end: string }
export type WorkingHoursErrors = Partial<Record<keyof WorkingHoursForm, string>>
export function toForm(h: { start: string; end: string }): WorkingHoursForm // "09:00" stays "09:00"
export function normalizeTime(s: string): string | null // trims; "9:00" → "09:00", "09.30" → "09:30"; bad → null
export function validateWorkingHours(f: WorkingHoursForm): WorkingHoursErrors
export function toRequest(f: WorkingHoursForm): { start: string; end: string } // normalised; call after validation
export function mapServerFields(fields: Record<string, string> | undefined): WorkingHoursErrors
```
- Accepted input: `H:MM` or `HH:MM`, `.` also as the separator, hours 0–23, minutes 00–59.
- Messages: empty → «Укажите время»; bad format → «Время в формате ЧЧ:ММ, например 09:00»; both valid and
  `start >= end` → on `end`: «Время окончания должно быть позже начала».

### `ui/working-hours-card.tsx`
- A bordered card (same look as the cards in `delivery-prices-page.tsx`: `rounded-xl border border-border bg-surface
  p-4`, `mb-6`), heading «Часы приёма заказов» (`h2`), a muted line «Сайт принимает заказы и показывает эти часы.
  Заказ «ко времени» можно выбрать только внутри них.».
- Two HeroUI v3 `TextField`s side by side (stacked below 640 px): «С» (`name='start'`) and «До» (`name='end'`),
  `inputMode='numeric'`, placeholder `09:00`, `FieldError` from the errors. Then a «Сохранить» button (`isPending`
  while saving, disabled while the form equals the loaded value).
- Loading: one `Skeleton` of the card's height. Load error: `role='alert'` with the message and «Повторить», like the
  page.
- Save: validate locally; on success `toastSuccess('Сохранено')`; on 400 `mapServerFields`, and `toastError(error)`
  for any error. Initialise the form from the query once (`key` on the inner form, as the settings page does).
- Muted text under the button: «Изменено {formatDateTime(updatedAt)}» (`src/shared/lib/format.ts`).

## Tests to write
- `form.test.ts`: `normalizeTime` table: `'9:00'` → `'09:00'`, `' 10:30 '` → `'10:30'`, `'09.30'` → `'09:30'`,
  `'23:59'` ok, `'24:00'`, `'9:60'`, `'900'`, `'abc'`, `''` → `null`. `validateWorkingHours`: empty start → «Укажите
  время»; `'25:00'` → format message; `'12:00'`/`'12:00'` and `'13:00'`/`'12:00'` → end message; `'9:00'`/`'22:00'` →
  `{}`. `toRequest({ start: '9:00', end: '22:00' })` → `{ start: '09:00', end: '22:00' }`. `mapServerFields` picks
  `start` and `end`, ignores others.
- `e2e/working-hours.spec.ts`: sign in, open «Доставка», read the two current values; set «С» `10:00`, «До» `23:00`,
  «Сохранить» → toast «Сохранено»; reload → the fields show `10:00` and `23:00`; set «До» `09:00` → the end error, no
  request saved; restore the original values at the end (also in `afterEach` if the test fails midway).

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/working-hours.spec.ts e2e/delivery-prices.spec.ts`. After
saving 10:00–23:00 in the UI, `curl -s "$VITE_API_URL/working-hours"` (URL from `.env`) prints those times. Set them
back to 09:00–22:00 afterwards.

## Out of scope
- The backend (citypark-go task 63) and the site (citypark-modern task 29).
- Weekday schedules, holidays, overnight hours.
- Moving the hours to the «Настройки» page.
