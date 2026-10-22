# Task 02: Typed API client and token storage

## Goal
All backend calls go through one typed client generated from the backend's OpenAPI file. It adds the Bearer token,
turns error bodies into `ApiError`, and on a 401 clears the token and sends the user to `/login`.

## Depends on
01.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `../citypark-go/docs/API.md` (sections "General" and "Errors"),
the README of `openapi-fetch` and `openapi-typescript` in `node_modules` after installing.

## Files to create
- `src/shared/api/schema.d.ts` (generated), `src/shared/api/client.ts`, `src/shared/api/auth.ts`,
  `src/shared/api/errors.ts`, `src/shared/api/errors.test.ts`, `src/shared/api/auth.test.ts`,
  `src/shared/api/index.ts`, `src/vite-env.d.ts` (if not present: typed `import.meta.env.VITE_API_URL`)

## Files to change
- `package.json`: `openapi-fetch` (dependency), `openapi-typescript` (dev), script
  `"api:gen": "openapi-typescript ../citypark-go/internal/apidocs/openapi.yaml -o src/shared/api/schema.d.ts"`.
- `.prettierignore`, ESLint ignores: `src/shared/api/schema.d.ts`.

## Spec

### `auth.ts`
```ts
const KEY = 'citypark-admin-token'
export type Session = { token: string; email: string; name: string }
export function getSession(): Session | null      // sessionStorage, JSON; invalid JSON → null and the key removed
export function setSession(s: Session): void
export function clearSession(): void
/** Subscribe to sign-in/sign-out in this tab (for React): returns unsubscribe. */
export function onSessionChange(fn: () => void): () => void
```

### `errors.ts`
```ts
export class ApiError extends Error {
	constructor(readonly status: number, message: string, readonly fields: Record<string, string> = {}) { super(message) }
}
/** Builds an ApiError from a fetch Response and its parsed body ({statusCode, message, errors}) or a network failure. */
export function toApiError(status: number, body: unknown): ApiError
```
- body with string `message` → that message; `errors` object of strings → `fields`;
- otherwise by status: 0 (network) → `Сервер недоступен. Проверьте соединение.`; 401 → `Требуется авторизация`;
  403 → `Недостаточно прав`; 404 → `Не найдено`; 5xx → `Ошибка сервера. Попробуйте ещё раз.`;
  other → `Не удалось выполнить запрос (код N)`.

### `client.ts`
```ts
export const api = createClient<paths>({ baseUrl: import.meta.env.VITE_API_URL ?? 'http://localhost:3000/api/v1' })
/** Unwraps an openapi-fetch result: returns data, or throws ApiError. */
export async function unwrap<T>(p: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T>
```
A middleware sets `Authorization: Bearer <token>` when there is a session. On a 401 response to any request **except**
`POST /auth/sign-in`: `clearSession()` and `window.location.assign('/login?next=' + encodeURIComponent(current path))`.
Network errors (fetch throws) become `ApiError` with status 0. A 204 returns `undefined`.

Also export `apiUpload(path: string, file: File): Promise<{ url: string }>` for multipart uploads (plain `fetch` with
the same auth header and error handling), used from task 08.

## Tests to write
- `errors.test.ts`: table tests for `toApiError` (validation body with `errors`; 401 without body; 500 HTML body;
  status 0; 409 with message).
- `auth.test.ts` (jsdom or a sessionStorage stub): set → get; clear → null; corrupted JSON → null and removed;
  `onSessionChange` called on set and clear.
- `client.test.ts`: with `fetch` mocked: the header is sent when signed in; a 401 clears the session and calls
  `location.assign` with `next`; a 401 from `/auth/sign-in` does not; `unwrap` throws `ApiError` with `fields` on 400.

## Done when
CONVENTIONS "Done means" passes; `npm run api:gen` regenerates `schema.d.ts` without a diff.

## Out of scope
Login UI (task 03), feature queries.
