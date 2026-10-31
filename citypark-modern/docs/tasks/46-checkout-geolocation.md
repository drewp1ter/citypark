# Task 46: Automatic checkout location with DaData

## Goal
Automatically suggest an editable delivery address and select a supported delivery town from the customer's location on website and Telegram Mini App checkout. Manual entry always remains available.

## Depends on
Website task 45. Existing backend delivery-prices API; no new backend or admin task.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/widgets/food-ordering/ui/checkout-form.tsx`
- `src/widgets/food-ordering/ui/address-field.tsx`
- `src/shared/api/dadata.ts`

## Files to create
- `app/api/address-geolocate/route.ts` — thin route re-export.
- `src/app/api/address-geolocate.ts` — validation, provider call and delivery-town matching.
- `src/entities/dish/lib/match-delivery-town.ts` — pure locality matching.
- `src/widgets/food-ordering/model/use-checkout-location.ts` — location request lifecycle.
- `scripts/qa/flows/checkout-geolocation.mjs` — targeted regression flow.

## Files to change
- `src/shared/api/dadata.ts` and `src/shared/api/index.ts` — server-side reverse geocoding and export.
- `src/entities/dish/index.ts` — export the matching helper.
- `src/widgets/food-ordering/ui/checkout-form.tsx` — empty initial town, location integration and quote gating.
- `src/widgets/food-ordering/ui/address-field.tsx` — controlled street value in both input modes.
- `src/pages/checkout/ui/checkout-page.tsx` and `src/pages/mini/ui/mini-checkout-page.tsx` — pass configuration availability to shared checkout.
- `.env.example` — document that DADATA_API_KEY enables both suggestions and reverse geocoding.
- `docs/PLAN.md` — document the frontend route and fallback behavior.
- Existing checkout, promo and Mini App QA flows — explicitly select a town where they previously relied on the first-town default.
- `docs/tasks/PROGRESS.md` — tick only after implementation checks pass.

## Spec
### Location and form lifecycle
- Enable on both surfaces using browser `navigator.geolocation`; native Telegram location integration is out of scope.
- When a nonempty delivery checkout is ready, automatically request location once per mounted checkout, with `timeout: 10000`, `maximumAge: 0`, `enableHighAccuracy: false`. Prevent duplicate requests during React effect replay. No automatic retries or repeated permission prompts.
- Skip location requests when DADATA_API_KEY is absent, geolocation is unsupported, or restored/customer-entered town or street values exist. Keep manual checkout functional.
- Start with no selected town instead of towns[0]. Preserve restored form values. Do not clear customer input on failure.
- Keep manual fields enabled during detection. Ignore pending results after manual town/street edits, switching to pickup, or unmounting. Never overwrite customer edits, including edits subsequently cleared by the customer.
- Apply detected town and street atomically. Make the street controlled in both plain TextField and suggesting ComboBox modes; avoid the existing town-change effect clearing freshly detected text.
- Preserve manual town-change clearing, DaData typed suggestions, apartment fields, validation-error restoration and payment-return behavior. Do not request typed suggestions until a town is selected.
- Suppress delivery quotes until a town is selected; invalidate stale totals. Recalculate normally after selection and preserve authoritative backend pricing and submission validation. Pickup quoting remains unchanged.
- Accessible nonblocking status text (role=status): pending `Определяем ваш адрес…`; success `Адрес определён. Проверьте улицу и номер дома.`; failure/unavailable `Не удалось определить адрес. Выберите населённый пункт и введите адрес вручную.`; unmatched `Не удалось подобрать населённый пункт доставки. Выберите его вручную.`. Clear pending status when the customer takes over editing or switches to pickup.

### Server integration
- Reuse the existing server-only DADATA_API_KEY. Read installed Next.js route-handler documentation before implementation.
- Add POST `/api/address-geolocate`, available to guest and authenticated checkouts without an account requirement.
- Request: `{ "lat": 44.77, "lon": 39.87 }`. Require finite numeric latitude in [-90,90] and longitude in [-180,180]; malformed JSON or invalid coordinates return HTTP 400 `{ "status": "invalid" }`.
- Call `https://suggestions.dadata.ru/suggestions/api/4_1/rs/geolocate/address` with token authentication, JSON body `{ lat, lon, count: 1, radius_meters: 100, language: "ru" }`, and a three-second timeout. Validate the provider response before using it.
- Official reference: https://dadata.ru/api/geolocate/ (results ordered by distance). Use the first returned address; never search farther away for a supported delivery town.
- Match against getDeliveryTowns() on the server. Prefer settlement when supplied, otherwise city; do not fall back from an unsupported settlement to its parent city. Require provider region value `Краснодарский` and region_type `край`, consistent with the existing regional address-suggestion restriction.
- Normalize case, whitespace, ё/е and recognized locality prefixes (г/город, п/пос/посёлок, х/хутор, ст/ст-ца/станица, с/село, д/деревня, аул), including optional periods. Require exactly one normalized delivery-town match; no fuzzy or nearest-town matching. Return its original canonical name.
- Construct street text from street_with_type, house_type/house, block_type/block; omit absent components, never invent a house or apartment. If no street exists, return an empty street string.
- HTTP 200 success: `{ "status": "matched", "town": "Белореченск", "street": "ул Гоголя, д 61" }`. Empty provider results, unsupported region/locality or ambiguous matches: HTTP 200 `{ "status": "unmatched" }`.
- Missing key: HTTP 503 `{ "status": "unavailable" }`. Provider timeout, non-2xx, malformed provider response or delivery-list failure: HTTP 502 `{ "status": "unavailable" }`. All client failure paths retain manual entry.
- Send Cache-Control: no-store; do not cache provider requests. Never expose credentials/raw provider responses or log coordinates/addresses. Order payloads remain unchanged.

## Tests to write
Use pure matching cases and intercepted provider/geolocation responses; do not create real orders or substitute a mock for the required running backend. Exercise the actual route against controlled provider responses in an isolated test harness.
- Matching: city/settlement prefixes, case/spacing and ё/е normalize; duplicate normalized names, another region, unsupported settlement and empty results do not match.
- Route: valid coordinates produce canonical town/street; invalid JSON, strings, missing values and out-of-range coordinates return 400; missing key returns 503; timeout, non-2xx, malformed response and delivery-list failure return 502. Missing house/block/street components are handled without invented text.
- Lifecycle: success, permission denied, unsupported API, geolocation timeout, provider failure and unmatched town; no duplicate request after effect replay and no automatic retry.
- Races: delayed results cannot overwrite edits, pickup selection, restored values or an unmounted form. Atomic town/street autofill survives town-change handling.
- Forms/pricing: plain and suggesting inputs stay editable; manual town change clears street; form errors retain values; no delivery quote before town selection; selected town triggers correct quote; pickup and promo behavior remain intact.
- Cover 390/1440 px and both themes. Exercise shared behavior on both surfaces; authenticated Telegram device permission checks require supplied session/device access and must be reported as pending if unavailable.

## API documentation
Document POST /api/address-geolocate and DADATA_API_KEY behavior in docs/PLAN.md. This is a Next.js frontend route; do not modify citypark-go/docs/API.md or backend OpenAPI.

## Done when
```bash
npm run check
npm run build
# Against the production frontend at BASE_URL and the configured running backend:
node scripts/qa/flows/checkout-geolocation.mjs
node scripts/qa/flows/checkout-and-contacts.mjs
node scripts/qa/flows/promo-checkout.mjs
node scripts/qa/flows/telegram-mini.mjs
```
Required checks pass and targeted flow covers route/matching/lifecycle cases above. Do not start backend/database services or perform live writes without authorization. Report unavailable authenticated/device checks honestly. Tick task and commit `task 46: Automatic checkout location with DaData` only after completion under project conventions.

## Out of scope
Backend/admin changes, database migrations, Yandex geocoding, maps, location persistence/tracking, native Telegram location API, automatic retries, nearest-town selection, delivery-price changes, real payments and order creation.
