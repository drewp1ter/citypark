# Task 07: Clear button for text inputs

## Goal
A text input can show a «×» button that empties it in one click, like HeroUI's `isClearable`. The menu search gets
it first.

## Depends on
None.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/shared/ui/fields.tsx`, `src/shared/ui/icon-button.tsx`, `src/shared/ui/icons.tsx`,
`src/shared/ui/index.ts`, `src/pages/menu/ui/menu-catalog.tsx`, `src/app/styles/globals.css`,
`scripts/qa/flows/menu.mjs` (the search part).

## Background
- The menu search (`menu-catalog.tsx`, around line 36) is a pill `<label>` with `SearchIcon` and a controlled
  `<input type="search">` (`query` state). Chrome draws its own small cancel «×» in search inputs; it is not styled
  and too small to tap.
- `TextField` is not a client component; it is rendered inside client forms. Uncontrolled fields (`defaultValue`)
  can only be cleared through a ref.
- `CloseIcon` already exists in `icons.tsx`.

## Files to create
- `src/shared/ui/clear-button.tsx`: `ClearButton` (Spec 1).

## Files to change
- `src/shared/ui/fields.tsx`: `clearable` on `TextField` (Spec 2).
- `src/shared/ui/index.ts`: export `ClearButton`.
- `src/pages/menu/ui/menu-catalog.tsx`: use `ClearButton` (Spec 3).
- `src/app/styles/globals.css`: hide the native search cancel button (Spec 3).
- `scripts/qa/flows/menu.mjs`: Spec 4.

## Spec

### 1. `ClearButton`
- `<button type="button">` with `CloseIcon` (about 16 px), `aria-label="Очистить поле"` by default (prop `label` to
  override), `onClick` prop, `className` prop.
- Size: a 44 × 44 px hit area (`size-11`), the visible circle can be smaller; `text-text-muted`, hover `text-text`;
  focus ring as everywhere (2 px `outline-accent` with offset). Inverse tone: `text-inverse-muted`, hover
  `text-inverse-text` (prop `tone?: FieldTone`).
- No state; it does not need `"use client"` itself (the parent passes `onClick`).

### 2. `TextField clearable`
- New props: `clearable?: boolean`, `onClear?: () => void`.
- When `clearable`, render the input inside a `relative` wrapper with `pr-12` on the input and `ClearButton`
  absolutely positioned at the right, vertically centred. Show the button only while the input has text.
- Controlled input (`value` given): show the button when `value` is not empty; the click calls `onClear`.
- Uncontrolled: track "has text" from `onChange`/`defaultValue`; the click sets `input.value = ""` through a ref,
  dispatches `new Event("input", { bubbles: true })` so React and the form see it, and calls `onClear` if given.
- After a click, focus goes back to the input.
- This needs state, so put the clearable variant in a small client component (`ClearableInput` in
  `src/shared/ui/clearable-input.tsx`, not exported from `index.ts`) that `TextField` renders only when `clearable`.
  `TextField` without `clearable` must stay exactly as it is now (no `"use client"` in `fields.tsx`).
- No existing form field gets `clearable` in this task.

### 3. Menu search
- Put `ClearButton` at the end of the search pill, shown while `query` is not empty; the click sets `query` to `""`
  and focuses the input (`useRef`). The pill height must not change (the button's 44 px fits in `min-h-12`; reduce
  the pill's right padding when the button shows if needed, so the text does not jump).
- In `globals.css` hide the browser's cancel button for search inputs:
  `input[type="search"]::-webkit-search-cancel-button { appearance: none; }` with a one-line comment.

### 4. QA in `menu.mjs`
In the search part, after `search no match`:
- `out("search: clear button shown", …)` → `true`.
- Click «Очистить поле»: `out("search: after clear (value / cards / focus)", …)` → empty value, the same card count as
  before the search, focus on the search input (`document.activeElement.type === "search"`) → `"" / N / true`.
- `out("search: clear button hidden when empty", …)` → `false`.
Keep the existing `await search.fill("")` working after these lines.

## Tests to write
No unit-test runner; check by hand at 390 and 1440 px, light and dark:
- type in the menu search: «×» appears; click it: text gone, all dishes back, cursor in the input;
- Tab reaches «×» after the input and `Enter` clears;
- no native cancel «×» in Chrome;
- the forms on `/checkout`, `/contacts` and `/restaurant` look and work exactly as before.

## Done when
CONVENTIONS "Done means" passes, and with the backend running and a production build on port 3111:
`node scripts/qa/flows/menu.mjs`, `npm run qa:a11y` and `npm run qa:keyboard` print no unexpected values.

## Out of scope
Clear buttons in existing form fields, textarea, the Select (task 06), the search logic itself, and any change in
`../citypark-go/`.
