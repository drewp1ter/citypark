# Task 19: Site settings from the backend

## Goal
Phones, address, e-mail, company details and social links come from citypark-go `GET /settings`, so the admin can
change them. Server Components read them with `getSiteSettings()`, client components with `useSiteSettings()`. If the
backend does not answer, the site shows the built-in values (contacts must never disappear).

## Depends on
17, and citypark-go task 51.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/shared/config/site.ts`, `src/shared/config/index.ts`, `src/app/layouts/*.tsx`,
`app/layout.tsx`, `src/app/theme/theme-provider.tsx` (provider pattern), `src/widgets/site-header/ui/phone-button.tsx`
(a client component that uses `site`), `src/entities/room/api/rooms-api.ts` (fetch pattern),
`../citypark-go/docs/API.md` (section `### Settings`).

## Background
`site.ts` mixes static parts (name, wordmark, chef, year, nav, legal) with contacts. Only the contacts move to the
backend (citypark-go task 51 seeded them from `site.ts`). About 20 files use `site.phones`, `site.address`,
`site.email`, `site.company` or `site.social`; several are client components (`grep -rln "site\." src`).

## Files to create
- `src/shared/api/get-site-settings.ts` (export from `src/shared/api/index.ts`)
- `src/shared/config/site-settings.ts`: the `SiteSettings` type, `defaultSiteSettings` (today's values), and
  `toSiteSettings(api)`.
- `src/app/providers/site-settings-provider.tsx` (or next to the theme provider, following the app layer's layout):
  `SiteSettingsProvider` and `useSiteSettings()`.

## Files to change
- `src/shared/config/site.ts`: keep only the static parts; contacts move to `defaultSiteSettings`.
- `app/layout.tsx` (or the app-layer root layout it re-exports): fetch settings once and wrap children in the provider.
- Every file that used the moved fields: Server Components call `await getSiteSettings()`, client components call
  `useSiteSettings()`. Shared UI that must stay generic gets the values as props.

## Spec
```ts
export type SitePhone = { label: string; shortLabel: string; display: string; href: string; alt?: { display: string; href: string } }
export type SiteSettings = {
	phones: { restaurant: SitePhone; hotel: SitePhone; free: SitePhone; landline: SitePhone }
	address: { postcode: string; city: string; street: string; full: string }
	email: string
	company: { name: string; inn: string; ogrn: string }
	social: readonly { href: string; label: string; name: string; display: string; handle: string }[]
	mapUrl: string
}
/** GET /settings (revalidate 60); on any error logs it once and returns defaultSiteSettings. Wrapped in React cache(). */
export async function getSiteSettings(): Promise<SiteSettings>
```
`toSiteSettings`: `href` = `tel:` + `number`; `alt` from `altDisplay`/`altNumber` when both are set; apply `nb()` to
`display` values (as `site.ts` did); `address.full` = `${postcode}, г. ${city}, ${street}`; `mapUrl` built from city and
street as `site.ts` does now; `social.handle` = the last path segment of `url`; `social.href` = `url`.

The values on every page must be identical to today's when the backend returns its seed.

## Tests to write
No unit-test runner. With citypark-go running:
- every page at 390 and 1440 px shows the same contacts as before;
- ask the user to change the hotel phone in the database or the admin → after 60 s the header, footer, contacts page
  and hotel page show the new number, and its `tel:` link works;
- backend stopped → all pages still show the default contacts; the server log has the error.

## Done when
CONVENTIONS "Done means" passes; on a production build `npm run qa:flows`, `npm run qa:a11y` and
`npm run qa:keyboard` print no unexpected values.

## Out of scope
Navigation and legal links, opening hours, the map embed.
