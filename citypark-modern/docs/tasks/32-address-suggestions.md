# Task 32: Address suggestions on checkout

## Goal
While the customer types «Улица и дом» on `/checkout`, show up to 5 street/house suggestions from DaData for the
selected «Населённый пункт». Choosing one fills the field. Typing any text is still allowed. Without
`DADATA_API_KEY` the field works as a plain text field, as now.

## Depends on
26, 28 (checkout form). No backend change.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/checkout/ui/checkout-form.tsx`, `src/pages/checkout/api/place-order.ts`,
`src/shared/ui/select.tsx` and `src/shared/ui/fields.tsx` (look, `controlClass`, `fieldLabelClass`, errors),
`src/shared/config/env.ts`, `src/entities/dish` (`getDeliveryTowns`), React Aria `ComboBox` docs
(https://react-spectrum.adobe.com/react-aria/ComboBox.html: `allowsCustomValue`, `menuTrigger`, controlled
`inputValue`, `name`), Next guide for Route Handlers in `node_modules/next/dist/docs/`, DaData address suggest API
(https://dadata.ru/api/suggest/address/: `locations`, `restrict_value`, `from_bound`/`to_bound`).
Old implementation for behaviour only: `../citypark_frontend/src/features/OrderDetails/components/DaData/DaData.js`.

## Background
- DaData: `POST https://suggestions.dadata.ru/suggestions/api/4_1/rs/suggest/address`, header
  `Authorization: Token <key>`, body `{ query, count: 5, locations: [...], restrict_value: true,
  from_bound: { value: 'street' }, to_bound: { value: 'house' } }`. With the bounds, `suggestion.value` is only
  «ул Гоголя, д 61» — no string cutting like the old site did.
- `locations` for a town: `[{ region: 'Краснодарский', city: town }, { region: 'Краснодарский', settlement: town }]`
  (Белореченск is a city, the villages are settlements). Check the real town names from the backend's delivery
  prices; if they have prefixes («п.», «х.»), strip them before sending.
- The browser calls our own Next route, not DaData and not citypark-go, so «the browser never calls the backend»
  still holds. The route only accepts towns from `getDeliveryTowns()`, so it is not an open proxy for the key.

## Files to create
- `src/shared/api/dadata.ts` (server only): `suggestAddress(query: string, town: string): Promise<string[]>`.
  Reads `serverEnv().DADATA_API_KEY`; no key → `[]`. `AbortSignal.timeout(3000)`. Non-2xx / network error →
  `console.error` in English and throw.
- `src/app/api/address-suggest.ts`: `export async function GET(request: Request)`. Query params `q` (trimmed,
  3–120 chars, else `200 []`) and `town` (must be in `getDeliveryTowns()`, else `400`). Returns
  `Response.json(string[])` with `Cache-Control: private, max-age=300`; DaData failure → `502 []`.
- `app/api/address-suggest/route.ts`: only re-exports `GET` (route files re-export from `src`).
- `src/features/address-suggest/` slice: `ui/address-field.tsx` (client) + `index.ts` exporting `AddressField`.

## Files to change
- `src/shared/config/env.ts`: optional `DADATA_API_KEY` (empty → `undefined`, like `YANDEX_MAPS_API_KEY`).
- `.env.example`: `DADATA_API_KEY=` with a comment (dadata.ru cabinet, «API-ключ»; read on the server;
  empty = no suggestions).
- `src/shared/api/index.ts`: export `suggestAddress`. Make sure no client component imports `@/shared/api`
  (it would pull server code into the bundle); if one does, import `dadata.ts` in the route by its path instead
  and say why in a comment.
- `src/pages/checkout/ui/checkout-form.tsx`: replace the «Улица и дом» `TextField` with `AddressField`.
- `src/pages/checkout/ui/checkout-page.tsx`: pass `suggestions={Boolean(serverEnv().DADATA_API_KEY)}` down.

## Spec: `AddressField`
```tsx
'use client'
export function AddressField(props: {
  label: ReactNode         // «Улица и дом»
  name: string             // 'street' — the form posts it as now
  town: string             // the selected «Населённый пункт»
  suggestions: boolean     // false → plain TextField, no requests
  defaultValue?: string    // values?.street after a server error
  error?: string
  className?: string
}): JSX.Element
```
- React Aria `ComboBox` with `allowsCustomValue`, `menuTrigger='input'`, controlled `inputValue`, `name`.
  Same look as `TextField` / `Select` (reuse `controlClass`, `fieldLabelClass`, `fieldErrorClass`, the Select
  popover/list classes). Tokens only.
- Fetch `/api/address-suggest?q=&town=` 300 ms after the last keystroke, only for ≥ 3 characters; abort the
  previous request (`AbortController`). Errors → no list, no message (hint only).
- Popover header: «Выберите вариант или продолжите ввод» (text from the old site). Highlight the typed words in
  each item with `<mark>` styled by tokens (no new dependency).
- Choosing an item sets the input to it and closes the list; the caret goes to the end so the customer can add
  «, корп. 2».
- When `town` changes, clear the input (old behaviour: a street from another town is wrong) — but not on the
  first render, so `defaultValue` survives a server error.
- Keep `autoComplete='off'` on the input (the browser's own autofill list would cover ours).
- `place-order.ts` and `schema.ts` stay as they are.

## Done when
```
npm run check
npm run build
DADATA_API_KEY=<key> npm start -- -p 3111
node scripts/qa/flows/checkout-and-contacts.mjs
npm run qa:a11y
npm run qa:keyboard
```
- Add to `checkout-and-contacts.mjs` (only when `QA_DADATA=1`, it calls DaData): type «Гоголя» → at least one
  option containing «Гоголя»; ArrowDown + Enter fills the field. Without the flag the existing steps
  (`fill('Гоголя 10')`) must pass unchanged — the label stays «Улица и дом».
- Screenshots of `/checkout` at 390 and 1440 px, both themes, with the list open.
- With `DADATA_API_KEY=` empty: no requests to `/api/address-suggest` in the network log, the field works as before.
- `curl '/api/address-suggest?q=Гоголя&town=Москва'` → 400.
- If the key is missing in `.env`, stop and write that in `PROGRESS.md` (the old key is in
  `../citypark_frontend/.env` as `REACT_APP_DADATA_API_KEY`; the owner must confirm it still works).
