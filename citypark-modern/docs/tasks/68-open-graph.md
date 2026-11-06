# Task 68: Page-specific Open Graph previews

## Goal
Public links expose Russian page-specific Open Graph and Twitter previews.

## Depends on
61.

## Read first
- docs/tasks/CONVENTIONS.md
- app/layout.tsx
- src/shared/config/search.ts

## Files to create
- src/shared/config/social.ts
- scripts/qa/open-graph.mjs

## Files to change
Public route metadata, room/article metadata generators and shared config exports.

## Spec
Provide complete Open Graph and summary_large_image Twitter metadata using the canonical production origin. Use existing page copy and relevant photos; HTTP/SVG/missing photos fall back to hotel-building.jpg. Room and news details use their own content. News uses article type. Preserve indexing restrictions and existing failure behavior.

## Tests to write
Production crawler checks: static and detail titles, descriptions, canonical URLs, locale, cards and accessible images.

## Done when
npm run check; npm run build; BASE_URL=http://localhost:3111 node scripts/qa/open-graph.mjs.

## Out of scope
Deployment, external social cache refresh, custom cards, backend writes.
