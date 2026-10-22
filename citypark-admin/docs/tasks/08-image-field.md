# Task 08: Image field (upload, alt text, focal point)

## Goal
A reusable `ImageField` lets staff upload a photo, see it, write its alt text and set its focal point by clicking on
the photo. Every content form (rooms, news, page blocks) uses it.

## Depends on
02, and citypark-go task 45.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/shared/api/client.ts` (`apiUpload`), `../GLOSSARY.md` (Focal point),
`../citypark-go/docs/API.md` (section `### Uploads`).

## Background
Upload is a separate step: `POST /uploads` (multipart, field `file`, JPEG/PNG/WebP up to 10 MB) returns
`{ url }`, and the form sends that URL when it is saved. Photos shipped with the site have paths like
`/images/room-vip.jpg`; show them from `VITE_SITE_URL` (new env, default `http://localhost:3001`) + path.
The focal point is a CSS `object-position` string `"x% y%"` (integers 0–100); the site uses it when it crops.

## Files to create
- `src/shared/ui/image-field.tsx`, `src/shared/lib/focal-point.ts` (+ test), `src/shared/lib/image-src.ts` (+ test)
- `src/routes/_app/dev.image-field.tsx`: a hidden demo page (not in the navigation) used by the e2e test
- `e2e/image-field.spec.ts`, `e2e/fixtures/photo.jpg` (a small JPEG, < 50 KB), `e2e/fixtures/not-an-image.txt`

## Files to change
- `.env.example`: `VITE_SITE_URL=http://localhost:3001` with a comment.

## Spec
```ts
export type PhotoValue = { url: string; alt: string; position: string }
type Props = {
	label: string
	value: PhotoValue
	onChange: (v: PhotoValue) => void
	/** Aspect ratio of the preview, like the place the photo is shown on the site (e.g. 4 / 3). */
	aspect?: number
	allowEmpty?: boolean // shows «Убрать фото» (news); required fields do not
	errors?: Partial<Record<'url' | 'alt' | 'position', string>>
}
export function ImageField(props: Props): JSX.Element
```
- Empty: a drop zone «Перетащите фото или выберите файл» (also a real file input with a label, keyboard accessible);
  accepts `image/jpeg,image/png,image/webp`; files > 10 MB are rejected before upload («Файл больше 10 МБ»).
- Uploading: progress spinner over the preview; errors from the server shown under the field.
- With a photo: a preview with `object-fit: cover` in the given aspect and `object-position` = `position || '50% 50%'`;
  a marker at the focal point. Clicking the preview (or arrow keys when it has focus, 5 % per press) moves the point:
  `position` = `toPosition(x, y)`. Buttons «Заменить», «Сбросить точку» and, if `allowEmpty`, «Убрать фото».
  A second small preview shows a square crop so the effect of the point is visible.
- «Описание фото (alt)» text input under it, max 200, help text «Что на фото, для незрячих и поисковиков».
`focal-point.ts`:
```ts
export function toPosition(xRatio: number, yRatio: number): string // clamps to 0..1, rounds to integer percent: "37% 60%"
export function parsePosition(p: string): { x: number; y: number }  // "" or invalid → { x: 50, y: 50 }
```
`image-src.ts`: `imageSrc(url)` → `''` for `''`; absolute URLs unchanged; paths starting with `/images/` →
`VITE_SITE_URL + url`.

## Tests to write
- `focal-point.test.ts`: `toPosition(0.374, 0.6)` → `37% 60%`; clamping `-1`/`2`; `parsePosition('20% 80%')`;
  invalid → centre.
- `image-src.test.ts`: the three cases.
- `e2e/image-field.spec.ts` on the demo page (backend running): upload `photo.jpg` → preview visible and the shown
  value has a `/uploads/` URL; click the top-left area → position near `10% 10%`; upload the `.txt` file → the server's
  error «Допустимы только JPEG, PNG и WebP».

## Done when
CONVENTIONS "Done means" passes with `npm run e2e -- e2e/image-field.spec.ts`.

## Out of scope
Cropping or resizing in the browser, several photos at once, a media library.
