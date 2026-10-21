# Conventions

Read this at the start of every task. Follow it exactly; later tasks rely on it.

## Before you write code

- Read `CLAUDE.md` (design, tokens, accessibility, FSD) and `AGENTS.md`. They are the project rules.
- This project uses Next.js 16. Read the guide in `node_modules/next/dist/docs/` for every Next.js API you use
  (caching, `error.js`, `next/image`, config). Your memory of older versions is often wrong here.
- `docs/PLAN.md` explains the architecture. Its section "Backend: seams now, integrations later" says where backend
  calls go.
- The backend API contract is `../citypark-go/docs/API.md`. Do not change files in `../citypark-go/`.

## Code rules

- FSD layers: `app` → `pages` → `widgets` → `features` → `entities` → `shared`. A layer imports only from layers
  below it. Import a slice through its `index.ts`. `npm run lint:fsd` (Steiger) must pass with no overrides.
- Route files in the root `/app` only re-export from `src`.
- Pages are Server Components. Only interactive parts are client components.
- Backend calls happen on the Next server only (Server Components, Server Actions). The browser never calls the backend.
- Colours only from tokens (`src/app/styles/tokens.css`). No hex colours in components.
- Text shown to users is in Russian. Code comments, logs and errors are in English.
- Keep exported names exactly as the task spells them. Other tasks use them.
- Match the comment density, naming and style of the code around you.

## Done means

Every task ends with these passing, plus whatever the task adds:

```
npm run check            # typecheck + eslint + Steiger, no errors and no warnings
npm run build
```

The QA flows the task names run against a production build (`npm run build && npm start -- -p 3111`) and print
expected values next to real ones; read the output and fix every difference.

Then tick the task in `docs/tasks/PROGRESS.md` and commit as `task NN: <title>`.

If the task cannot be done as written, stop. Write one line under the task in `PROGRESS.md` that says what blocked
you, and do not tick it.
