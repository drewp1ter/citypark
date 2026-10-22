# citypark-admin

Admin panel for «Сити Парк» (hotel, restaurant and food delivery in Белореченск): delivery prices, order history,
the menu's dish photos and sync, the site's content (rooms, news, page texts, contacts, hotel prices), free rooms and
the hotel reports. It is a static single-page app that talks to the backend `../citypark-go` with a Bearer JWT.
Why it looks like this: `../docs/adr/0001-admin-panel.md`. Terms: `../GLOSSARY.md`.

## How to work in this repository

1. Read `docs/tasks/CONVENTIONS.md`.
2. Open `docs/tasks/PROGRESS.md` and take the first unchecked task.
3. Read that task file in `docs/tasks/` and the files it lists under "Read first". Do not read other task files unless
   the task says so.
4. Do only that task. Run every command under "Done when".
5. Tick the task in `PROGRESS.md` and make one commit: `task NN: <title>`.

The backend API contract is `../citypark-go/docs/API.md` (OpenAPI: `../citypark-go/internal/apidocs/openapi.yaml`).
Do not change files in `../citypark-go/` or `../citypark-modern/`. If the backend lacks something a task needs, stop and
write it under the task's line in `PROGRESS.md`.

Do one task per session. If a task cannot be completed as written, stop and write the reason under the task's line in
`PROGRESS.md` instead of improvising.

## Stack (decided, keep unless the owner asks otherwise)

- **Vite + React 19 + TypeScript** (strict). Package manager: npm.
- **HeroUI v3** (`@heroui/react`, `@heroui/styles`) on **Tailwind CSS v4**. HeroUI v3 differs a lot from v2 (no
  `HeroUIProvider`, compound components, React Aria underneath): read its docs (context7 library for HeroUI, version 3,
  or the README in `node_modules/@heroui/react`) before using a component. Your memory of v2 is wrong here.
- **TanStack Router** (file-based routes, `@tanstack/router-plugin` for Vite) and **TanStack Query** for server state.
- **openapi-typescript + openapi-fetch**: types generated from the backend's OpenAPI file (`npm run api:gen`); all
  requests go through `src/shared/api/client.ts`.
- **Tiptap** for rich text, **Recharts** for charts, **Vitest** for logic, **Playwright** for user flows.
- UI language: **Russian only**, no i18n. Code, comments and logs in English.

## Layout

```
src/
  main.tsx, router.tsx        entry, router and QueryClient
  routes/                     TanStack file routes; thin: they compose feature components
  features/<name>/            one folder per section: api.ts (queries/mutations), ui/*.tsx, model/*.ts (pure logic)
  shared/api/                 client.ts, schema.d.ts (generated, never edit), auth.ts (token storage)
  shared/ui/                  reusable components (ImageField, RichTextEditor, PageHeader, ...)
  shared/lib/                 formatting and other pure helpers
e2e/                          Playwright flows
```
A feature imports only from `shared` and never from another feature. Routes import features.

## Rules

- Colours and spacing from HeroUI's theme and Tailwind; no hex colours in components. The brand accent is defined
  once in `src/styles.css` (task 01).
- Every screen works at 390 px width and on desktop. Light and dark themes.
- Every form shows the backend's field errors (`errors` in the 400 body) next to the fields and its `message` in a
  toast. Never lose what the user typed on an error.
- Money is rubles as numbers in the API; show it with `formatRub` (`4 000 ₽`, non-breaking spaces).
- Labels on every input, `aria-label` on icon buttons, visible focus.
- No `localStorage` for anything but UI preferences (theme). The token lives in `sessionStorage`.

## Commands

```bash
npm run dev          # http://localhost:5173, backend expected at VITE_API_URL (default http://localhost:3000/api/v1)
npm run check        # typecheck + eslint + prettier --check + vitest run; must be clean
npm run api:gen      # regenerate src/shared/api/schema.d.ts from ../citypark-go/internal/apidocs/openapi.yaml
npm run build        # static files in dist/
npm run e2e          # Playwright flows; needs the backend running (cd ../citypark-go && make db-up && make run)
```
The dev admin account comes from the backend's `.env`: `ADMIN_EMAIL=dev@dev.io`, `ADMIN_PASSWORD=devdevdev`.
