# City Park «Modern» — build plan (Feature-Sliced Design)

## Context
The repo has only a finished design (`design/` + `public/images/`): 8 pages × light/dark × desktop/mobile,
tokens in `design/tokens.css`, inventories in `pages.md` / `components.md`. Goal: a Next.js app in this
folder that matches the references and follows the stack and quality rules in `CLAUDE.md`.
**The owner asked for Feature-Sliced Design.** This replaces the `src/components/…` + `src/content/…`
layout from CLAUDE.md. Everything else in CLAUDE.md stays the same (Next App Router, TS, Tailwind v4,
tokens, next-themes, Server Actions + zod, no styled UI kit (headless react-aria-components is allowed)).
Environment: Node 24.22, npm 11.19; latest Next 16.4.x, Tailwind 4.3.x. No Playwright browser installed yet.

## FSD × Next.js App Router: folder layout
Both Next.js and FSD use the names `app` and `pages`. We follow the FSD guide for Next.js (changed during step 2:
the first idea, a `views` layer, was dropped because Steiger does not know that name and skipped the layer):
the Next router is the **root `/app`** (thin re-exports only), a stub **`/pages/README.md`** stops Next from using
`src/pages` as the Pages Router, and all FSD layers live in `src/`.

```
app/                        # Next router (root): layout.tsx, (site)/layout.tsx, (site)/page.tsx, restaurant/page.tsx, …
                            #   each route: `export { HomePage as default } from '@/pages/home'`
pages/README.md             # stub, see above
src/
  app/                      # FSD app layer: styles/{globals,tokens}.css, theme/ (next-themes), layouts/, cart provider later
  pages/                    # FSD pages layer: home, restaurant, rooms, hotel, menu, checkout, news,
                            #   news-article, contacts, legal
  widgets/                  # site-header (incl. mobile menu), site-footer, contacts-block, mobile-quick-actions,
                            #   home-hero, rooms-section, restaurant-section, delivery-section, news-section,
                            #   cart-panel, cart-bar, rooms-catalog, menu-catalog, hotel-rules, chef-blog …
  features/                 # theme-toggle, add-to-cart (+ / stepper / size switch), filter-menu
                            #   (category chips + search), search-rooms (StayDatesBar), filter-rooms,
                            #   book-table (form + action), place-order (checkout form + action),
                            #   contact-us (form + action)
  entities/                 # room, dish, article (news + chef blog), cart (model: context/reducer/localStorage), hotel
                            #   (rules, amenities, documents). Each: model/ (types + data), ui/ (cards), api/ (getters)
  shared/
    ui/                     # AppLink, Button, IconButton, icons; later Chip, Badge, Card, StatTile, Section,
                            #   Breadcrumbs, Field/Input/Select/Textarea/DateInput/PhoneInput/RadioCards/Checkbox
    config/site.ts          # nav, phones (+tel: hrefs), address, e-mail, requisites, chef
    lib/                    # cn, nb / formatPrice (NBSP), phone mask later
```
Rules: imports go only downward (app → pages → widgets → features → entities → shared), and
there are no cross-imports between slices of the same layer. Every slice and every `shared` segment exposes a public
API through `index.ts`. `"use server"` actions live in a slice's `api/` segment. Barrels may re-export client
components, but server-only code (actions) is imported from `@/features/x/api` so that it never ends up in a client bundle.
Content data (verbatim copy from the references) lives in `entities/*/model/*.data.ts`, plus `shared/config/site.ts`.
This replaces `src/content/`. Later a CMS can replace only these data files.
Lint: **Steiger** (`steiger` + `@feature-sliced/steiger-plugin`, `steiger.config.ts`) as `npm run lint:fsd`,
next to `npm run lint`. Steiger runs with no overrides. Single-use slices were merged into their only consumer (contact and order forms into their pages, the theme toggle into the header, the home contacts block into the home page).

## Backend: seams now, integrations later
There is no real backend in this phase, but the code is shaped so that adding one later changes only the `api/` segments:
- **Reads:** every entity exposes async getters in `entities/*/api` (`getRooms()`, `getMenu()`,
  `getNews()`, `getRoomAvailability(dates)`). Today they return the static `*.data.ts`; later they `fetch`
  from a CMS or booking engine. Pages and widgets only call the getters.
- **Writes:** Server Actions → zod → one port, `shared/api/leads.ts` (`submitLead(kind, payload)`). For now it
  only logs. Later it sends to Telegram, e-mail or a CRM, switched by env. Each action has one place for its `TODO`.
- **Config:** `shared/config/env.ts` reads and validates env vars with zod. `.env.example` has empty keys.
- Owner's answer: decide later. The restaurant uses **a POS (system not named yet)**; the rest is unknown. So the `dish` model
  is shaped to fit a POS: `id` (our slug) + optional `externalId`, `available` (stop-list), size variants
  as `variants[{id, label, weight, price}]` (POS modifiers), prices as integers in rubles. The cart stores
  `dishId + variantId` only and reads prices from `getMenu()`, so prices are never trusted from the client.
- The UI already has states that need a backend (room availability, "Только свободные", СБП online
  payment). They work against the stub getters; real availability and payment are a later step.

## Observations from the design (beyond the .md files)
- Mobile has a sticky bottom bar `<nav aria-label="Быстрые действия">` (Позвонить / Забронировать) with a
  fade `linear-gradient(transparent, var(--bg) 35%)` → widget `mobile-quick-actions`.
- Mobile home differs: 2×2 hero tiles, horizontal-scroll room and news rails, dishes as list rows, only 2 stats,
  short footer (`СИТИ<br>ПАРК`, 54px). The CSS breakpoints (`md`/`lg`) choose the layout; no JS.
- The `prefers-color-scheme` block in `tokens.css` overrides only some variables. This is harmless with next-themes
  (it always writes a resolved `data-theme`).

## Step 0 — Save this plan in the repo
Copy this plan to `docs/PLAN.md` in the project, so it is versioned next to the code and readable in later sessions.

## Step 1 — Scaffold (start here)
1. Run `create-next-app@latest` in a temp dir (this folder is not empty) and move the result in:
   TS, App Router, `src/`, Tailwind v4, ESLint, npm, alias `@/*`. Keep `public/images`.
2. Create the FSD skeleton above (empty layers with `index.ts` where a slice is created). Install `steiger`.
3. `src/app/styles/tokens.css` = copy of `design/tokens.css`. `globals.css`: `@import "tailwindcss"`,
   tokens, `@custom-variant dark (&:where([data-theme=dark], [data-theme=dark] *))`, and `@theme inline`
   mapping colors / radii / fonts / container. Breakpoints: `md` = 768px, `lg` = 1024px. Base styles:
   body `bg-bg text-text font-body`, `a:hover{opacity:.75}`, a `:focus-visible` 2px accent outline with offset, and a reduced-motion reset.
4. `layout.tsx`: `<html lang="ru" suppressHydrationWarning>`, Unbounded (400/500/600) + Onest (400–700)
   with `latin`+`cyrillic` via `next/font/google` → `--font-display`/`--font-body`. Wrap the app in
   `app/theme` (next-themes: `attribute="data-theme"`, `defaultTheme="system"`, `enableSystem`).
5. `ThemeToggle` (now in `widgets/site-header/ui`, moved there when Steiger showed it had a single consumer): client component, a 44px circle with a sun/moon icon and `aria-label="Переключить тему"`.
6. Update `CLAUDE.md` (Stack + quality rule 7) so it describes the FSD layout instead of `src/components` and `src/content`.
7. Temporary home page with the token swatches and the toggle. `npm run lint`, `lint:fsd` and `build` pass.

## Step 2 — shared/ui + layout widgets ✅ done (Field inputs, Chip, Card, Badge, StatTile, Section, Breadcrumbs are built together with the first page that uses them)
Primitives → site-header (sticky pill; ThemeToggle before the phone button; mobile: logo / cart / menu), mobile menu (inside site-header; a native modal `<dialog>`: links, phones, ThemeToggle, closes on Esc, focus trap), mobile-quick-actions, site-footer, contacts-block.

## Step 3 — entities ✅ done (room, dish, article, hotel, cart; the news entity became `article`)
room, dish (with size variants), news, hotel data and their cards. cart model: React context + `useReducer`,
persisted to `localStorage` (key `citypark-cart`, read after mount), lines keyed by `dishId:size`.

## Step 4 — pages (all done: home, menu, rooms, hotel, restaurant, news list, checkout, contacts, news article, 404), in order: home → rooms → hotel → restaurant → menu (+ cart) → checkout → news → contacts
Pages and widgets are Server Components. Client code only in features plus cart-panel and cart-bar.
Date defaults are today / tomorrow, computed on the client.
**After each page:** Playwright screenshots at 390 and 1440 px, light and dark. Compare them with
`design/screenshots/…`, fix the differences, then move on.

## Step 5 — Forms & states ✅ done for the forms that have a page (book-table, place-order, contact-us); the pages rooms, hotel, restaurant, menu, news list are still step 4
zod schemas + Server Actions in `features/{book-table,search-rooms,place-order,contact-us}/api`. Each action
validates, logs the payload and returns `{ok}|{errors}`, with `// TODO: send to e-mail / Telegram / CRM`.
`useActionState` for the pending / success / error states. Also: empty cart, news article, 404, legal placeholders.

## Learning-mode contribution points (you write 5–10 lines each)
- Cart reducer `addItem` / `changeQty` in `entities/cart/model` (size variants, what qty 0 does).
- Phone mask `formatPhoneInput()` in `shared/lib` for `+7 ___ ___-__-__`.

## Verification
- `scripts/screenshots.mjs` (`SEED_CART=1` puts the design's cart into localStorage) (Playwright + `npx playwright install chromium`): for every route ×
  {390, 1440} × {light, dark}, set `localStorage.theme` and take a full-page screenshot into the scratchpad.
  Compare each one with the design JPG.
- `npm run lint`, `npm run lint:fsd` (no layer or cross-import violations), `npm run build`.
- No hex colours in slices: `grep -rnE '#[0-9A-Fa-f]{3,6}\b' src --include=*.tsx` must be empty.
- Keyboard check (focus ring, mobile-nav), one `h1` per page.

## QA results (final round)
- **axe-core**: 0 violations in 48 page states (12 routes x light/dark x mobile/desktop), contrast included. It found 4
  issues on the way, all fixed (a photo rail without keyboard access, a button name that did not contain its visible
  text, a bar outside any landmark, a redundant alt text).
- **Responsive**: 11 routes x 12 widths (360-1920 px): no horizontal overflow, no JavaScript errors.
- **Keyboard**: skip link, a visible focus ring on every Tab stop of 9 pages x 2 sizes, mobile menu traps focus and
  returns it on Esc. Found and fixed: the focus ring of the calendar button inside date inputs.
- **Lighthouse**: desktop 98-100, accessibility / best practices / SEO 100 (checkout SEO 63 on purpose: it is
  disallowed in robots.txt), layout shift 0 on every page. Mobile (simulated slow 4G, 4x CPU) 80-96, median about
  84-90; one run varies by +-8 points on this machine, so compare medians. Tried and dropped, because the medians
  did not improve: `images.formats` AVIF and `experimental.inlineCss`. Kept: `fetchPriority="high"` + eager loading on
  the images that are the LCP element (`priority` is deprecated in Next 16).
- Added during QA: favicon (`app/icon.svg`), `robots.txt` (blocks `/checkout`), a "skip to content" link.

## Status
Steps 1–5 and all pages are done. Open items: real data and integrations (booking engine, POS, 1 кг pie price, delivery
tariffs, PDF documents, VK/OK links, Yandex Map, news texts and photos), the two functions left for the owner to write
(`normalizeQuantity`, `formatPhoneInput`), and a final QA round.
