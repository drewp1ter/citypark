# Task 27: Promo campaign management

## Goal
Provide a Russian admin screen to create, edit and disable campaigns.

## Depends on
admin26; backend86

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](../../../citypark-go/docs/tasks/PROMOCODES.md)
- `src/features/menu/api.ts`
- `src/shared/api/client.ts`

## Files to create
- `src/routes/_app/promo-codes.tsx`
- `src/features/promo-codes/api.ts`
- `src/features/promo-codes/ui/promo-codes-page.tsx`
- `src/features/promo-codes/ui/promo-code-form.tsx`
- `src/features/promo-codes/model/form.ts`
- `src/features/promo-codes/model/form.test.ts`
- `e2e/promo-codes.spec.ts`

## Files to change
- `src/shared/api/schema.d.ts`
- `src/shared/ui/nav-items.ts`

## Spec
Provide a Russian admin screen to create, edit and disable campaigns.
Use generated backend types, HeroUI v3 and TanStack Query; add «Промокоды» to the existing navigation where
its items are currently defined. Read installed HeroUI docs. List code, percentage, enabled state and schedule.
Form offers «Все блюда» or «Выбранные блюда и категории» with searchable independent multi-selects.
Query menu endpoints inside this feature, not by importing the menu feature. Code is readonly after creation.
Fields: «Процент скидки», «Минимальная сумма подходящих блюд», «Максимальная скидка», «Начало»,
«Окончание», «Активен». Empty optional values map to null. Convert Moscow wall time explicitly regardless of
browser timezone. Explain category membership is dynamic and minimum counts only eligible dishes.
Preserve form values on errors, display API fields/toasts and invalidate campaign queries after saves.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- form.test.ts: nullable values, 1/99 bounds, mixed targeting, Moscow conversion from a non-Moscow browser timezone.
- e2e/promo-codes.spec.ts: create all/selected codes, edit active limits/schedules, disable, reject duplicates,
  failed-save preservation and responsive keyboard operation in both themes. Use unique fixture codes and disable them.
Run npm run e2e -- e2e/promo-codes.spec.ts.

## API documentation
No endpoint changes in this task; consume the published backend contract.

## Done when
Run npm run api:gen, npm run check, npm run build, and the named npm run e2e spec against the
configured existing backend. Never start the backend/database. Tick only this task after all checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
