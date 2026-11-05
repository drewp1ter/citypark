# Task 61: Indexing foundations

## Goal
Prepare production metadata and prevent staging/private-page indexing.

## Depends on
60.

## Read first
`docs/tasks/CONVENTIONS.md`, `app/layout.tsx`, `app/robots.ts`, `.env.example`.

## Files to create
This task document and `docs/SEARCH_DISCOVERY.md`.

## Files to change
Root metadata, existing public page metadata, room/article metadata, private layouts and robots configuration.

## Spec
Use https://cityparkvip.ru as the single metadata base. Public pages have self-canonical URLs without query strings and Russian titles/descriptions. Existing dynamic content titles remain. SITE_INDEXING_ENABLED defaults to false; set true only on production after launch. Account, checkout and Mini App layouts always emit noindex. Robots allows crawling so crawlers can read noindex. Sitemap is task 62.

## Tests to write
Configuration and metadata only: run static checks/build and inspect generated metadata configuration. Live HTTP acceptance is tracked in SEARCH_DISCOVERY.md.

## Done when
`npm run check` and `npm run build` pass.

## Out of scope
Sitemap, structured data, new customer content and Webmaster registration.
