# Task 14: Site settings (contacts)

## Goal
«Настройки сайта» edits the contacts the whole site shows: four phones, address, e-mail, company details and social
links.

## Depends on
11, and citypark-go task 51.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/features/rooms/model/form.ts` (form pattern),
`../citypark-go/docs/API.md` (section `### Settings`).

## Background
`PUT /settings` replaces the whole object, so the form always sends everything it loaded. A phone has a display text
(«8 918 311-97-91», what visitors see) and a number for the call link («+79183119791»), plus an optional second
(landline) number.

## Files to create
- `src/features/settings/api.ts`, `src/features/settings/model/form.ts` (+ test), `src/features/settings/ui/settings-page.tsx`,
  `ui/phone-fields.tsx`, `ui/social-list.tsx`, `e2e/settings.spec.ts`

## Files to change
- `src/routes/_app/content.settings.tsx`

## Spec
Sections (cards): «Телефоны» (Ресторан и доставка, Отель, Бесплатно по России, Городской; each: «Подпись»,
«Короткая подпись», «Как показывать», «Номер для звонка», and a collapsible «Второй номер»: «Как показывать»,
«Номер для звонка»); «Адрес» (Индекс, Город, Улица и дом, with a preview line «352630, г. Белореченск, ул. Гоголя, 61»);
«E-mail»; «Юрлицо» (Название, ИНН, ОГРН); «Соцсети» (rows: Название, Короткая подпись, Ссылка (https://…), Как
показывать; add, remove, reorder; max 10). One «Сохранить» for the whole form (sticky at the bottom on phones), enabled
when something changed; unsaved-changes blocker. A button «Подставить номер из отображения» next to each number field
fills it from the display text (`displayToNumber`).
`form.ts`: `toForm`, `toRequest` (trims), `validateSettings` (mirrors the backend rules in `API.md`), `mapServerFields`
(dotted keys like `phones.hotel.number`, `social[0].url`), and
```ts
/** "8 918 311-97-91" → "+79183119791"; "8 800 100-24-41" → "88001002441"; "(86155) 2-25-38" → "+78615522538". */
export function displayToNumber(display: string): string
```
Rule for `displayToNumber`: keep digits; 11 digits starting with `8` and not `8800` → `+7` + the last 10; `8800…` →
as is; 10 digits → `+7` + digits; otherwise the digits.

## Tests to write
- `form.test.ts`: the three `displayToNumber` examples and `+7 918 311 97 91` → `+79183119791`; validation: bad e-mail,
  5-digit postcode, `http://` social link, second number half-filled → errors; `mapServerFields`.
- `e2e/settings.spec.ts`: change the e-mail to a test value, save, reload → kept; restore; invalid postcode → error
  under the field and nothing saved.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/settings.spec.ts`.

## Out of scope
Navigation, legal pages, opening hours.
