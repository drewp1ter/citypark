# Task 36: Admin PWA

## Read first

- AGENTS.md and docs/tasks/CONVENTIONS.md
- src/main.tsx and src/shared/ui/app-shell.tsx
- nginx/default.conf.template

## Deliver

Installable Russian admin for mobile and desktop, brand icons, standalone launch,
installation control/instructions and a production-only worker caching only an offline page.
Keep sessionStorage authentication. Do not cache API data or queue writes.
Preserve active forms during worker updates. Serve PWA assets with correct types and revalidation.

## Done when

- npm run check
- npm run build
- npx playwright test --config playwright.pwa.config.ts
- npm run e2e -- e2e/auth.spec.ts
- Record physical-device installation verification separately (Android Chrome, iOS/iPadOS Safari, desktop Chromium).
