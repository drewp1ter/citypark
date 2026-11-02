# Task 92: Homepage menu configuration

## Goal
Persist ordered homepage categories and up to three ordered dishes per category independently of iiko sync.

## Depends on
91.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/deliverysettings/handler.go`, `internal/store/store.go`, `internal/catalog/store.go`.

## Files to create
- `migrations/00032_homepage_menu.sql`
- `internal/homepagemenu/{store,handler,homepagemenu_test}.go`

## Files to change
- Wire routes in `internal/app/app.go`; update OpenAPI and API.md.

## Spec
Public GET and admin PUT `/homepage-menu` return `{categories:[{categoryId,productIds:[]}]}`.
Arrays define order; empty categories means default first four visible menu categories; empty products means automatic suggestions.
Require non-null arrays, positive unique category IDs, and at most three positive unique product IDs belonging to each category.
Reject hidden/deleted/missing categories and deleted/missing/moved products; allow ordering blocks and closed hours.
Return indexed field errors with the existing validation message. Save atomically with catalog row locks.
Keep raw IDs after catalog changes so the admin can repair them; website resolves them against the current menu.
Storage is a singleton JSONB row with updated_at; sync does not write it.

## Tests to write
- `TestHomepageMenuPersistenceAndValidation`: public defaults, auth, ordering, empty/reset, membership, duplicates, limit, malformed JSON, rejected writes preserve data, catalog changes preserve raw IDs.
- `TestHomepageMenuMigration`: up/down/up in isolated schema restores defaults.
- `TestHomepageMenuErrors`: missing row and database failures.
- iiko regression: full sync preserves saved selections.

## API documentation
Document both methods, field errors and stale-ID behavior in API.md and OpenAPI.

## Done when
All convention checks pass, changed package coverage ≥80%, wiring auth checks pass, and sync preservation passes.

## Out of scope
Admin editor (admin31) and homepage consumer (website51).
