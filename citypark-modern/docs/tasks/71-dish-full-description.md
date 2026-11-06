# Task 71: Full dish descriptions

## Goal
Implement the owner's requested full dish description across backend, admin and dish page.

## Depends on
Existing catalog, rich text editor/sanitizer and shareable dish pages.

## Read first
`AGENTS.md`, `docs/tasks/CONVENTIONS.md`; catalog/menu/dish implementations.

## Spec
Separate admin-managed `fullDescription` HTML from iiko's ordinary description. Reuse the short rich-text
profile, limit to 10000 visible characters, normalize empty markup, preserve through sync.
Admin-only PUT `/products/{id}/full-description`, required string, returns updated Product.
Admin editor supports save/reload/clear and preserves drafts on errors/refetch. Selection lives in the URL.
Only the dish page renders full HTML; empty values fall back to the ordinary plain-text description.
Cards and metadata retain the ordinary description.

## Tests to write
Backend permissions/validation/sanitization/persistence/clear/deleted product and sync preservation.
Admin selection parsing and browser editor save/reload/clear/error behavior; public dish rendering/fallback.

## API documentation
Update API.md and OpenAPI and regenerate admin types.

## Done when
Project convention checks/build/tests pass, with real-backend browser acceptance.

## Out of scope
Deployment, payment/order changes and editing the iiko description.
