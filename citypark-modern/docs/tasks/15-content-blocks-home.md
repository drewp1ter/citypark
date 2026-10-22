# Task 15: Content blocks, part 1: the entity and the home page

## Goal
A new entity `content` holds the editable copy of the site as **content blocks** (key, type, label, value) in one data
file, read through `getPageContent(page)`. The home page's copy comes from it. The page looks exactly as before; the
point is that every editable text and photo now has a block key, ready for the backend (citypark-go tasks 50 and 54).

## Depends on
13, 14.

## Read first
`docs/tasks/CONVENTIONS.md`, `../GLOSSARY.md` (Content block, Block key, Price placeholder, Rich text profile),
`../citypark-go/docs/tasks/50-content-blocks.md` (types, rules and API shape to mirror), `src/pages/home/ui/*.tsx`,
`src/shared/lib/photo.ts`, `src/shared/lib/placeholders.ts`, `src/entities/room/index.ts`.

## Background
Decided by the owner (ADR 0001): staff edit page copy in the admin. Lists with structure (rooms, news) and contacts
(site settings) have their own entities and are **not** blocks. Titles, captions and labels are `text` (plain);
paragraph-like body text is `richtext` (*short* profile); bulleted lists of plain items are `lines`; photos are
`image`. Prices never appear in a block: use placeholders (`{{minRoomPrice}}`).

## Files to create
- `src/entities/content/model/types.ts`, `src/entities/content/model/blocks.data.ts`,
  `src/entities/content/api/get-page-content.ts`, `src/entities/content/index.ts`
- `src/shared/ui/rich-text.tsx` (+ export from `src/shared/ui/index.ts`)

## Files to change
- `src/pages/home/ui/*.tsx`: read copy from blocks (Spec 3).

## Spec

### 1. Types and data
```ts
type BlockBase = { key: string; section: string; label: string; maxLength?: number }
export type Block = BlockBase &
	(
		| { type: 'text'; value: string }
		| { type: 'lines'; value: readonly string[] }
		| { type: 'richtext'; value: string }
		| { type: 'image'; image: Photo }
	)
export type ContentPage = 'home' | 'hotel'
```
`blocks.data.ts` exports `blocks: readonly Block[]`, ordered as the blocks appear on the page (top to bottom, desktop
order). This file is the source of citypark-go task 54's migration, so keep it plain data (no `typo()`, no function
calls, normal spaces; non-breaking spaces are added when rendering).

Keys: `home.<section>.<name>`, lowercase latin, digits and `-` (`home.hero.title`, `home.rooms.heading`). When the
desktop and mobile references word the same thing differently, make two blocks: `<name>` (desktop, label «…») and
`<name>-mobile` (label «… (мобильная версия)»). `section` is the Russian section name staff recognise
(«Первый экран», «Номера», «Ресторан», «Доставка», «Новости», «Контакты»). `maxLength`: 80 for titles and labels,
200 for captions and leads, 1000 for `richtext`.

### 2. Getter
```ts
export type PageContent = {
	/** Plain text with placeholders filled; '' when the key is missing. */
	text(key: string, values?: Readonly<Record<string, number>>): string
	lines(key: string): readonly string[]
	/** Sanitized HTML (the backend sanitizes; static data is trusted) with placeholders filled; '' when missing. */
	html(key: string, values?: Readonly<Record<string, number>>): string
	image(key: string): Photo | null
}
export async function getPageContent(page: ContentPage): Promise<PageContent>
```
A missing key or a key of another type returns the empty value and logs `console.error('content block missing: <key>')`
once per key. Use these accessors in Server Components and pass strings down to client components.

### 3. RichText
```tsx
/** Renders sanitized HTML from a content block or a news body. */
export function RichText({ html, variant, className }: { html: string; variant: 'short' | 'article'; className?: string })
```
A `div` with `dangerouslySetInnerHTML`; styles with tokens only: paragraph spacing, `strong`, `em`, links (underline,
the focus ring from CLAUDE.md), `ul`/`ol` with markers; `article` also styles `h2`, `h3`, `blockquote` (task 18 uses
it). Text inherits colour and size from the parent, so a block looks like the text it replaces. `typo()` cannot run
inside HTML: write `&nbsp;` in richtext values where the reference has non-breaking spaces. For `text` and `lines`,
the accessors apply `typo()` themselves.

### 4. Home page
Turn into blocks: every heading, lead, caption, badge text, card text and photo (with alt and `position`) of the hero,
rooms, restaurant, delivery and news sections, and the home-specific texts of the contacts block. Not blocks: the
address, phones and e-mail (site settings, task 19), room names, details and prices (rooms entity), news items, dish
data, navigation, button and form labels, `aria-label`s. The hero tile caption becomes
`от {{minRoomPrice}} / ночь` (desktop) and `от {{minRoomPrice}}` (mobile) with `values = { minRoomPrice }` from
`minPricePerNight(rooms)` (task 13). Headings with forced line breaks (the hero `h1`) are `lines` blocks rendered with
`<br />` between items.

## Tests to write
No unit-test runner. On a production build, `/` at 390 and 1440 px in both themes is pixel-identical to before
(compare screenshots taken before and after the change). Then rename one key in the data file → the slot renders
empty and the server log shows `content block missing`; revert.

## Done when
CONVENTIONS "Done means" passes; `node scripts/qa/flows/home.mjs` and `npm run qa:a11y` print no unexpected values;
`grep -n "'[А-Яа-яЁё]" src/pages/home/ui/*.tsx` lists only labels, `aria-label`s and other non-block strings (say in
the commit message which ones remain and why).

## Out of scope
The hotel and rooms pages (task 16), reading blocks from the backend (task 20), other pages (restaurant, contacts,
news, legal).
