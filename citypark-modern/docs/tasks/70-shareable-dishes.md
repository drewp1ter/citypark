# Task 70: Shareable dish pages

## Goal
Public dish pages with stable links and social previews.

## Depends on
68.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/widgets/food-ordering/ui/dish-card.tsx`
- `src/shared/config/social.ts`
- `src/entities/dish/api/get-menu.ts`

## Files to create
- `app/(menu)/menu/[id]/page.tsx`
- `src/pages/dish/index.ts`
- `src/pages/dish/ui/dish-page.tsx`
- `src/pages/dish/ui/dish-controls.tsx`
- `scripts/qa/dish-pages.mjs`

## Files to change
- Public menu cards: link photos and names, preserving Mini navigation.
- Sitemap: include public dish IDs.

## Spec
Serve `/menu/[id]` from the public server menu. Show name, image, composition, selected portion weight/price and existing cart controls. Unknown/hidden dishes return 404; backend failures reach the existing menu error boundary. Keep unavailable dishes visible with ordering restrictions. Default to the first portion. Include category breadcrumb, production canonical URL and existing Open Graph/Twitter image policy. Copy button says «Копировать ссылку», success «Ссылка скопирована»; failure exposes a selectable URL. Use tokens, Russian UI and responsive layout.

## Tests to write
Read-only production browser/HTTP checks for dish metadata, sitemap inclusion, unknown ID, responsive themes, clipboard success/failure and menu navigation. Existing cart controls retain existing behavior; inspect portion/availability branches where live fixtures are absent.

## API documentation
No backend API changes.

## Done when
`npm run check`, `npm run build`, `node scripts/qa/dish-pages.mjs` pass. External social previews require deployment.

## Out of scope
Backend/admin changes, Mini navigation changes, generated social images.
