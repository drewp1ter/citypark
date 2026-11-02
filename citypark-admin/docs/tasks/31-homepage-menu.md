# Task 31: Homepage menu editor

## Goal
Staff choose and order homepage categories and up to three dishes per category at «Контент → Главная».

## Depends on
admin30; backend92 deployed.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/routes/_app/content/home.tsx`, `src/features/page-content/ui/page-content.tsx`, `src/shared/api/client.ts`.

## Files to create
- `src/features/homepage-menu/api.ts`, `model/selection.ts`, `model/selection.test.ts`, `ui/category-editor.tsx`, `ui/homepage-menu-editor.tsx`
- `e2e/homepage-menu.spec.ts`

## Files to change
- Compose the editor in the home content route; regenerate API types.

## Spec
Use generated `/homepage-menu` types and TanStack Query. Query visible categories and all paginated products independently of other feature slices.
Add/remove categories; reorder with accessible arrows. Search dishes within each category; select at most three; reorder and remove with previews.
Retain stale IDs with Russian repair warnings. Show loading/retry, indexed backend field errors and toasts. Keep drafts after failures and background refetches.
Explicit save replaces the whole configuration. Preserve pending form state and prevent edits while saving. Warn before leaving dirty forms.
Empty configuration uses first four visible categories; empty dish lists use automatic suggestions. Provide reset to automatic defaults.

## Tests to write
Vitest: immutable moves and boundaries; category-filtered case-insensitive search.
Playwright: save/reload, ordering, maximum, search, stale IDs, loading/retry, field and transport errors, dirty navigation/refetch, pending states, responsive layout in both themes.

## Done when
Conventions checks/build and the named Playwright flow pass against the configured backend. Restore temporary homepage configuration.

## Out of scope
Catalog sync, orders and payments.
