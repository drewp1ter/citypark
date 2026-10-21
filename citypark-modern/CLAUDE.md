# City Park (Сити Парк) — website, "Modern" design

Website for **«Сити Парк»** in Belorechensk (Белореченск), Russia: a hotel (12 rooms), a restaurant with a
summer café, banquets, and food delivery. The site is in **Russian**. This repo starts with a finished
design (light + dark themes, desktop + mobile). Your job is to build it as a Next.js app that matches the
design closely.

## What is in this folder

```
CLAUDE.md                 ← this file
design/
  tokens.css              ← colours, fonts, radii, spacing as CSS variables (light + dark). Source of truth.
  pages.md                ← routes, sections of each page, what is not designed
  components.md           ← component inventory with notes
  reference/{light|dark}/{desktop|mobile}/<page>.html   ← static HTML of every screen (open in a browser)
  screenshots/{light|dark}/{desktop|mobile}/<page>.jpg  ← full-page screenshots of the same screens
public/images/            ← photos used by the design (+ placeholder.svg)
```

Pages: `home, restaurant, rooms, hotel, menu, checkout, news, contacts` (8 pages × 2 themes × 2 sizes = 32 screens).

**How to use the references:** the HTML files use inline styles with raw hex colours and pixel values.
Read them for exact layout, spacing, font sizes and Russian copy; look at the screenshots for the overall
picture. When you build components, convert every colour to the matching token from `design/tokens.css`
(the mapping table is at the top of that file). Light and dark references have identical markup — only
colours differ — so build each component once and let the tokens switch the theme.

> Screenshots were rendered **without the web fonts** (fallback sans-serif). The real fonts are
> **Unbounded** (display) and **Onest** (body). Trust the font-family/size values in the HTML, not the
> letter shapes in the screenshots.

## Stack (decisions — keep unless the owner asks otherwise)

- **Next.js, App Router, TypeScript**, current stable version. `src/` directory. Package manager: npm.
- **Architecture: Feature-Sliced Design** (owner's decision) — see "Code structure (FSD)" below and `docs/PLAN.md`.
- **Tailwind CSS v4.** Import `design/tokens.css` (copy it to `src/app/styles/tokens.css`) and expose the
  variables through `@theme` (e.g. `--color-bg: var(--bg)`) so classes like `bg-bg text-text border-border`
  work. Do not add a second colour palette.
- **Fonts:** `next/font/google` — `Unbounded` (weights 400/500/600) and `Onest` (400/500/600/700) with
  `subsets: ['latin', 'cyrillic']`; map them to `--font-display` / `--font-body`.
- **Themes:** `next-themes` with `attribute="data-theme"`, `defaultTheme="system"`, `enableSystem`.
  This avoids a flash of the wrong theme. Add a `ThemeToggle` button (sun/moon) in the header and in the
  mobile nav. It is the only new UI element; style it like the header's existing 44 px icon buttons.
- **Images:** `next/image` from `public/images`. Keep `alt` texts from the references.
- **Content:** typed data files inside FSD slices: `shared/config/site.ts` and
  `entities/{room,dish,article,hotel}/model/*.data.ts`, read only through async getters in each entity's `api/`
  (so a CMS / booking engine / POS can replace them later). Copy the texts, prices, weights and phone numbers
  **exactly** from the reference HTML. No CMS for now.
- **Cart:** client state (React context or Zustand) persisted to `localStorage`. Used by DishCard,
  CartPanel (desktop `/menu`), CartBar (mobile), header cart button and `/checkout`.
- **Forms** (table booking, checkout, contact form, room search): Server Actions with zod validation.
  There is no backend yet, so the actions validate, log the payload and return a success state.
  Leave a clear `TODO: send to …` where the real integration (e-mail / Telegram / CRM) goes.
- No UI kit (no shadcn, MUI, etc.). Components are small and custom, built from the tokens.

## Routes

`/` home · `/restaurant` · `/rooms` · `/hotel` · `/menu` (delivery) · `/checkout` · `/news` · `/contacts`.
Details per page are in `design/pages.md`.

## Responsive rules

- Mobile design is 390 px wide; desktop design is 1440 px with content max-width 1240 px.
- Use the mobile layout below **768 px** and the desktop layout from **1024 px**. Between them, use the
  desktop structure with fewer columns (2 instead of 3–4) — keep it simple and readable.
- Exception: the site header keeps its compact form (logo + buttons + menu sheet) up to **1279 px**; the
  full navigation needs about 1090 px plus the page gutters, so it starts at 1280 px.
- Mobile and desktop references sometimes differ in content (fewer sections on mobile). Follow each
  reference for its size; don't invent sections.

## Site facts (use these, don't invent others)

- Address: 352630, г. Белореченск, ул. Гоголя, 61
- Phones: restaurant & delivery **8 918 311-97-91** · hotel **8 918 311-97-10** · free in Russia **8 800 100-24-41** · landline (86155) 2-25-38
- E-mail: belcitypark@mail.ru
- Company: ООО «Олимп», ИНН 2309117400, ОГРН 1092309002378
- Chef: Денис Троицкий
- Phone links as `tel:+79183119791` etc.

## Quality rules

1. **Match the design.** Compare every page with its screenshots in both themes at 390 px and 1440 px.
2. **Tokens only.** No hex colours in components. If you need a colour that has no token, stop and add a
   token to `tokens.css` for both themes.
3. **Both themes always.** Every component must look right in light and dark. Text on lime (`--accent`)
   always uses `--on-accent`.
4. **Russian copy verbatim** from the references (including «ёлочки», `₽`, non-breaking spaces in prices
   like `4 000 ₽` and in phone numbers). `<html lang="ru">`.
5. **Accessibility:** semantic landmarks (header/nav/main/footer), one `h1` per page, labels on every
   input, `aria-label` on icon buttons (copy them from the references), touch targets ≥ 44 px, visible
   focus ring (2 px outline in `--accent` with offset), `prefers-reduced-motion` respected.
6. **Performance:** pages are Server Components; only interactive parts are client components. Use
   `next/image` with proper `sizes`. No layout shift from fonts or images.
7. Keep components small; one component per file, inside its FSD slice (`ui/` segment).

## Code structure (FSD)

Layers, top → bottom: `app` → `pages` → `widgets` → `features` → `entities` → `shared`, all inside `src/`.
A layer imports only from layers below it; slices of the same layer never import each other. Each slice has a
public API (`index.ts`) and imports go through it (`@/shared/ui`, `@/widgets/site-header`, …).

Next.js and FSD both want the names `app` and `pages`. We follow the FSD guide for Next.js: the Next router
lives in the **root `/app`** folder and only re-exports from `src` (`export { HomePage as default } from
"@/pages/home"`; layouts come from `@/app/layouts`), and the root **`/pages/README.md`** stub stops Next from
treating `src/pages` as the Pages Router. Do not add route files to `/pages`. Providers, global styles and
layouts live in the FSD app layer, `src/app`. Server Actions live in a slice's `api/` segment and are imported
from `@/features/<slice>/api`, not from the barrel. Check with `npm run lint:fsd` (Steiger).

## Suggested build order

1. Scaffold Next.js + Tailwind v4, copy tokens, set up fonts and `next-themes`, add ThemeToggle.
2. Primitives (Button, IconButton, Chip, Card, Field inputs) → `SiteHeader`, `MobileNav`, `SiteFooter`, `ContactsBlock`.
3. Entities with their data and getters (see Content above).
4. Pages in this order: home → rooms → hotel → restaurant → menu (+ cart) → checkout → news → contacts.
5. Forms and server actions; empty/success/error states.
6. QA: run the app, take Playwright screenshots at 390 and 1440 in both themes, compare with
   `design/screenshots/`, fix differences. Run `npm run lint` and `npm run build` with no errors.

## Forms (how they work)

Every form is a Server Action in the `api/` segment next to its UI, with a zod schema in `model/`. The table-booking
form is a feature (`features/book-table`, used by home and restaurant); the checkout and contact forms live in their
pages (`pages/checkout`, `pages/contacts`) because only one page uses each. The action validates, then calls
`submitLead(kind, payload)` from `shared/api` (the only place that will talk to e-mail / Telegram / CRM; for now it
logs). It returns `{ status: "error", errors, values }` so the form can show field errors and keep what was typed, or
`{ status: "success" }`. The order action re-prices the cart from the menu on the server and redirects to
`/checkout/done`, where the cart is cleared. Rules that depend on other fields (address for delivery, time for
"Ко времени…") are separate functions, not zod refinements, because zod skips object-level refinements while any
field is invalid.

**Start small:** UI used by one page stays in that page (`pages/<name>/ui`); move it to a widget or feature only when a
second page needs it. Steiger (`npm run lint:fsd`) enforces this, so keep it free of overrides.

## Known gaps

- Three news photos are missing (`/public/images/placeholder.svg` is used). The owner will send them.
- Map is a placeholder; later embed Yandex Maps (address above).
- Not designed: mobile nav sheet, news article page, 404, empty cart, form success/error states — build
  them from the existing components and tokens (see `design/pages.md`).
- Date inputs in the screenshots show sample dates; use real defaults (today / tomorrow).
- Legal pages (Документы, Оферта, Конфиденциальность) have no content yet — create routes with a
  placeholder heading.
