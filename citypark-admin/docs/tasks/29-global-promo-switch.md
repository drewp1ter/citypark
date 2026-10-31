# Task 29: Global promo-code switch in settings

## Goal
Implement the global promo-code controls and checkout behavior described below, preserving existing orders.

## Depends on
Backend task 90 and admin task 28.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/features/settings/ui/settings-page.tsx
- `src/features/settings/model/form.ts`
- `src/features/settings/api.ts``

## Files to create
None expected; extend existing implementation and tests.

## Files to change
The implementation files listed above, their existing tests, and the supporting files explicitly named in the spec.

## Spec
- Regenerate src/shared/api/schema.d.ts with npm run api:gen after backend 90.
- Add a HeroUI v3 switch labeled `Промокоды включены` in an ordering settings section. Read the installed component documentation first.
- Include promoCodesEnabled in toForm and request mapping; keep it boolean. Preserve existing save, dirty-state, navigation protection and error behavior. Correct the comment claiming all form values are strings.
- Toggling only changes the local form until Save succeeds. Do not modify individual campaigns or hide their management page.

## Tests to write
- Extend model/form.test.ts for true/false mapping and request serialization.
- Extend e2e/settings.spec.ts for loading false, enabling/saving/reloading, disabling/saving/reloading, unsaved-change protection and failed-save value preservation. Restore the original setting after testing.

## API documentation
Consume backend task 90; do not change backend files.

## Done when
```bash
npm run api:gen
npm run check
npm run build
npm run e2e -- e2e/settings.spec.ts
```
All checks required by CONVENTIONS pass. Backend changed-package statement coverage is at least 80% where applicable. Use configured services; do not start the backend/database. Live QA writes require existing session authorization; otherwise document the exact pending checks.

## Out of scope
Other projects, unrelated task completion, individual campaign redesign, retroactive repricing, real payments and iiko dispatch. On completion, tick this task and commit according to the project workflow.
