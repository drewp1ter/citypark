# Task 34: Installable site with an offline page

## Goal
The site can be installed to the home screen (Android Chrome, iOS Safari, desktop Chrome) as «Сити Парк», opens in
standalone mode, and shows a Russian offline page when a page is opened without a network. Nothing else is cached:
the menu, prices, rooms and checkout always come from the network.

## Depends on
Nothing. No backend change.

## Read first
`docs/tasks/CONVENTIONS.md`, `node_modules/next/dist/docs/01-app/02-guides/progressive-web-apps.md` (sections 1, 5–8;
skip push notifications), `node_modules/next/dist/docs/01-app/03-api-reference/03-file-conventions/01-metadata/manifest.md`,
the `viewport` / `appleWebApp` parts of the `generateMetadata` / `generateViewport` reference in the same docs folder,
`app/layout.tsx`, `app/icon.svg`, `next.config.ts`, `src/app/styles/tokens.css` (`--bg`, `--text`, `--accent`,
`--on-accent` for both themes), `scripts/qa/flows/home.mjs` (style of QA flows).

## Background
- Install needs: HTTPS, a manifest with 192 and 512 PNG icons, `display: standalone`. iOS uses `apple-touch-icon`
  and `appleWebApp` metadata and ignores manifest icons.
- The service worker is hand-written (no `next-pwa`/Serwist: they are not maintained for Next 16 + Turbopack, and we
  cache almost nothing). It handles **only navigation requests**: network first; when the network fails, it returns
  the cached `/offline.html`. Other requests (`/_next/*`, images, Server Actions, `/api/*`) are not intercepted.
- `/offline.html` must work with no network, so it is one static file with inline CSS and no web fonts. The Next
  CSS chunks have hashed names and cannot be precached reliably.
- **Raw hex is allowed in exactly three places**, because browsers read them outside our CSS: `app/manifest.ts`,
  the `viewport.themeColor` in `app/layout.tsx`, and `public/offline.html`. Copy each value from `tokens.css` and put
  the token name in a comment next to it (`'#f4f4f1' // --bg (light)`).
- `themeColor` follows the system theme via `media`; a user who picked the other theme with ThemeToggle gets the
  system colour in the browser bar. Accept that.
- Register the worker only in production builds, so it never interferes with `next dev` HMR.

## Files to create
- `scripts/pwa-icons.mjs` — one-off generator: opens `app/icon.svg` in Playwright Chromium and screenshots it to PNG.
  Outputs (commit them):
  - `public/pwa/icon-192.png` (192×192), `public/pwa/icon-512.png` (512×512) — the SVG as is;
  - `public/pwa/icon-maskable-512.png` (512×512) — full-bleed `--accent` square with the «СП» glyph scaled to fit in
    the central 80 % safe zone (no rounded corners: the OS masks it);
  - `app/apple-icon.png` (180×180) — like the maskable one (iOS adds its own rounding). Next picks it up by file convention.
  Header comment: what it does and `Usage: node scripts/pwa-icons.mjs`.
- `app/manifest.ts` — `MetadataRoute.Manifest`:
  `name: 'Сити Парк — отель, ресторан и доставка'`, `short_name: 'Сити Парк'`, `description` = the root layout's
  description, `lang: 'ru'`, `start_url: '/'`, `scope: '/'`, `display: 'standalone'`,
  `background_color` and `theme_color` = light `--bg`, the three icons above (`purpose: 'any'` for the two plain ones,
  `'maskable'` for the maskable one), and `shortcuts`: «Меню и доставка» → `/menu`, «Номера» → `/rooms`.
- `public/sw.js` — spec below.
- `public/offline.html` — spec below.
- `src/app/pwa/register-service-worker.tsx` (`'use client'`) and `src/app/pwa/index.ts` exporting
  `RegisterServiceWorker`. Component renders `null`; in `useEffect`, if `process.env.NODE_ENV === 'production'` and
  `'serviceWorker' in navigator`, calls `navigator.serviceWorker.register('/sw.js', { scope: '/', updateViaCache: 'none' })`
  and logs a failure with `console.error` (English message).

## Files to change
- `app/layout.tsx`:
  - add `export const viewport: Viewport` with `themeColor: [{ media: '(prefers-color-scheme: light)', color: <light --bg> }, { media: '(prefers-color-scheme: dark)', color: <dark --bg> }]`;
  - add to `metadata`: `applicationName: 'Сити Парк'`, `appleWebApp: { title: 'Сити Парк', statusBarStyle: 'default' }`;
  - render `<RegisterServiceWorker />` inside `<body>` (import from `@/app/pwa`).
- `next.config.ts`: add `async headers()` for `source: '/sw.js'` with `Cache-Control: no-cache, no-store, must-revalidate`
  and `Content-Type: application/javascript; charset=utf-8` (as in the Next PWA guide, section 8). Keep the other
  security headers from the guide out of this task.

## Spec

`public/sw.js`:
```js
// Service worker: shows /offline.html when a page cannot be loaded. Caches nothing else on purpose
// (menu and prices must always be fresh). Bump VERSION when offline.html changes.
const VERSION = 'v1'
const CACHE = `citypark-offline-${VERSION}`
const OFFLINE_URL = '/offline.html'
```
- `install`: open `CACHE`, `add(new Request(OFFLINE_URL, { cache: 'reload' }))`, then `self.skipWaiting()`.
- `activate`: delete every cache whose name starts with `citypark-offline-` and is not `CACHE`; then `self.clients.claim()`.
- `fetch`: only when `event.request.mode === 'navigate'` → `event.respondWith(fetch(event.request).catch(() => caches.match(OFFLINE_URL)))`.
  Return without calling `respondWith` for everything else.
- No `push` / `notificationclick` handlers.

`public/offline.html` (`<html lang="ru">`, viewport meta, `<title>Нет соединения — Сити Парк</title>`):
- Centered card, system font stack, colours from `tokens.css` for light, and for dark under
  `@media (prefers-color-scheme: dark)`, each hex with its token name in a comment.
- `<main>` with `<h1>Нет соединения с интернетом</h1>`, `<p>Проверьте подключение и обновите страницу.</p>`,
  a button «Обновить» (`onclick="location.reload()"`, `--accent` background, `--on-accent` text, ≥ 44 px tall,
  visible focus ring: 2 px `--accent` outline with offset), and
  `<p>Заказать по телефону: <a href="tel:+79183119791">8&nbsp;918&nbsp;311-97-91</a></p>`
  (the default restaurant phone from `src/shared/config/site-settings.ts`; offline there are no backend settings).
- No external resources at all (no images, fonts or scripts besides the inline `onclick`).

## Out of scope
Push notifications, a custom install button, caching of pages/assets/menu, `experimental.useOffline`, the other
security headers from the PWA guide.

## Done when
```
node scripts/pwa-icons.mjs
npm run check
npm run build
npm start -- -p 3111
node scripts/qa/flows/pwa.mjs
npm run qa:flows
```
- Create `scripts/qa/flows/pwa.mjs` in the style of the other flows (`out(label, value)` with expected next to real):
  - `GET /manifest.webmanifest` → status 200, `short_name` «Сити Парк», `display` standalone, icons 192 and 512
    present and each icon URL answers 200 with `image/png`;
  - `/` has `<link rel="manifest">`, `<link rel="apple-touch-icon">`, two `<meta name="theme-color">`;
  - `GET /sw.js` → `Cache-Control` contains `no-cache`;
  - in Chromium: open `/`, wait for `navigator.serviceWorker.ready`, then `context.setOffline(true)`, go to `/menu`
    → the page `h1` is «Нет соединения с интернетом»; `setOffline(false)`, reload → the real `/menu` `h1`;
  - while online, a `/menu` load has no response served by the service worker other than the navigation
    (`response.fromServiceWorker()` false for `/_next/*` and images).
- Lighthouse (`scripts/qa/lighthouse.mjs`, if it reports PWA/installability) has no installability errors.
- Manual (owner, needs HTTPS — e.g. `new.cityparkvip.ru`): install on Android Chrome and iOS Safari («На экран
  „Домой“»); the icon is the lime «СП», the name «Сити Парк», it opens without the browser bar; in airplane mode
  a page shows the offline page; checkout still works online.
