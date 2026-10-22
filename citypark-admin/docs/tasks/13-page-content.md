# Task 13: Page content (home and hotel)

## Goal
«Главная» and «Отель» show every content block of that page, grouped by section in page order, each with the right
editor for its type. An admin changes a block's value and saves it.

## Depends on
08, 10, and citypark-go tasks 50 and 54.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `../GLOSSARY.md` (Content block, Block key, Price placeholder),
`src/shared/ui/image-field.tsx`, `src/shared/ui/rich-text-editor.tsx`, `../citypark-go/docs/API.md` (section
`### Content`).

## Background
Blocks are created by the backend; the admin only edits values (no add, no delete, no reordering). Types: `text`
(one line), `lines` (a list of plain lines), `richtext` (*short* profile), `image`. Each block has a Russian `label`, a
`section` and a `maxLength`. Prices are never typed: a text uses placeholders, listed by the backend docs
(`minRoomPrice` «Мин. цена номера», `lateCheckoutPerHour` «Поздний выезд, за час», `smokingFine` «Штраф за курение»,
`breakfastDelivery` «Доставка завтрака»). Blocks whose key ends with `-mobile` are the phone wording of the block
without the suffix.

## Files to create
- `src/features/page-content/api.ts`, `src/features/page-content/model/sections.ts` (+ test),
  `src/features/page-content/model/placeholders.ts`, `src/features/page-content/ui/page-content.tsx`,
  `ui/block-editor.tsx`, `ui/text-block.tsx`, `ui/lines-block.tsx`, `ui/richtext-block.tsx`, `ui/image-block.tsx`,
  `e2e/page-content.spec.ts`

## Files to change
- `src/routes/_app/content.home.tsx`, `src/routes/_app/content.hotel.tsx`: `<PageContent page='home' | 'hotel' />`.

## Spec
`sections.ts`:
```ts
export type BlockSection = { name: string; blocks: ApiBlock[] }
/** Groups blocks by section in the order they come from the API; pairs "<key>-mobile" right after "<key>". */
export function toSections(blocks: ApiBlock[]): BlockSection[]
```
Page: `PageHeader` with the page title and a link «Открыть на сайте ↗» (`VITE_SITE_URL` + `/` or `/hotel`). Sections
as HeroUI accordion items (all open on desktop, first open on phones), each block as a small card: label, a caption
with the key in muted monospace, the editor, and «Сохранить» (enabled only when changed; saving one block does not
touch the others). Unsaved blocks show a dot; leaving the page with unsaved blocks asks to confirm.
Editors:
- `text`: input with counter `n / maxLength`; «Вставить цену» menu for placeholders.
- `lines`: one input per line with «Удалить строку», «Добавить строку», and up/down buttons to reorder; counter of lines
  (1–50).
- `richtext`: `RichTextEditor` profile `short`, `maxLength`, `placeholders`.
- `image`: `ImageField` (aspect from the image's natural ratio, no `allowEmpty`).
`PUT /content/blocks/{key}` with `{ value }` or `{ image }`; field errors under the editor (`value`, `image.url`, …);
400 `неизвестная подстановка …` shown as is. Success → toast «Сохранено», the block's `updatedAt` («изменено
22.10.2026, 12:05») refreshes.

## Tests to write
- `sections.test.ts`: grouping keeps API order; `-mobile` pairs; a `-mobile` block without its main block stays where
  it is.
- `e2e/page-content.spec.ts` (needs task 54 data): on «Главная» change a `text` block, save, reload → kept; restore
  the original; on «Отель» add a line to «В каждом номере», save, then remove it and save; type `{{price}}` in a text
  block → the server's error is shown.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/page-content.spec.ts`.

## Out of scope
Adding blocks or pages, preview of the site inside the admin, history of changes.
