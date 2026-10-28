# Task 30: Yandex map on the home and contacts pages

## Goal
Replace the two «Яндекс Карта» placeholders with a real Yandex map (JS API v3) with one accent pin on
ул. Гоголя, 61. The map follows the site theme (light/dark). Without an API key the placeholders stay as they are.

## Depends on
29 (only for order; no backend change).

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/home/ui/contacts-block.tsx`, `src/pages/contacts/ui/contacts-page.tsx`,
`src/shared/config/env.ts`, `src/widgets/site-header/ui/theme-toggle.tsx` (how `useTheme` is used),
Yandex docs: https://yandex.ru/dev/jsapi30/doc/ru/ (quick start, `YMap`, `YMapDefaultSchemeLayer` `theme`,
`YMapMarker`, `behaviors`) and the `@yandex/ymaps3-types` README.

## Background
- JS API v3 loads from `https://api-maps.yandex.ru/v3/?apikey=<KEY>&lang=ru_RU`, then `await ymaps3.ready`.
  Coordinates are **[longitude, latitude]**.
- The key is public (it is in the browser anyway); the owner restricts it by HTTP Referer in the Yandex developer
  cabinet. It is read on the **server** (`YANDEX_MAPS_API_KEY`, not `NEXT_PUBLIC_`) and passed as a prop, so the
  same build works with another key.
- This is not a call to our backend, so the rule "the browser never calls the backend" is not broken.

## Files to create
- `src/shared/ui/yandex-map.tsx` (client component)
- `src/shared/lib/load-ymaps3.ts` (loads the script once)

## Files to change
- `package.json`: devDependency `@yandex/ymaps3-types` (types only, no runtime package).
- `src/shared/config/env.ts`: optional `YANDEX_MAPS_API_KEY` (empty → `undefined`, like `UPLOADS_ORIGIN`).
- `src/shared/config/site.ts`: `map: { center: [39.875414, 44.759943] as const, zoom: 16 }`.
- `.env.example`: `YANDEX_MAPS_API_KEY=` with a comment (where to get it, restrict by Referer, empty = placeholder).
- `src/shared/ui/index.ts`, `src/shared/lib/index.ts`: export the new names.
- `src/pages/home/ui/contacts-block.tsx`, `src/pages/contacts/ui/contacts-page.tsx`: render the map.
- `CLAUDE.md` "Known gaps": remove the map line. `design/components.md`: ContactsBlock note → map is Yandex JS API v3.

## Spec

### `shared/lib/load-ymaps3.ts`
```ts
/** Adds the JS API v3 script once and resolves when `ymaps3.ready` resolves. Rejects on script error. */
export function loadYmaps3(apiKey: string): Promise<typeof ymaps3>
```
Module-level promise cache; a failed load clears the cache so a later mount can retry.

### `shared/ui/yandex-map.tsx`
```tsx
'use client'
export function YandexMap(props: {
  apiKey: string
  center: readonly [number, number] // [lon, lat]
  zoom: number
  label: string          // aria-label of the region, e.g. "Карта: ул. Гоголя, 61"
  className?: string
}): JSX.Element
```
- Renders a `div` (`role="region"`, `aria-label={label}`) that fills its parent. Loads the API only when the div comes
  near the viewport (IntersectionObserver, `rootMargin: '200px'`), because both maps can be below the fold.
- Creates `YMap` with `YMapDefaultSchemeLayer({ theme })` + `YMapDefaultFeaturesLayer` and one `YMapMarker` at
  `center`. `theme` = `resolvedTheme` from `next-themes` (`'dark'` or `'light'`); on theme change call
  `layer.update({ theme })`, do not recreate the map.
- `behaviors: ['drag', 'pinchZoom', 'dblClick']` — **no `scrollZoom`**, the page must scroll over the map.
- Marker: a 44 px round element in `bg-accent text-on-accent` with the `PinIcon` path (same look as the contacts
  placeholder pin), `aria-hidden`. Tokens only, no hex.
- `prefers-reduced-motion`: no `location` animation (`duration: 0`).
- Destroy the map (`map.destroy()`) on unmount.
- While loading and on load error: keep showing the children / placeholder look (bg-inverse-surface or
  bg-placeholder from the caller), never a blank box. Log the error in English (`console.error`).

### Pages
- Home `ContactsBlock`: if `serverEnv().YANDEX_MAPS_API_KEY` is set, the map box (`hidden md:flex`, `min-h-[340px]`,
  `rounded-lg`, `overflow-hidden`) holds `<YandexMap … className='absolute inset-0'/>`; otherwise the current
  placeholder. Mobile still has no map (follow the reference).
- `/contacts`: the map goes **under** the address overlay card (overlay stays on top, `relative z-10`), the card keeps
  `260px` mobile / `min-h-[480px]` desktop. The «Маршрут» button stays: it is the keyboard/screen-reader way to the map.
- `label`: `Карта: ${address.street}`.

## Done when
```
npm run check
npm run build
YANDEX_MAPS_API_KEY=<key> npm start -- -p 3111   # owner's key from .env
npm run qa:a11y
npm run qa:responsive
node scripts/qa/flows/home.mjs
node scripts/qa/flows/checkout-and-contacts.mjs
```
- Screenshots of `/` at 1440 px and `/contacts` at 390 and 1440 px, both themes: the pin is on ул. Гоголя, 61,
  the map is dark in the dark theme, the address card is above the map, mouse wheel over the map scrolls the page.
- Without the key (`YANDEX_MAPS_API_KEY=` empty) both pages show the old placeholders and the console has no errors.
- If the key is missing in `.env` or the owner has not sent the coordinates (`<LON>, <LAT>` are still in this file), stop and write that in
  `PROGRESS.md`.

