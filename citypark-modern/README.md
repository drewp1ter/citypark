# Сити Парк — website ("Modern" design)

Website for the hotel, restaurant and delivery «Сити Парк» in Белореченск. Next.js (App Router) + TypeScript +
Tailwind CSS v4, built from the designs in `design/`. The site is in Russian.

## Run

```bash
npm install
cp .env.example .env.local   # MENU_API_URL: the citypark-go API (the menu is read from it)
npm run dev          # http://localhost:3000 (the backend uses :3000 too: npm run dev -- -p 3001)
npm run build && npm start
npm run check        # typecheck + eslint + Steiger (FSD rules): must be clean
```

## How it is organised

[Feature-Sliced Design](https://feature-sliced.design). Layers inside `src/`, top to bottom:
`app → pages → widgets → features → entities → shared`; a layer only imports from layers below it.
The Next.js router is the root `app/` folder (thin re-exports); the stub `pages/README.md` stops Next from reading
`src/pages` as a Pages Router. See `CLAUDE.md` for the rules and `docs/PLAN.md` for the plan and the current status.

- `design/tokens.css` is the source of truth for colours, radii, fonts. `src/app/styles/tokens.css` is its copy
  (plus a few tokens added for errors and wider layouts); components use only token classes (`bg-surface`, …).
- Content is typed data next to its entity (`src/entities/*/model/*.data.ts`) and read through async getters in
  `api/`, so a CMS, booking engine or POS can replace it without touching the pages.
- Forms are Server Actions with zod; they all send through `src/shared/api/leads.ts`, which only logs for now.

## Quality checks (the server must be running, e.g. `npm run build && npm start`)

| Command | What it does |
|---|---|
| `npm run qa:a11y` | axe-core on every page, light + dark, mobile + desktop (WCAG 2.2 AA, contrast included) |
| `npm run qa:responsive` | every page at 12 widths from 360 to 1920 px: horizontal overflow, JavaScript errors |
| `npm run qa:keyboard` | skip link, visible focus ring on every Tab stop, mobile menu focus trap |
| `npm run qa:flows` | the user flows in a browser: cart, search, filters, booking, checkout, contact form, 404 |
| `node scripts/screenshots.mjs <dir> [routes]` | full-page screenshots at 390 / 1440 px, light + dark (`SEED_CART=1` fills the cart) |
| `node scripts/qa/lighthouse.mjs <url> <name> [mobile\|desktop]` | Lighthouse (needs `npm i --no-save lighthouse`) |

`BASE_URL` changes the address (default `http://localhost:3111`). Playwright needs its browser once:
`npx playwright install chromium`.

## Still needs real data from the owner

Booking engine and room availability · Menu content in the backend (descriptions, weights and photos of the dishes) · delivery tariffs below 500 ₽ and for СОТ/ДНТ · PDF documents and legal texts · text and photos of the
news and recipes · the Yandex Map · a 3D panorama link. The integration points are marked with `TODO` in the code.
