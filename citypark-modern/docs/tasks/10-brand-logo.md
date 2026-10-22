# Task 10: Brand logo in the header and footer

## Goal
The site shows the real «Сити Парк» logo (maple leaf + «Сити Парк» in a serif face) instead of the typed wordmark
«СИТИ ПАРК». The header, the mobile menu and the checkout header show a small logo. The footer shows it large, in
place of the giant Unbounded wordmark.

## Depends on
None.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/widgets/site-header/ui/site-header.tsx`, `src/widgets/site-header/ui/mobile-menu.tsx`,
`src/pages/checkout/ui/checkout-header.tsx`, `src/widgets/site-footer/ui/site-footer.tsx`, `src/shared/ui/index.ts`,
`src/shared/config/site.ts`, `src/app/styles/globals.css` (link hover rule, `@utility` blocks).

## Background
- The client gave two files, both in the folder above this repo (`/workspace/citypark/`):
  - `сити парк логотип кириллица.svg`: vector, **one colour** (`#1F1B20`), one `<path>`, exported from CorelDRAW on an
    A4 page (`viewBox="0 0 21000 29700"`), so the logo sits in a small part of a big empty page.
  - `логотип2_Монтажная область 1.png`: the same logo as a 3508 × 2481 raster, brown with a leaf pattern inside the
    letters.
- **Use only the SVG.** The site palette is neutral + lime (`--accent`), with a dark theme. A one-colour mark can
  take `--text` and switch with the theme; the brown PNG cannot, its colours are not tokens, and it is raster.
  Do not copy the PNG into the repo.
- The measured bounds of the logo in SVG units: x 2400, y 8062, width 16201, height 13585 (aspect ≈ 1.19 : 1).
- The logo appears up to 4 times on a page (header mobile + desktop, mobile menu, footer). Its path is ~13 KB, and
  an inline SVG would be repeated in the HTML and in the RSC payload. So the file is served once from `public/` and
  drawn with a CSS mask: `background-color: currentColor` + `mask-image: url(...)`. The colour then comes from
  `color`, so the logo follows the theme and the link hover opacity with no extra code.
- The type wordmark in the footer is the design's signature. The real logo replaces it; it becomes the one large
  element of the footer. Leave `--text-wordmark` in `tokens.css` as it is (the file is a copy of `design/tokens.css`).
- Favicon (`app/icon.svg`) stays as it is: the two-line logo is unreadable at 16–32 px.

## Files to create
- `public/brand/logo.svg`: the cleaned logo (Spec 1).
- `src/shared/ui/logo.tsx`: `Logo` (Spec 2).

## Files to change
- `src/shared/ui/index.ts`: export `Logo`.
- `src/shared/config/site.ts`: `wordmark: 'Сити Парк'` (it becomes the logo's accessible name; the text is how the
  logo spells it).
- `src/widgets/site-header/ui/site-header.tsx`, `src/widgets/site-header/ui/mobile-menu.tsx`,
  `src/pages/checkout/ui/checkout-header.tsx`: Spec 3.
- `src/widgets/site-footer/ui/site-footer.tsx`: Spec 4.
- `scripts/qa/flows/home.mjs`: Spec 5.

## Spec

### 1. `public/brand/logo.svg`
Start from a copy of `/workspace/citypark/сити парк логотип кириллица.svg` and make exactly these changes:
- Remove the XML declaration, the `DOCTYPE`, the `<!-- Creator -->` comment, `<defs>`/`<style>`, `<metadata>`, the
  `<g>` wrapper, `xml:space`, `version`, `style`, `width`, `height` and the `xlink`/`xodm` namespaces.
- Root: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="2380 8040 16240 13630">` (the bounds above plus ~20 units
  of air on each side).
- The path keeps its `d` **character for character**; replace `class="fil0"` with `fill="#000"` and add
  `fill-rule="evenodd"` (the file set it in `style`; without it the leaf veins and letter counters fill in).
- Check: open the file in the browser at `/brand/logo.svg`. The logo fills the frame with a thin even margin, and the
  white veins of the leaf and the holes in «а», «р» are open.

### 2. `Logo`
```tsx
/** The «Сити Парк» logo, drawn in `currentColor`. Size it with a height class; the width follows the aspect ratio. */
export function Logo({ className }: { className?: string }): JSX.Element;
```
- Renders `<span role='img' aria-label={site.wordmark} className=...>`. No children, no `<img>`, no inline SVG.
- Base classes: `inline-block aspect-[1624/1363] bg-current` and the mask
  `[mask:url(/brand/logo.svg)_center/contain_no-repeat]` (Tailwind arbitrary property; if the build drops it, add a
  `@utility logo-mask` in `globals.css` with `mask` and `-webkit-mask` instead, next to the other `@utility` blocks).
- Windows High Contrast: add `forced-colors:bg-[CanvasText]`, otherwise the forced background hides the logo.
- `className` is merged with `cn()` from `@/shared/lib`. `Logo` sets no height of its own.
- `shared/ui` may import `@/shared/config` (same layer). If Steiger objects, take `label` as a prop instead and pass
  `site.wordmark` from the callers.

### 3. Header, mobile menu, checkout header
In each `AppLink href='/'` that prints `{site.wordmark}` today, replace the text with `<Logo />`. The link keeps
`min-h-11 inline-flex items-center`; drop `font-display`, `font-semibold` and the `text-*` size from it.

| Place | Logo height | Pill left padding |
|---|---|---|
| `SiteHeader` mobile / tablet | `h-10` (40 px) | `pl-[18px]` → `pl-3` |
| `SiteHeader` desktop | `h-11` (44 px) | `pl-6` → `pl-4` |
| `MobileMenu` dialog head | `h-10` | `pl-[18px]` → `pl-3` (must match the header so the logo does not jump when the menu opens) |
| `CheckoutHeader` desktop | `h-11` | unchanged (the logo is centred there) |

- The pill heights must not change: the logo is no taller than the 44 px buttons next to it. Measure the pill
  before and after at 390 px and 1440 px.
- The link's accessible name becomes «Сити Парк» (from the `role='img'` label). Do not add a separate `aria-label` to
  the link.
- The mobile checkout bar has no logo today; keep it that way.

### 4. Footer
Replace the `<p>` with the «СИТИ ПАРК» text by
`<Logo className='mb-4 h-[120px] md:mb-8 md:h-[clamp(180px,18vw,240px)]' />`, left aligned, colour `text-text`
(inherited). It is not a link (the page already has a home link in the header).
- At 1440 px the logo is 240 px tall (≈ 286 px wide); at 390 px it is 120 px tall. Nothing else in the footer moves.
- Check both themes: the logo is near-black on `--bg` in light and near-white in dark.

### 5. QA in `scripts/qa/flows/home.mjs`
Add a block at the end, in the file's style:
- `out("logo: header link name", …)` → accessible name of the first `header a[href="/"]` →
  `Сити Парк`.
- `out("logo: header / footer height", …)` → `getBoundingClientRect().height` of `header [role=img]` (the visible one)
  and `footer [role=img]` at 1440 px → `44 / 240`.
- `out("logo: mask loaded", …)` → `fetch('/brand/logo.svg')` status and `content-type` → `200 / image/svg+xml`.

## Tests to write
There is no unit-test runner. Verify by hand at 390 px, 1280 px and 1440 px, light and dark, and report what you saw:
- the logo is sharp (it is a vector) and not cut at any edge in all four places;
- the header pill and the menu dialog head have the same height as before the change;
- hover on the header logo dims it like other links;
- open the mobile menu: the logo stays in the same spot;
- emulate `forced-colors: active` in DevTools (Rendering tab): the logo is still visible.

## Done when
CONVENTIONS "Done means" passes, and with a production build on port 3111: `node scripts/qa/flows/home.mjs`,
`npm run qa:a11y` and `npm run qa:responsive` print no unexpected values.

## Out of scope
The favicon and Apple touch icon, Open Graph images, the brown PNG, any new colour tokens, changes to
`design/tokens.css` or `design/reference/`, and any change in `../citypark-go/`.
