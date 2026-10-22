# Task 12: News

## Goal
«Новости» lists news items newest first; an admin creates, edits and deletes them, with a rich text body and one
optional photo. A save is live on the site within a minute.

## Depends on
11, and citypark-go task 49.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/features/rooms/` (form pattern from task 11),
`src/shared/ui/image-field.tsx`, `src/shared/ui/rich-text-editor.tsx`, `../citypark-go/docs/API.md` (section
`### News`).

## Background
The backend makes the slug (the item's address on the site, `/news/<slug>`) from the title when it is empty and makes
it unique; a slug typed by the admin that exists gives 409. The date defaults to today on the server. The list
endpoint has no `body`; the edit page loads the item with `GET /news/{slug}`.

## Files to create
- `src/features/news/api.ts`, `src/features/news/model/form.ts` (+ test), `src/features/news/ui/news-page.tsx`,
  `ui/news-form.tsx`, `ui/delete-news-dialog.tsx`, `src/routes/_app/content.news.new.tsx`,
  `src/routes/_app/content.news.$slug.tsx`, `e2e/news.spec.ts`

## Files to change
- `src/routes/_app/content.news.tsx` (search param `page`)

## Spec
List: `PageHeader` «Новости» with «Добавить новость»; rows with photo thumbnail (or «Без фото»), title, `formatDate`,
a link «На сайте ↗» (`VITE_SITE_URL/news/<slug>`, new tab), «Изменить», «Удалить». Pages of 20 with «Найдено: N».
Form (new and edit share it): «Заголовок» (required, ≤ 200), «Короткий заголовок» (≤ 100, help «Для главной и мобильной
версии, если заголовок длинный»), «Анонс» (≤ 300, textarea), «Дата» (HeroUI date picker, default today),
«Адрес на сайте» (slug; on a new item empty with help «Оставьте пустым — заполнится из заголовка»; only latin, digits and
dashes), «Фото» (`ImageField`, aspect 16/9, `allowEmpty`), «Текст» (`RichTextEditor` profile `article`, maxLength
50 000). Create → `POST` → go to the edit page of the returned slug with toast «Новость опубликована»; edit → `PUT`,
toast «Сохранено» (if the slug changed, replace the URL). 409 → slug field error «Такой адрес уже занят». Delete
confirm: «Удалить новость «…»? Она сразу исчезнет с сайта.». Unsaved-changes blocker as in task 11.
`form.ts`: `emptyNews(today)`, `toForm(item)`, `toRequest(form)`, `validateNews(form)` (slug pattern
`^[a-z0-9]+(-[a-z0-9]+)*$` when not empty), `mapServerFields`.

## Tests to write
- `form.test.ts`: validation of title and slug; `toRequest` trims and keeps empty slug as `''`; date format.
- `e2e/news.spec.ts`: create «Тест <timestamp>» with a bold word and the fixture photo → it is first in the list with
  today's date; open it → the body is bold, the photo shown; change the title → saved; create another with the same
  explicit slug → «Такой адрес уже занят»; delete both → gone.

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/news.spec.ts`.

## Out of scope
Drafts, scheduled publishing, galleries, the blog (removed).
