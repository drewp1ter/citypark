# Task 06: Dropdown Select on React Aria

## Goal
Every select on the site opens our own dropdown list (styled with the tokens, in both themes) instead of the browser's
native list. Behaviour and accessibility (keyboard, ARIA, focus, positioning) come from the headless library
`react-aria-components`; the look is ours. Forms keep sending the same values.

## Depends on
None.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/shared/ui/fields.tsx`, `src/shared/ui/radio-pills.tsx`, `src/shared/ui/icons.tsx`,
`src/shared/ui/index.ts`, `src/features/book-table/ui/table-booking-form.tsx`,
`src/pages/checkout/ui/checkout-form.tsx`, `src/features/search-rooms/ui/stay-dates-bar.tsx`,
`src/app/styles/tokens.css`, `scripts/qa/flows/restaurant.mjs`, `scripts/qa/flows/checkout-and-contacts.mjs`,
`scripts/qa/flows/rooms.mjs`, and the React Aria Select docs: https://react-spectrum.adobe.com/react-aria/Select.html
(also the `.d.ts` files in `node_modules/react-aria-components/dist/` after installing).

## Background
- `SelectField` in `fields.tsx` wraps a native `<select>` in `FieldShell` (label above, error under it with
  `role="alert"`, `aria-invalid` + `aria-describedby` on the control). `tone="inverse"` is used in the dark booking
  block on `/restaurant`.
- Four `SelectField`s exist: «Гостей» and «Зал» in `table-booking-form.tsx` (uncontrolled, `defaultValue`), «Когда» and
  «Населённый пункт» in `checkout-form.tsx` (controlled, `value` + `onChange`). One more native `<select name="guests">`
  is in `stay-dates-bar.tsx` (rooms page; a GET form, hidden below 768 px but still submitted).
- The forms post through Server Actions (`FormData`), and the rooms bar through the URL. React Aria's `Select` with a
  `name` prop renders a hidden native field, so the submitted key and value can stay the same. Check what it renders
  in the installed version (a hidden `<select>` or `<input type="hidden">`) before you update the QA selectors.
- `CLAUDE.md` (Stack) and `docs/PLAN.md` (line 9) say "No UI kit". A headless library is not a UI kit, but the rule
  must say so explicitly.

## Files to create
- `src/shared/ui/select.tsx`: `Select`, a client component (Spec 2).

## Files to change
- `package.json` / `package-lock.json`: `npm install react-aria-components@1.22.1` (exact version, no `^`).
- `src/shared/ui/fields.tsx`: share the label/error shell with `Select` (Spec 2); remove `SelectField` when nothing
  uses it.
- `src/shared/ui/index.ts`: export `Select`; drop `SelectField`.
- `src/features/book-table/ui/table-booking-form.tsx`, `src/pages/checkout/ui/checkout-form.tsx`,
  `src/features/search-rooms/ui/stay-dates-bar.tsx`: use `Select` (Spec 3).
- `CLAUDE.md`, `docs/PLAN.md`: the rule (Spec 1).
- `scripts/qa/flows/restaurant.mjs`, `scripts/qa/flows/checkout-and-contacts.mjs`, `scripts/qa/flows/rooms.mjs`: Spec 4.

## Spec

### 1. The rule
In `CLAUDE.md` replace the "No UI kit" line with:
`- No styled UI kit (no HeroUI, shadcn, MUI, etc.). Components are small and custom, built from the tokens. Complex
widgets (select, later combobox / date picker) may use headless \`react-aria-components\` for behaviour, styled only
with our tokens.` In `docs/PLAN.md` line 9 change "no UI kit" to "no styled UI kit (headless react-aria-components is
allowed)".

### 2. `Select` (client, `src/shared/ui/select.tsx`)
- Props:
  ```ts
  type SelectOption = { value: string; label: ReactNode; /** plain text for type-ahead when label is markup */ textValue?: string };
  type SelectProps = {
    label: ReactNode;
    name: string;
    options: readonly SelectOption[];
    value?: string;              // controlled
    defaultValue?: string;       // uncontrolled
    onValueChange?: (value: string) => void;
    error?: string;
    tone?: FieldTone;            // "default" | "inverse"
    className?: string;
    required?: boolean;
    disabled?: boolean;
  };
  ```
  Map them onto React Aria's `Select` (`selectedKey` / `defaultSelectedKey` / `onSelectionChange`, `name`,
  `isInvalid`, `isRequired`, `isDisabled`). Export `SelectOption` type too.
- Parts: React Aria `Label`, a `Button` trigger with `SelectValue` + `ChevronDownIcon` (rotate 180° while open;
  `motion-safe:transition-transform`), `Popover` → `ListBox` → `ListBoxItem`. Error text under the trigger with the
  same classes, `role="alert"` and `id` as `FieldShell` uses; React Aria's `FieldError` is fine if it renders the same
  look, otherwise keep ours and pass `aria-describedby` to the trigger.
- Label and spacing: identical to `FieldShell` for both tones (column, `gap-1.5`, `text-[13px] md:text-sm`, inverse =
  `text-inverse-muted`, default = `font-semibold`). Move the class strings into named constants in `fields.tsx` and
  export them, so the two stay the same; do not copy-paste them.
- Trigger: `controlClass` / the inverse control class, plus `flex items-center justify-between gap-2 text-left`.
  Invalid border: React Aria sets `data-invalid` on the field, not `aria-invalid` on the button, so add the matching
  `data-[invalid]:border-danger` / `border-danger-on-inverse` variant. Focus: 2 px `outline-accent` with offset on
  `data-[focus-visible]`.
- Popover: `bg-surface` (inverse tone: `bg-inverse-surface`), `border` in `border-border` / `border-inverse-border-strong`,
  `rounded-xs`, `p-1`, width = trigger width (`w-[var(--trigger-width)]`), `max-h-72 overflow-auto`, `offset={6}`.
  A shadow only if a token exists for it; do not add hex colours.
- Option: `min-h-11 rounded-[12px] px-3 flex items-center cursor-pointer`; hover/focused =
  `data-[focused]:bg-bg` (inverse: `bg-inverse-bg`); selected = `bg-inverse-bg text-inverse-text font-semibold` like
  the checked pill in `RadioPills` (inverse tone: `bg-accent text-on-accent`). Selected + focused must still be
  visibly focused.
- Enter/exit animation only under `motion-safe:` (fade + small translate); nothing under reduced motion.
- Both themes must look right; check `tone="inverse"` in the dark booking block on `/restaurant`.
- Mobile (390 px): the popover stays inside the viewport and options are ≥ 44 px tall.

### 3. Replace the selects
- `table-booking-form.tsx`: «Гостей» → `options={guestOptions.map((o) => ({ value: o, label: o }))}`, keep
  `defaultValue`, `tone`, `error`; «Зал» → options from `halls`. The posted values must not change («5–8», `cafe`, …).
- `checkout-form.tsx`: «Когда» (`asap` / `scheduled`, labels unchanged) and «Населённый пункт» (`city` / `outskirts`,
  the label keeps `typo(...)`, give it `textValue`). Use `value` + `onValueChange` in place of `value` + `onChange`.
- `stay-dates-bar.tsx`: the «Гости» select becomes `Select` with `name="guests"` and options `1`, `2`, `3-4`
  (label «3–4»). Keep the wrapper so it stays hidden below 768 px and keeps the left border; the visible label is
  «Гости». If the bar's label style differs from `FieldShell`, pass it through `className` instead of adding a new tone.
- After this no native `<select>` is left in `src/` (`grep -rn "<select" src` prints nothing).

### 4. QA
- Replace `.selectOption(...)` calls: click the trigger (`getByRole("button", { name: /Гостей/ })` or by label —
  check what React Aria exposes) then `getByRole("option", { name: "5–8" }).click()`.
- `restaurant.mjs`: add `out("select: keyboard", …)`: focus the «Гостей» trigger, press `Enter`, `ArrowDown`,
  `Enter`, print the trigger text → the second option; then open again, press `Escape`, print whether the listbox is
  gone and focus is back on the trigger → `true / true`. The browser cannot see the posted values, so the existing
  "valid -> success" line stays the proof that the action accepted them.
- The "unlabelled" and "small" checks in `restaurant.mjs` must stay at 0 / empty; if the hidden native field is counted,
  exclude it there by its React Aria wrapper (hidden fields are not visible, so `vis()` should already skip them).
- `checkout-and-contacts.mjs`: switch «Населённый пункт» and «Когда» through the dropdown; the delivery price and the
  time field must react exactly as before.
- `rooms.mjs`: `select[name=guests]` reads must read the hidden field the new `Select` renders; the printed values
  (`guests param -> select`, `mobile submit keeps guests`) must stay the same.

## Tests to write
No unit-test runner; check by hand at 390 and 1440 px, light and dark:
- every select opens with a click, with `Enter`/`Space`/`ArrowDown` on the trigger, closes with `Escape` and on a
  click outside; type-ahead jumps to an option;
- the selected option is marked, the trigger shows it, the form submits it (booking success, checkout price, rooms URL);
- a server error on a select shows the red border and the message, and keeps the chosen value;
- `tone="inverse"` on `/restaurant` looks right; the popover never goes off-screen at 390 px.

## Done when
CONVENTIONS "Done means" passes, `grep -rn "<select" src` prints nothing, and with a production build on port 3111:
`node scripts/qa/flows/restaurant.mjs`, `node scripts/qa/flows/home.mjs`, `node scripts/qa/flows/checkout-and-contacts.mjs`,
`node scripts/qa/flows/rooms.mjs`, `npm run qa:a11y`, `npm run qa:keyboard` and `npm run qa:responsive` print no
unexpected values.

## Out of scope
Combobox / autocomplete, multi-select, a custom date or time picker, the clear button (task 07), restyling text
inputs, and any change in `../citypark-go/`.
