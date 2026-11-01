# Task 49: Delivery information from the backend

## Goal
Replace hardcoded delivery information with backend-managed cooking text and outskirts label/range. Reuse Белореченск's existing town tariff and free-delivery threshold in banners and cart estimates.

## Depends on
Website task 48 and [backend task 91](../../../citypark-go/docs/tasks/91-delivery-settings.md), deployed at `MENU_API_URL`. [Admin task 30](../../../citypark-admin/docs/tasks/30-delivery-settings.md) supplies editing controls; the website can be implemented/tested directly against the API contract without waiting for that UI.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/entities/dish/model/delivery.ts`
- `src/entities/dish/lib/quote-delivery.ts`
- `../citypark-go/docs/tasks/91-delivery-settings.md`

## Files to create
- `src/entities/dish/api/get-delivery-info.ts`
- `src/entities/dish/lib/format-delivery-info.ts`
- `src/entities/dish/model/delivery-provider.tsx`
- `src/app/providers/delivery-provider.tsx` and `src/app/providers/index.ts`
- `scripts/qa/flows/delivery-settings.mjs`

## Files to change
- `src/entities/dish/model/delivery.ts`: default informational values and delivery-information types; remove hardcoded authoritative threshold.
- `src/entities/dish/lib/quote-delivery.ts`: accept runtime delivery information and reuse `townDeliveryPrice`.
- `src/entities/dish/index.ts`: export getter, runtime type, quote helper and client provider/hook; remove the old `delivery` constant export after migrating all consumers.
- `app/layout.tsx`: compose the app-layer async delivery provider alongside existing providers.
- `src/pages/home/ui/hero.tsx`, `src/pages/home/ui/delivery-section.tsx`: fetch runtime information for captions, without changing layout/content beyond dynamic delivery facts.
- `src/pages/menu/ui/menu-page.tsx`: runtime badges and threshold visibility.
- `src/widgets/cart-summary/ui/cart-panel.tsx`, `src/widgets/cart-summary/ui/cart-sheet.tsx`: use runtime information from context.
- `docs/tasks/PROGRESS.md`: completion status.

## Spec

### Data and architecture
- Fetch public `GET /delivery-settings` on the Next server only, with `next: { revalidate: 60 }`. Reuse `getDeliveryTowns()` for `GET /delivery-prices`; do not duplicate tariff requests or parsing. Read installed Next documentation before changing caching/layout APIs.
- Expose `getDeliveryInfo(): Promise<DeliveryInfo>` from the dish entity. Use the same getter for server-rendered consumers and the app-layer provider. React request memoization may deduplicate repeated getter calls within one render; do not add another persistent cache on top of the 60-second fetch cache.
- `DeliveryInfo` has `cooking: string`, `outskirtsFee: string`, `outskirtsRange: string`, and `city: DeliveryTown | null`. It contains no separate mutable `freeFrom` field.
- Select the exact normalized locality `Белореченск` using existing locality-normalization helpers; do not choose the first town, a substring match or `address.city` from contact settings. No match or ambiguous multiple matches → `city: null`.
- Load informational settings and towns independently. A settings failure uses the current informational defaults (`~30 мин`, `СОТ и ДНТ`, 200–350 ₽). A town-load failure uses `city: null`; never restore a hardcoded 500-ruble threshold. Log failures without secrets.
- Put `DeliveryInfoProvider` / `useDeliveryInfo` in the dish entity; the app-layer async wrapper fetches the value and supplies it. Keep root router composition thin, expose through public APIs and preserve FSD boundaries. Do not move entity state into `shared`.
- Context default is informational defaults plus `city: null`. Both website and Mini App cart components inherit the same runtime data; do not require a Telegram session to fetch public settings.
- Keep settings text as escaped plain text. Apply existing typography helpers and price formatting consistently.

### Captions and quotes
- Generate `outskirtsRange` once: different bounds → `200–350 ₽`; equal bounds → `200 ₽`, using existing nonbreaking-space conventions. `outskirtsFee` is the label followed by that range, e.g. `СОТ и ДНТ 200–350 ₽`.
- Preserve cooking text verbatim apart from the existing typography helper; keep the menu's current `готовим` suffix.
- Menu free-delivery badge and home free-delivery caption appear only when `city.freeFrom > 0`. Format the backend value; never retain `500` in runtime logic.
- When no threshold is available or it equals zero, use `Доставка еды` for the home hero's delivery-tile caption. In the home delivery section omit only the free-price claim and its adjoining separator; retain hours and outskirts information. Keep existing unrelated content blocks intact.
- Change `quoteDelivery` to accept the runtime `DeliveryInfo` as a second argument, while preserving its existing method/zone/subtotal input and `DeliveryQuote` result.
- Pickup → amount 0 and formatted zero price. Outskirts → amount null and runtime short range. City with a tariff → `townDeliveryPrice(city, subtotal)` and its formatted label. City with no tariff → amount null and `уточним`.
- Honor `freeFrom = 0` as never free by threshold. Below a positive threshold show the town's actual price rather than the previous `уточним` assumption.
- Cart delivery remains an estimate for Белореченск until checkout selects a town. Do not change cart food subtotal, delivery total presentation, checkout submission, promo adjustments or backend-authoritative pricing. Existing checkout town tariffs and order quotes remain authoritative.
- Remove outdated design-assumption comments and all imports of the old `delivery` constant. Do not add new banners to Mini App pages; only their existing shared cart estimates change.

### Freshness and failure behavior
- Saved settings and town prices become visible on a subsequent server fetch after the 60-second cache lifetime, following Next's existing stale/revalidation behavior. Already-open pages update on reload/navigation; no polling or live push is added.
- Do not claim a hard 60-second propagation deadline: static page/route caching may also revalidate. Integration QA must reload/poll through revalidation rather than assume a single immediate refresh.
- Invalid settings response data uses the informational fallback; validate text lengths, integer bounds and min/max ordering before formatting. Invalid city tariff data must not generate a free-delivery claim or a numeric estimate.

## Tests to write
Use production Playwright flow assertions, not logs alone. Use the existing configured backend; never start a backend/database or mock a successful live tariff source.

- Pure behavior checks in the QA flow: range formatting (different/equal/zero/max bounds), cooking/label typography, pickup/city/outskirts quotes, below/at/above threshold, zero threshold, missing/ambiguous city and invalid settings/tariff response handling.
- At 390/1440 px in light/dark: home hero/section, menu badges, mobile sheet and desktop cart all show the same configured facts; show actual city price below threshold and free delivery at threshold.
- Change settings and Белореченск's threshold only with explicit backend-write authorization. Capture originals and restore in `finally`. Poll fresh page loads through cache revalidation and verify settings/tariff changes propagate independently. Verify zero-threshold captions do not promise free delivery.
- Failure cases: informational defaults remain if delivery-settings fails; missing/unavailable city hides free-delivery claims and shows `уточним`. Exercise pure selector/format/quote helpers without replacing the configured backend with a fake service.
- Existing menu and checkout flows still pass with order-writing gates off; mobile cart animations, keyboard operation and checkout totals are preserved. Mini App shared data path is checked statically; authenticated device checks remain pending unless a valid session is supplied.
- Do not make settings/tariff writes without authorization or mark unexecuted live-edit scenarios as passed.

## API documentation
Consume backend task 91's GET contract and existing `/delivery-prices`; no endpoint changes. Cooking/outskirts fields are informational and do not change the order API.

## Done when
`npm run check`, `npm run build`, and production-build `delivery-settings.mjs`, `menu.mjs` and `checkout-and-contacts.mjs` flows pass using configured services (no order writes). Restore any authorized fixtures. Record authenticated Mini App/manual checks honestly. Then tick this task and commit per CONVENTIONS.

## Out of scope
Backend/admin implementation, new pricing rules, checkout or promo calculation changes, town selection before checkout, live updates in already-open pages, content-block redesign and unrelated visual changes.
