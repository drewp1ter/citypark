# Task 03: Sign-in, access guard and app shell

## Goal
`/login` signs an admin in; every other page requires a session with role `admin`. Signed-in pages share a shell: a
sidebar with the sections, the user's e-mail and «Выйти», the theme toggle, and a drawer on phones.

## Depends on
02.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/api/` (all files), `src/routes/__root.tsx`,
`../citypark-go/docs/API.md` (sections `POST /auth/sign-in`, `GET /user`), the TanStack Router guide on
authenticated routes (`beforeLoad` + `redirect`) for the installed version.

## Files to create
- `src/routes/login.tsx`, `src/routes/_app.tsx` (pathless layout route with the guard and the shell),
  `src/routes/_app/index.tsx` (redirects to `/orders`), placeholder routes for each section (Spec 3)
- `src/features/auth/api.ts`, `src/features/auth/ui/login-form.tsx`, `src/features/auth/model/next-path.ts` (+ test)
- `src/shared/ui/app-shell.tsx`, `src/shared/ui/nav-items.ts`, `src/shared/ui/page-header.tsx`,
  `src/shared/ui/toast.ts` (a thin wrapper over HeroUI's toast: `toastError(err: unknown)`, `toastSuccess(text)`)
- `e2e/auth.spec.ts`, `e2e/helpers.ts` (`signIn(page)` with the dev admin)

## Spec

### 1. Login
Form: «E-mail» (type email, required), «Пароль» (type password, required), button «Войти». Submit →
`POST /auth/sign-in`:
- 200 and `role === 'admin'` → `setSession({ token, email, name })`, navigate to `next` (Spec 1a) or `/orders`;
- 200 and another role → do not store the token; show «Нет доступа: нужен аккаунт администратора»;
- 401 → «Неверный e-mail или пароль»; 400 → field errors; network → the `ApiError` message.
The button shows a spinner while the request runs and the form stays filled after an error.
1a. `safeNextPath(next: string | undefined): string`: only paths starting with `/` and not `//`, not `/login`;
otherwise `/orders`.

### 2. Guard
`_app.tsx` `beforeLoad`: no session → `redirect({ to: '/login', search: { next: location.href } })`. `/login` with a
session → redirect to `/orders`. A 401 later is handled by the client (task 02).

### 3. Shell and sections
`nav-items.ts` (in this order; `to` is the route):
| Label | Route | Task |
|---|---|---|
| Заказы | `/orders` | 05 |
| Доставка | `/delivery` | 04 |
| Меню | `/menu` | 07 |
| Номера | `/content/rooms` | 11 |
| Новости | `/content/news` | 12 |
| Главная | `/content/home` | 13 |
| Отель | `/content/hotel` | 13 |
| Настройки сайта | `/content/settings` | 14 |
| Цены отеля | `/content/hotel-prices` | 15 |
| Свободные номера | `/free-rooms` | 06 |
| Отчёты | `/reports` | 16 |
Group the content items under a sidebar heading «Контент». Each route gets a placeholder page with its `PageHeader`
title and «Раздел в разработке» until its task replaces it.
Desktop (≥ 1024 px): fixed sidebar 240 px with the logo text «Сити Парк», the items (active one highlighted, the link
has `aria-current="page"`), and at the bottom the e-mail, `ThemeToggle` and «Выйти». Below 1024 px: a top bar with a
menu button (`aria-label` «Открыть меню») that opens the same navigation in a HeroUI drawer; choosing an item closes
it. «Выйти» → `clearSession()` → `/login`.
`PageHeader({ title, actions? })`: `h1` and a slot for buttons on the right (they wrap under the title on phones).

## Tests to write
- `next-path.test.ts`: `/orders?page=2` → kept; `//evil.com` → `/orders`; `https://evil.com` → `/orders`;
  `/login` → `/orders`; `undefined` → `/orders`.
- `e2e/auth.spec.ts` (backend running): wrong password → the error text; dev admin → lands on «Заказы»; reload keeps
  the session; «Выйти» → `/login`; opening `/delivery` signed out → `/login?next=...` → after sign-in back on
  `/delivery`; at 390 px the menu button opens the drawer and navigates.
- (Optional if a `user` account can be created through `POST /auth/sign-up`): a non-admin sees
  «Нет доступа: нужен аккаунт администратора».

## Done when
CONVENTIONS "Done means" passes (`npm run e2e -- e2e/auth.spec.ts e2e/smoke.spec.ts`; update the smoke test to the
new start page).

## Out of scope
Feature screens, password reset, token refresh (the backend has none; a 401 sends the user to sign in again).
