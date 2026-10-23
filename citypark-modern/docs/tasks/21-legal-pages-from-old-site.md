# Task 21: Legal pages from the old site

## Goal
`/privacy` and `/offer` show the full texts of the old site, and `/documents` shows the scanned hotel documents,
instead of «Раздел скоро появится.». The «Документы» cards of the hotel page link to them.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/legal/ui/legal-page.tsx`, `src/pages/legal/index.ts`,
`app/(site)/privacy/page.tsx` (the offer and documents routes are the same), `src/entities/hotel/model/hotel.data.ts`,
`src/pages/hotel/ui/guest-info.tsx`, `src/pages/news-article/ui/` (look of a long text page; reuse its classes).

Source texts (old CRA site, read only):
- `/workspace/citypark_frontend/src/features/ProvatePolicy/components/PrivatePolicy/PrivatePolicy.js`
- `/workspace/citypark_frontend/src/features/PublicOffer/components/PublicOffer/PublicOffer.js`
- `/workspace/citypark_frontend/src/features/Hotel/components/Documentation/Documentation.js` and its `images/`

## Background
CLAUDE.md "Known gaps" says the legal pages have no content yet; the old site has it. Both texts are nested ordered
lists (`<ol>` / `<li>` with a title and sub-items, up to three levels) plus a few paragraphs. They are static: the
owner did not ask to edit them in the admin, so they are typed data in the page slice, not backend content.

## Files to create
- `src/pages/legal/model/types.ts`: `LegalItem`, `LegalDocument` (Spec 1).
- `src/pages/legal/model/privacy.data.ts`, `src/pages/legal/model/offer.data.ts`: the two texts.
- `src/pages/legal/model/scans.data.ts`: the list of scans (Spec 3).
- `src/pages/legal/ui/legal-list.tsx`: renders `LegalItem[]` recursively.
- `src/pages/legal/ui/documents-page.tsx`: `DocumentsPage`.
- `public/documents/*.jpg`: the scans (Spec 3).

## Files to change
- `src/pages/legal/ui/legal-page.tsx`: `LegalPage({ document }: { document: LegalDocument })`.
- `src/pages/legal/index.ts`: export `LegalPage`, `DocumentsPage`, `privacyPolicy`, `publicOffer` (or export two
  ready pages `PrivacyPage`, `OfferPage`; choose one and keep the route files thin re-exports).
- `app/(site)/privacy/page.tsx`, `app/(site)/offer/page.tsx`, `app/(site)/documents/page.tsx`.
- `src/entities/hotel/model/hotel.data.ts`: `href` and `meta` of `guestDocuments` (Spec 4).
- `CLAUDE.md`: remove the "Legal pages … placeholder heading" line from "Known gaps".

## Spec

### 1. Types
```ts
export type LegalItem = {
	/** Text of the item. Plain text; `typo()` is applied when rendering. */
	text: string
	items?: readonly LegalItem[]
}

export type LegalDocument = {
	title: string
	/** Paragraphs before the numbered list. */
	intro: readonly string[]
	items: readonly LegalItem[]
}
```

### 2. Texts
Copy every paragraph and list item word for word. Remove only markup (`<br />`, `className`, `&nbsp;` → a space).
Fix obvious typos the same way as in other tasks (double spaces, a missing letter), nothing else. Titles: «Политика
обработки персональных данных» (`/privacy`) and «Публичная оферта» (`/offer`). Render: `h1` = title; intro as `<p>`;
items as nested `<ol>` with numbering «1.», «1.1.», «1.1.1.» (CSS counters, so screen readers read real list
items). Max text width as on the news article page. `metadata.title`: «<title> — Сити Парк».

### 3. Documents
Copy to `public/documents/` with these names: `report_1.jpg`, `report_2.jpg` → `classification-1.jpg`,
`classification-2.jpg`; `f1.jpg` … `f8.jpg` → `privacy-policy-1.jpg` … `privacy-policy-8.jpg`. Do not copy the old
thumbnails (`t*.jpg`, `*t.jpg`); `next/image` makes the small sizes.
`/documents`: `h1` «Документы»; two sections with `id="classification"` and `id="privacy-policy"`, headings
«Выписка из единого реестра объектов классификации в сфере туристической индустрии» and «Политика ООО «Олимп» в
отношении обработки персональных данных». Each page of a scan is a card with a thumbnail (`next/image`, width 1264 or
1240, `sizes` for a 2-column mobile / 4-column desktop grid) that links to the full JPG in a new tab, `alt`
«<section title>, страница N». Under the second heading add a link «Текст политики» to `/privacy`.

### 4. Hotel page cards
`guestDocuments`: `Выписка из реестра объектов классификации` → `href: '/documents#classification'`,
`meta: typo('2 стр.')`; `Политика обработки персональных данных` → `href: '/privacy'`, `meta: typo('8 стр.')`;
`Публичная оферта` → `href: '/offer'`, `meta: 'Текст'`. Update the doc comment of `GuestDocument.href` (the files
exist now).

## Tests to write
No unit-test runner. On a production build:
- `/privacy` and `/offer` at 390 and 1440 px, both themes: one `h1`, numbering 1. / 1.1. / 1.1.1., no horizontal scroll.
- Compare the number of list items with the old files: `grep -c '<li' <old file>` equals the count of `li` on the page.
- `/documents`: 10 thumbnails, each opens its JPG; `/documents#classification` scrolls to the first section.
- `/hotel`: the three document cards link to the right pages.

## Done when
CONVENTIONS "Done means" passes; `npm run qa:a11y` and `npm run qa:responsive` print no unexpected values (add
`/privacy`, `/offer`, `/documents` to their page lists if they are not there).
Write under the task line in PROGRESS.md: «Owner: check the offer is current (domain cityparkvip.ru, payment
methods) and send the classification certificate if it changed.»

## Out of scope
Editing these texts in the admin, PDF files, redirects from the old URLs.
