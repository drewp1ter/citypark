# Task 27: Remove the table booking form

## Goal
The site no longer has a «Забронировать столик» form. Guests book a table by phone. Both forms are removed: the card
in the «Ресторан» section of the home page and the dark `#booking` section at the bottom of `/restaurant`. Buttons
that led to the form now call the restaurant or open the menu. The `features/book-table` slice is deleted.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`, `CLAUDE.md` (sections «Forms (how they work)» and «Code structure (FSD)»),
`src/features/book-table/` (all files), `src/pages/home/ui/restaurant-section.tsx`,
`src/pages/restaurant/ui/` (`restaurant-page.tsx`, `hero.tsx`, `booking-section.tsx`),
`src/pages/news-article/ui/article-page.tsx`, `src/widgets/mobile-quick-actions/ui/mobile-quick-actions.tsx`,
`src/shared/ui/fields.tsx`, `src/shared/api/leads.ts`, `scripts/qa/flows/home.mjs`, `scripts/qa/flows/restaurant.mjs`.

## Background
This is the owner's decision (2026-10-28). It differs from the design references, which still show the form, so do
not copy it back from `design/reference/`. Phone numbers come from `getSiteSettings()` (server) or
`useSiteSettings()` (client), as elsewhere.

## Files to delete
- `src/features/book-table/` (the whole slice).
- `src/pages/restaurant/ui/booking-section.tsx`.

## Files to change
- `src/pages/home/ui/restaurant-section.tsx`
- `src/pages/restaurant/ui/restaurant-page.tsx`, `src/pages/restaurant/ui/hero.tsx`
- `src/pages/news-article/ui/article-page.tsx`
- `src/widgets/mobile-quick-actions/ui/mobile-quick-actions.tsx`
- `src/shared/api/leads.ts`, `src/shared/ui/fields.tsx` (only if `tone='inverse'` becomes unused)
- `scripts/qa/flows/home.mjs`, `scripts/qa/flows/restaurant.mjs`
- `CLAUDE.md`, `docs/PLAN.md`, `design/pages.md`

## Spec

### Home page, «Ресторан» section (`restaurant-section.tsx`)
- Remove `<TableBookingForm …/>`, its comment and its import.
- Layout without the form: the grid is `md:grid-cols-2` at every desktop width (drop `lg:grid-cols-3`). The terrace
  photo keeps `md:col-span-2`. The banquet card and the 3D panorama card sit side by side under it, with the same
  height. There must be no empty cell at 1024 px or 1440 px.
- Mobile (below 768 px): the «Забронировать столик» button (`href='/restaurant'`) becomes a call button:
  label «Позвонить в ресторан», `href={phones.restaurant.href}` (make the component `async` and use
  `getSiteSettings()`). Same `variant` and `size`.

### Restaurant page
- `restaurant-page.tsx`: remove `<BookingSection />` and its import.
- `hero.tsx`: the «Забронировать столик» button (`href='#booking'`) becomes «Позвонить» with
  `href={phones.restaurant.href}`. The «Меню» button stays. If the hero is not `async` yet, make it `async` and use
  `getSiteSettings()`.

### News article page (`article-page.tsx`)
Remove the outline «Забронировать столик» button. «Меню доставки» stays.

### Mobile quick actions (`mobile-quick-actions.tsx`)
On `/restaurant` the second button «Столик» (`#booking`) becomes «Меню» with `href='/menu'`, same `variant` and
`size`. The first button stays «Позвонить» (restaurant phone). Update the doc comment above the component.

### Shared
- `leads.ts`: remove `'table-booking'` from `LeadKind` and from the comment.
- `fields.tsx`: if nothing else uses `FieldTone` `'inverse'` after the deletion (`grep -rn "tone=" src`), remove
  that tone and its classes. If something still uses it, leave the file as it is.

### QA scripts
- `home.mjs`: remove the line `booking form date = today` and the whole `// ---- booking form` block.
- `restaurant.mjs`: remove the dark booking form checks, `home booking form (no hall) -> success`, and the mobile
  «Столик» check. Add these checks:
  - `/restaurant` at 1440 px: `#booking` count = `0`, and the hero link «Позвонить» has `href` = `tel:+79183119791`;
  - `/restaurant` at 390 px: the quick-actions link «Меню» has `href` = `/menu`;
  - `/` at 1440 px: the number of `form` elements inside the «Ресторан» section = `0`.

### Docs
- `CLAUDE.md`, «Forms (how they work)»: remove the sentence about `features/book-table`. The forms are now the
  checkout form and the contact form. In «Stack → Forms», remove «table booking».
- `docs/PLAN.md`: remove `book-table` from the slice list and from step 5. Add one line: the table booking form was
  removed on 2026-10-28 (owner's decision).
- `design/pages.md`: in the `/` and `/restaurant` rows, strike out the table booking form (`~~table booking form~~ —
  removed, task 27`), the same way as the chef blog card.

## Done when
```
npm run check
npm run build
grep -rn "book-table\|TableBookingForm\|BookingSection\|#booking" src app scripts   # no output
npm start -- -p 3111 &
node scripts/qa/flows/home.mjs
node scripts/qa/flows/restaurant.mjs
```
Also look at `/` and `/restaurant` at 390, 1024 and 1440 px in both themes: no empty grid cells and no gap where the
form used to be.
