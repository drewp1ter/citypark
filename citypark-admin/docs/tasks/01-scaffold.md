# Task 01: Project scaffold

## Goal
An empty but complete project: Vite + React 19 + TypeScript, HeroUI v3 on Tailwind v4, TanStack Router (file routes)
and Query, ESLint, Prettier, Vitest and Playwright, with `npm run check`, `npm run build` and `npm run e2e` working and
one placeholder page «Сити Парк — админка» in light and dark themes.

## Depends on
Nothing.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `../citypark-modern/package.json` and `../citypark-modern/eslint.config.mjs`
(versions and lint style used in this workspace), `../citypark-modern/.prettierrc*` if present.

## Background
This folder already contains `CLAUDE.md` and `docs/`. It lives inside the workspace git repository (like
`citypark-go/` and `citypark-modern/`), so do not run `git init`.

## Files to create
- `package.json`, `package-lock.json`, `tsconfig.json` (+ `tsconfig.node.json` if the Vite template uses it),
  `vite.config.ts`, `index.html`, `eslint.config.js`, `.prettierrc`, `.prettierignore`, `.gitignore`, `.env.example`,
  `vitest.config.ts` (or `test` in `vite.config.ts`), `playwright.config.ts`
- `src/main.tsx`, `src/router.tsx`, `src/styles.css`, `src/routes/__root.tsx`, `src/routes/index.tsx`,
  `src/shared/lib/format.ts`, `src/shared/lib/format.test.ts`, `src/shared/ui/theme-toggle.tsx`
- `e2e/smoke.spec.ts`, `README.md` (commands from `CLAUDE.md`, two lines on what the app is)

## Spec

### Packages (latest stable at the time of the task)
Runtime: `react`, `react-dom`, `@heroui/react`, `@heroui/styles` (+ their peer dependencies), `@tanstack/react-router`,
`@tanstack/react-query`. Dev: `vite`, `@vitejs/plugin-react`, `@tanstack/router-plugin`, `typescript`,
`tailwindcss`, `@tailwindcss/vite`, `eslint` + `typescript-eslint` + `eslint-plugin-react-hooks`, `prettier` +
`prettier-plugin-tailwindcss`, `vitest`, `@playwright/test`. Later tasks add `openapi-*`, Tiptap and Recharts.

### Scripts
```json
"dev": "vite",
"build": "tsc -b && vite build",
"preview": "vite preview",
"typecheck": "tsc -b --noEmit",
"lint": "eslint .",
"format": "prettier --write .",
"test": "vitest run",
"check": "npm run typecheck && npm run lint && prettier --check . && npm run test",
"e2e": "playwright test"
```
Prettier: the same style as `citypark-modern` (tabs, single quotes, no semicolons, print width 120) if its config says
so; otherwise Prettier defaults.

### Styles and theme
`src/styles.css`: Tailwind v4 import and HeroUI v3 styles as its docs say. Define the brand accent once: lime `#D6F24A`
with dark text `#111111` on it (from `../citypark-modern/design/tokens.css`), mapped onto HeroUI's accent/primary
variable the way the HeroUI v3 theming docs describe. Dark mode: HeroUI v3's dark theme switched by its documented
attribute/class on `<html>`; `ThemeToggle` (icon button, `aria-label` «Сменить тему») cycles light/dark and stores the
choice in `localStorage` (`citypark-admin-theme`), default from `prefers-color-scheme`. Set the theme before first paint
(a tiny inline script in `index.html`).

### Routing
TanStack Router file routes with the Vite plugin (generated `routeTree.gen.ts`, ignored by ESLint and Prettier,
committed). `__root.tsx` renders `<Outlet />` inside a `QueryClientProvider`. `index.tsx` shows a heading
«Сити Парк — админка» and the theme toggle. `<html lang="ru">`, `<title>Сити Парк — админка</title>`.

### `format.ts`
```ts
/** 4000 → "4 000 ₽", 5.5 → "5,50 ₽" (non-breaking spaces, ru-RU). */
export function formatRub(value: number): string
/** "2026-10-22T09:05:00Z" → "22.10.2026, 12:05" in Europe/Moscow. */
export function formatDateTime(iso: string): string
/** "2026-10-22" → "22.10.2026". */
export function formatDate(isoDate: string): string
```

### Env
`.env.example`: `VITE_API_URL=http://localhost:3000/api/v1` with a comment. Not read yet (task 02).

### Playwright
`webServer: { command: 'npm run dev', url: 'http://localhost:5173', reuseExistingServer: true }`, Chromium only,
`baseURL` `http://localhost:5173`. `e2e/smoke.spec.ts`: the heading is visible; the theme toggle switches the theme
attribute/class on `<html>`; the choice survives a reload.

## Tests to write
- `format.test.ts`: `formatRub(4000)` → `4 000 ₽` (with ` `), `formatRub(0)` → `0 ₽`, `formatRub(5.5)` →
  `5,50 ₽`, `formatRub(150)` → `150 ₽`; `formatDateTime` for a UTC time → Moscow time; `formatDate`.
- `e2e/smoke.spec.ts` as above.

## Done when
CONVENTIONS "Done means" passes (`npm run e2e -- e2e/smoke.spec.ts`; no backend needed for this one), and
`npm run dev` shows the page in both themes at 390 and 1440 px.

## Out of scope
API client and auth (task 02), the layout with navigation (task 03).
