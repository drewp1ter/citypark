# Conventions

Read this at the start of every task. Follow it exactly; later tasks rely on it.

## Before you write code

- Read `CLAUDE.md` (stack, layout, rules).
- Read the docs of every library API you use (HeroUI v3, TanStack Router/Query, Tiptap, Recharts) for the installed
  version: `node_modules/<pkg>/README.md`, its `dist` types, or context7. Do not rely on memory of older versions.
- The backend contract is `../citypark-go/docs/API.md`. Use the generated types; never hand-write an API type that the
  schema has.

## Code rules

- TypeScript strict, no `any`, no `@ts-ignore`. Prefer `type` over `interface`.
- Server state only through TanStack Query: one `queryOptions`/hook per endpoint in the feature's `api.ts`, query keys
  as arrays starting with the feature name (`['orders', filters]`). Mutations invalidate the queries they change.
- Requests only through `api` from `src/shared/api/client.ts` (task 02). Errors come back as `ApiError`
  (`status`, `message`, `fields`); show them with the shared helpers from task 02.
- URL state (filters, page, open item) lives in route search params, validated with the route's `validateSearch`.
- Components small, one per file, in `ui/`. Pure logic (mapping, validation, formatting) in `model/` or
  `shared/lib/`, with Vitest tests next to it (`*.test.ts`).
- Russian user-facing text, written in place (no i18n layer). Keep exported names exactly as the task spells them.

## Tests

- Vitest for every pure function with a branch, a loop or a format (`*.test.ts` next to the file).
- Playwright flows in `e2e/<feature>.spec.ts` for every feature screen, against the real backend on
  `http://localhost:3000` and the dev server (Playwright's `webServer` starts `npm run dev`). Sign in through the UI
  with the dev admin. A flow may create data; it deletes what it created at the end, or uses unique names
  (`Тест ${Date.now()}`).
- Use roles and labels in locators (`getByRole('button', { name: 'Сохранить' })`), not CSS classes.

## Done means

Every task ends with these passing, plus whatever the task adds:

```
npm run check
npm run build
npm run e2e -- <the task's spec files>     # with the backend running
```

Then tick the task in `docs/tasks/PROGRESS.md` and commit as `task NN: <title>`.

If the task cannot be done as written (for example, the backend endpoint is missing or behaves differently from
`API.md`), stop. Write one line under the task in `PROGRESS.md` that says what blocked you, and do not tick it.
