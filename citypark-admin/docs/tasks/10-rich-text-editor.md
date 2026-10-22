# Task 10: Rich text editor

## Goal
A reusable `RichTextEditor` (Tiptap) edits HTML with exactly the formatting of a rich text profile: *article* (news) or
*short* (room descriptions, hotel texts, page blocks). It also inserts price placeholders where a field allows them.

## Depends on
01.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `../GLOSSARY.md` (Rich text profile, Price placeholder),
`../citypark-go/docs/tasks/47-rich-text-sanitizer.md` (the backend allowlist this must match), the Tiptap v3 React
docs for the installed version (`@tiptap/react`, `@tiptap/starter-kit`, `@tiptap/extension-link`).

## Background
The backend sanitizes on save and removes anything outside the profile, so the editor must not offer more: otherwise
staff would see formatting disappear after saving. Placeholders like `{{smokingFine}}` are plain text in the HTML; the
site replaces them with prices.

## Files to create
- `src/shared/ui/rich-text-editor.tsx`, `src/shared/ui/rich-text-toolbar.tsx`, `src/shared/lib/rich-text.ts` (+ test)
- `src/routes/_app/dev.rich-text.tsx` (hidden demo page for e2e), `e2e/rich-text.spec.ts`

## Files to change
- `package.json`: `@tiptap/react`, `@tiptap/pm`, `@tiptap/starter-kit`, `@tiptap/extension-link`,
  `@tiptap/extension-placeholder` (and nothing else from Tiptap).

## Spec
```ts
export type RichTextProfile = 'article' | 'short'
type Props = {
	label: string
	value: string                 // HTML
	onChange: (html: string) => void
	profile: RichTextProfile
	placeholders?: readonly { name: string; label: string }[] // e.g. { name: 'smokingFine', label: 'Штраф за курение' }
	maxLength?: number            // visible characters; shows a counter «123 / 1000», red when over
	error?: string
}
export function RichTextEditor(props: Props): JSX.Element
```
- StarterKit configured per profile: both: paragraph, hard break, bold, italic, bullet list, ordered list, history;
  `article` also heading levels 2 and 3 and blockquote. Disable everything else (code, codeBlock, strike, underline,
  horizontalRule, heading 1/4–6). Link extension: `openOnClick: false`, protocols `http`, `https`, `mailto`, `tel`,
  relative `/…` allowed; no `target`/`rel` attributes added by the editor (the backend adds them).
- Toolbar (icon buttons with `aria-label` and `aria-pressed`): Жирный, Курсив, Маркированный список, Нумерованный
  список, Ссылка (a small popover with «Адрес ссылки», «Применить», «Убрать ссылку»), and for `article`: Заголовок 2,
  Заголовок 3, Цитата. Plus «Отменить»/«Повторить». If `placeholders` is set: a menu «Вставить цену» that inserts
  `{{name}}` at the cursor; show the list of available placeholders under the editor as help text.
- Pasting from Word or a website keeps only the allowed formatting (Tiptap does this with the schema; check it).
- `onChange` gets `normalizeHtml(editor.getHTML())`.
`rich-text.ts`:
```ts
/** "<p></p>" and whitespace-only → ""; otherwise unchanged. */
export function normalizeHtml(html: string): string
/** Visible character count, same idea as the backend's TextLength (tags stripped, entities decoded). */
export function textLength(html: string): number
```
Styling: editor content uses the same element styles as the site's `RichText` would (paragraph spacing, lists with
markers, headings smaller than the page `h1`), from Tailwind/HeroUI tokens, both themes. Visible focus ring around the
editor.

## Tests to write
- `rich-text.test.ts`: `normalizeHtml('<p></p>')` → `''`; `textLength('<p>При&nbsp;вет</p>')` → 7;
  `textLength('<ul><li>ab</li></ul>')` → 2.
- `e2e/rich-text.spec.ts` on the demo page (shows both profiles and the current HTML in a `<pre>`): type text, make a
  word bold → HTML contains `<strong>`; the `short` editor has no heading button; the `article` one makes an `h2`;
  insert a link; paste HTML with `<h1>` and `style` into `short` → no `h1`, no `style`; insert a placeholder →
  `{{smokingFine}}` in the HTML.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/rich-text.spec.ts`.

## Out of scope
Images, tables or colours in text, Markdown, collaboration.
