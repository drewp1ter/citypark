# Task 18: News from the backend

## Goal
`getNews()` and `getNewsItem(slug)` read citypark-go `/news`, and `/news/[slug]` shows the article body as rich text.
News added or edited in the admin appear on `/`, `/news` and their own page within a minute.

## Depends on
12, 17, and citypark-go task 49.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/article/` (all files), `src/pages/news/ui/news-page.tsx`,
`src/pages/news-article/ui/article-page.tsx`, the route file of `/news/[slug]` in `/app`,
`src/pages/home/ui/news-section.tsx`, `src/shared/ui/rich-text.tsx`, `src/entities/room/api/rooms-api.ts` (fetch
pattern from task 17), `../citypark-go/docs/API.md` (section `### News`).

## Background
The backend seeded the seven news items of `news.data.ts` (citypark-go task 49). Items without a photo have
`photo.url` `""`; the site shows `PLACEHOLDER_PHOTO` for them, as now. The body is HTML of the *article* profile
(`p`, `h2`, `h3`, `strong`, `em`, `a`, `ul`, `ol`, `li`, `blockquote`, `br`), sanitized by the backend.

## Files to create
- `src/entities/article/api/news-api.ts`

## Files to change
- `src/entities/article/api/get-news.ts`: both getters (Spec 1).
- `src/entities/article/model/types.ts`: `body?: readonly string[]` → `bodyHtml: string` (`''` when empty).
- `src/entities/article/model/news.data.ts`: delete it.
- `src/pages/news-article/ui/article-page.tsx`: render `bodyHtml` with `<RichText variant='article'>`; when it is
  `''`, keep today's behaviour for an article without text.
- The `/news/[slug]` route: no `generateStaticParams` from a static list any more (read the Next.js 16 docs on dynamic
  routes and caching; the page is rendered on demand and cached for 60 s). `generateMetadata` uses `getNewsItem`.
- `/news`, home news section: handle a failing backend like task 17 (hide the section on `/`; on `/news` show a short
  message «Не удалось загрузить новости.»).

## Spec

### 1. Getters
```ts
/** Newest first. Reads every page of GET /news (pageSize 100). */
export async function getNews(): Promise<readonly NewsItem[]>
/** GET /news/{slug}; undefined on 404. */
export async function getNewsItem(slug: string): Promise<NewsItem | undefined>
```
Mapping: `slug`, `date`, `title`, `shortTitle` (`''` → `undefined`), `excerpt` (`''` → `undefined`), `photo`
(`url ''` → `{ src: PLACEHOLDER_PHOTO, alt: title }`, else `{ src: url, alt, position: position || undefined }`),
`bodyHtml` (`''` in list items, the body in `getNewsItem`). Fetch with `{ next: { revalidate: 60 } }`; non-OK other
than 404 → throw.

### 2. Home section
The home page today picks `new-menu-2026` as its second card by slug. Replace that with "the three newest items" (the
slug may be deleted in the admin).

## Tests to write
No unit-test runner. With citypark-go running:
- `/`, `/news`, `/news/new-menu-2026` look as before (except the home card choice in Spec 2, if it changes what is
  shown; say so in the commit message);
- ask the user to add a news item with a body containing a heading, a list and a link → it appears first on `/news`
  and its page renders the formatting in both themes;
- `/news/does-not-exist` → 404 page.

## Done when
CONVENTIONS "Done means" passes; on a production build `node scripts/qa/flows/news.mjs`,
`node scripts/qa/flows/home.mjs` and `npm run qa:a11y` print no unexpected values.

## Out of scope
Pagination UI on `/news`, galleries, the blog (removed in task 12).
