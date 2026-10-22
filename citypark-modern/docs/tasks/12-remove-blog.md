# Task 12: Remove the chef's blog

## Goal
The site has no blog any more: no «Блог шеф-повара» on `/news`, no blog card on the home page, no blog posts at
`/news/[slug]`, and no blog types or data. `/news` shows only news.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`, `../docs/adr/0001-admin-panel.md` (last bullet of "Decision"),
`src/entities/article/index.ts`, `src/entities/article/api/get-news.ts`, `src/entities/article/model/types.ts`,
`src/pages/news/ui/news-page.tsx`, `src/pages/home/ui/news-section.tsx`, `src/pages/home/ui/home-page.tsx`,
`src/pages/news-article/ui/article-page.tsx`, `scripts/qa/flows/news.mjs`.

## Background
The owner decided to drop the blog: nobody will write posts, and the admin panel has no blog section. This is a
deliberate difference from `design/reference/` (news.html and mobile home.html show the blog).

## Files to delete
- `src/pages/news/ui/chef-blog.tsx`, `src/pages/news/ui/section-tabs.tsx` (the tabs only switch between news and blog;
  check that nothing else uses it first)
- Photos used only by the blog: `public/images/blog-*.jpg`, and `public/images/chef.jpg` if nothing else uses it
  (`grep -r` the image names in `src/` before deleting).

## Files to change
- `src/entities/article/model/types.ts`: remove `BlogPost`, `BlogAuthor`.
- `src/entities/article/model/news.data.ts`: remove `blogAuthor`, `blogPosts`.
- `src/entities/article/api/get-news.ts`, `src/entities/article/index.ts`: remove `getBlogPosts`, `getBlogPost`,
  `getBlogAuthor` and their exports.
- `src/pages/news/ui/news-page.tsx`: no tabs, no blog section; the news grid and the social CTA stay. Keep one `h1`.
- `src/pages/home/ui/news-section.tsx` + `home-page.tsx`: the section shows **three news cards**: the two it shows
  today plus the next newest item, in the same card style as the news cards (no blog card, no author). On mobile,
  follow the reference's number of visible cards for news; if the reference shows the blog card in that slot, show
  the third news card there.
- `src/pages/news-article/ui/article-page.tsx` (and its route's `generateStaticParams`/`generateMetadata` if they list
  blog slugs): only news items; an unknown slug, including a former blog slug, → `notFound()`.
- `scripts/qa/flows/news.mjs` (and `home.mjs` if it checks the blog card): remove blog checks; add
  `/news/frittata` → 404 and "home news cards: 3".
- `design/pages.md`: in the `/` and `/news` rows strike the blog parts and add "(blog removed, ADR 0001)".
- `CLAUDE.md` is not changed.

## Spec
- No text «Блог», «шеф-повар» or «рецепт» remains in `src/` (check with `grep -ri`), except the chef's name where the
  restaurant page uses it (`site.chef`).
- The news grid on `/news` keeps its current layout and order.

## Tests to write
No unit-test runner. QA flows above, plus by hand on a production build at 390 and 1440 px, light and dark: `/news` has
no tabs and no blog, the home news section has three news cards and no empty gap.

## Done when
CONVENTIONS "Done means" passes, and on a production build (port 3111) `node scripts/qa/flows/news.mjs`,
`node scripts/qa/flows/home.mjs` and `npm run qa:a11y` print no unexpected values.

## Out of scope
News from the backend (task 18), the article page design.
