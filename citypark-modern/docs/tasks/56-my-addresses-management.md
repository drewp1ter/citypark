# Task 56: My addresses management screens

## Goal
Guests and signed-in customers can add, name, edit, delete and select a default address; Telegram uses its own account session.

## Depends on
Website 55b and backend 98 (all backend compatibility/merge work). Do not modify backend files.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/entities/address/api/server.ts`
- `src/pages/account/ui/account-page.tsx`
- `src/pages/mini/ui/mini-account-page.tsx`

## Files to create
- `src/features/manage-addresses/index.ts`
- `src/features/manage-addresses/api/actions.ts`
- `src/features/manage-addresses/model/types.ts`
- `src/features/manage-addresses/ui/address-manager.tsx`
- `src/features/manage-addresses/ui/address-editor.tsx`
- `src/features/manage-addresses/ui/address-card.tsx`
- `src/pages/addresses/index.ts`
- `src/pages/addresses/ui/addresses-page.tsx`
- `src/pages/mini/ui/mini-addresses-page.tsx`
- `app/(plain)/account/addresses/page.tsx`, `app/(mini)/mini/addresses/page.tsx`
- `scripts/qa/address-management-harness.mjs`, `scripts/qa/flows/my-addresses.mjs`

## Files to change
- `src/pages/mini/index.ts` — page export.
- `src/pages/account/ui/account-page.tsx`, `src/pages/mini/ui/mini-account-page.tsx` — navigation.
- `src/widgets/site-header/ui/site-header.tsx`, `src/widgets/site-header/ui/mobile-menu.tsx` — guest-accessible link.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
Read installed Next.js Server Actions/cookies/revalidation docs. Route files only re-export pages.
`AddressesPage` calls loadAddressBook('web') and permits guest use without redirecting `/account/addresses`
to sign-in. `MiniAddressesPage` uses mini and existing relaunch behavior when unauthenticated.
An account API error is a retry/auth state, not a guest book. Fetch delivery towns and suggestion configuration
server-side; list management still renders saved entries if town lookup fails, with editor retry feedback.

Export AddressManager and a controlled AddressEditor through the feature barrel; public props carry
AddressContext, initial entries, town options and suggestionsEnabled. Each component stays in its own file.
AddressManager uses guest-store methods for true guests and Server Actions for account contexts.
Actions `saveAddressAction`, `deleteAddressAction`, `defaultAddressAction` validate surface, expectedAccountId,
entry ID/fields, use entity server helpers and return a typed success/error result; revalidate the current
surface's address-management/checkout paths. Pending UI prevents duplicate submissions. Preserve form
values on every error. Clear/reset account data on changed context; ignore late old-account responses.

Both routes have one h1 `Мои адреса`. Copy:
- Guest note: `Адреса сохраняются только в этом браузере`.
- Empty: `У вас пока нет сохранённых адресов`; action `Добавить адрес`.
- Form: `Название адреса`, placeholder `Например, Дом или Работа`; AddressFields labels from 55b.
- Actions: `Сохранить`, `Сохраняем…`, `Отмена`, `Изменить`, `Удалить`, `Сделать основным`.
- Default badge: `Основной адрес`; success `Адрес сохранён` / `Адрес удалён` / `Основной адрес изменён`.
- Delete confirmation: `Удалить этот адрес?`, `Удалить`, `Отмена`; cancel preserves state.
- Legacy review: `Проверьте и уточните адрес`; preserve/show full raw text, initialize structured street
  empty and require explicit correction. Never truncate or attempt to parse apartment from legacy text.
- Unavailable town: `Доставка в этот населённый пункт сейчас недоступна`; keep the entry editable/deletable.
- Loading: `Загружаем адреса…`; retry: `Попробовать ещё раз`.
Use backend/entity validation and storage error copy verbatim. Surface corrupt-storage reset only after
`Сбросить сохранённые адреса?` confirmation; unsupported schemas remain read-only.

Use existing tokens/controls for light/dark. Inline editing or an accessible dialog is acceptable; no UI kit.
Keyboard focus enters the editor/confirmation, returns to its trigger, or to the next entry/add button after
deletion. Status/errors are announced. Buttons have at least 44 px touch targets. Avoid nested interactive
elements. Names are optional; fallback display is formatted address text. Changing default does not rename
or reorder data unpredictably (keep ascending backend ID or guest creation order).

Navigation links use `Мои адреса`; checkout linking comes in 57. Do not add authentication requirements
to the entire account layout. Page-specific state stays in pages unless reused; management feature is
shared by web and mini. No cross-page or cross-feature imports.

## Tests to write
- `managementActions`: all actions validate input/context; success, duplicate create, edit conflict, missing
  entry, expired session, backend failure and stale account return correct outcomes without guest fallback.
- `guestManagement`: add/name/edit/default/delete/cancel, reload, empty state, duplicate/conflict and storage reset.
- `accountManagement`: own entries persist across sessions, another account sees only its own data;
  web/mini use the intended session, account values never enter guest localStorage.
- `legacyAndUnavailable`: full legacy text retained until explicit correction; missing town stays manageable.
- `managementAccessibility`: keyboard/focus/announcements, no overflow at 390/1440, both themes.

## API documentation
No endpoint changes. Add QA setup and required session/test-data inputs to the scripts' headers.

## Done when
`node scripts/qa/address-management-harness.mjs`, `npm run check`, `npm run build` and
`node scripts/qa/flows/my-addresses.mjs` pass for this task's flows against a production build.
The live backend must already be running. Under AGENTS.md, ask the owner to run/authorize live mutations;
provide concrete setup commands in the QA header. Local guest tests need no backend writes. Record unavailable
authenticated/native-device checks as blockers, not passes. Tick 56 and commit `task 56: manage saved addresses`.

## Out of scope
Checkout integration (57), import (58), backend changes, live orders/payments and deployment.
