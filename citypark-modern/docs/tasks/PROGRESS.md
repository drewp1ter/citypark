# Progress

Planned feature: [My addresses](../../../citypark-go/docs/tasks/MY_ADDRESSES.md). Tasks through 56 are complete; follow the remaining task queue. Follow the unchecked queue below.

## Homepage menu
- [x] [51 Homepage category switching and curated dishes](51-homepage-menu.md) [website50; backend92]
  - Check/build, pure resolver checks, the real-backend homepage-menu flow and existing menu/cart regression pass. Browser QA covers saved/manual/default selections at 390/1440px in both themes, exact dish order, switching without navigation or scroll changes, keyboard/focus, cart persistence and no page overflow/errors. Original homepage configuration restored; no orders or payments.

Take the first unchecked task. Tick it only when every "Done when" command passes. The numbers in brackets are the tasks it depends on.

## Backend integration
- [x] 01 Menu from the backend API

## Menu
- [x] 02 Dish description on the menu card [01]
- [x] 03 Cart sheet on the mobile menu [01]
- [x] 04 Quantity steppers in the desktop cart panel [03]
- [x] 05 Solid menu toolbar and category arrows

## Forms
- [x] 06 Dropdown Select on React Aria
- [x] 07 Clear button for text inputs

## Home page
- [x] 08 Free rooms count on the home page
- [x] 09 Free rooms from the backend [08] (needs citypark-go task 41)

## Brand
- [x] 10 Brand logo in the header and footer

## Rooms page
- [x] 11 Room availability from the backend [09] (needs citypark-go task 41)

## Admin panel support (see ../docs/adr/0001-admin-panel.md)
- [x] 12 Remove the chef's blog
- [x] 13 Room prices from one place
- [x] 14 Hotel prices as numbers
- [x] 15 Content blocks, part 1: the entity and the home page [13, 14]
- [x] 16 Content blocks, part 2: the hotel and rooms pages [15]
  - `blocks.data.ts` has 101 entries (46 home + 55 hotel). The content code now lives in `entities/content`.
- [x] 17 Rooms from the backend [13, 15] (needs citypark-go tasks 45, 48)
- [x] 18 News from the backend [12, 17] (needs citypark-go task 49)
- [x] 19 Site settings from the backend [17] (needs citypark-go task 51)
- [x] 20 Content blocks and hotel prices from the backend [16, 17] (needs citypark-go tasks 50, 52, 54)

## Content from the old site (/workspace/citypark_frontend)
- [x] 21 Legal pages from the old site
  - Owner: check the offer is current (domain cityparkvip.ru, payment methods) and send the classification certificate if it changed.
- [x] 22 3D tours from the old site
  - `public/tour` is 46M (`du -sh`): the owner can decide later whether to move it to a CDN.
- [x] 23 Full-size room photos and hotel texts from the old site [20] (needs citypark-go task 55)
- [x] 24 Short room text on the rooms page [17, 23] (needs citypark-go task 56)
  - No other place rendered `descriptionHtml` on a card. Checked on a build at 1440 px (light, dark) and 390 px; first row is about 455 px tall. In `qa:responsive` the 400s on `/` and `/news` come from Next's private-IP image guard for backend uploads (`192.168.4.15`), not from this task.
- [x] 25 Room pages with a gallery [17, 22, 24] (needs citypark-go task 57)
  - `qa:a11y` still reports `target-size` on the `/hotel` list link (obscured by the mobile quick bar, same as a `/news` item) and landmark warnings on `/news/frittata`; none on `/rooms/vip`. The 400s in `qa:responsive` are the private-IP image guard on `/` and `/news`.

## Payment
- [x] 26 Checkout sends the order to the backend and opens the payment page [01] (needs citypark-go task 58)
  - Owner: run `QA_PLACE_ORDER=1 MENU_API_URL=... node scripts/qa/flows/checkout-and-contacts.mjs` (creates a real order; the URL host must be the Moneta assistant host) and place one manual order on a local backend with `MONETA_TEST_MODE=true`: the Moneta demo form must open, not a signature error. Not run here: both POST to the backend.
- [x] 28 Checkout sends the delivery time as its own field [26] (needs citypark-go task 61)
  - Owner: with a local citypark-go that has task 61, place a «Ко времени» order. In Telegram it has the «Ко времени» line, and the comment has only the customer's text. Not run here: it POSTs to the backend.

## Cleanup
- [x] 27 Remove the table booking form

## Working hours
- [x] 29 Working hours from the backend [28] (needs citypark-go task 63)
  - Owner: the backend at `MENU_API_URL` (192.168.4.15) is older than task 63 (`GET /working-hours` → 404), so here the pages showed the built-in `9:00–22:00`. Deploy citypark-go with task 63, then check that changing the hours in admin shows on `/menu`, `/`, `/contacts`, `/checkout` within 60 s and that «Ко времени» `08:30` gives «Мы работаем с 9:00 до 22:00» under the time field. The `min`/`max` of the time input make the browser block out-of-range times before the server sees them.

## Map
- [x] 30 Yandex map on the home and contacts pages [29]
  - Owner: `/` and `/contacts` are static (revalidate 1m), so `YANDEX_MAPS_API_KEY` is read at build time and refreshed on revalidation: rebuild after changing the key. Coordinates `39.8749086, 44.7596453` come from OpenStreetMap. A dev server behind nginx needs `Upgrade`/`Connection` headers for `/_next/hmr`, or the page does not hydrate.

## Online payment only
- [x] 31 Stop sending paymentType [26] (needs citypark-go task 65)
  - Owner: the checkout QA flow ran without `QA_PLACE_ORDER=1` (a real order POSTs to the backend), so the order without `paymentType` was not placed here. Place one order on a backend with task 65 and check that the Moneta page opens.

## Address suggestions
- [x] 32 Address suggestions on checkout [26, 28]
  - Deviation: `AddressField` lives in `src/pages/checkout/ui/address-field.tsx`, not in a `features/address-suggest` slice. Steiger rejects a slice with one consumer (`fsd/insignificant-slice`) and the conventions forbid overrides.

## Delivery price
- [x] 33 Free delivery from the town's threshold on checkout [26]

## PWA
- [x] 34 Installable site with an offline page
  - Owner: the `menu` and `checkout-and-contacts` flows need the backend on :3000, which was down, so `qa:flows` stopped there; the other flows and `pwa.mjs` pass. Do the manual install check on HTTPS (Android Chrome, iOS Safari). Lighthouse was not run.

## Customer accounts and Telegram Mini App
- [x] [35 Customer sign-in and verification](35-customer-account-auth.md) [34; backend71]
  - Live checks passed: verification, sign-in, logout, expired-session checkout, guest retry and account-history isolation. Test orders 5 and 6 are unpaid and pending; retry reused order 6.
- [x] [36 Recovery and Google sign-in](36-customer-recovery-google.md) [35; backend72,74]
  - `npm run check`, `npm run build`, and the named production-build flow pass. Write scenarios were gated because backend POSTs need owner approval. Owner check: with configured Google credentials and approved test data, run `QA_CUSTOMER_WRITES=1 QA_GOOGLE_START=1 QA_CUSTOMER_EMAIL=... node scripts/qa/flows/customer-recovery-google.mjs`, then complete a real Google login and a password reset from emailed links.
- [x] [37 Account, linked methods and order history](37-customer-account-history.md) [36; backend78]
  - `npm run check`, `npm run build`, and the read-only production flow pass. Live account writes, order fixtures, and two-sided pairing are gated by `QA_CUSTOMER_WRITES=1`; they were not run here.
- [x] [38 Telegram Mini App entry and ordering](38-telegram-mini-entry.md) [37; backend75]
  - `npm run check`, `npm run build`, and the production-build `telegram-mini.mjs` flow pass. Existing menu and checkout flows pass without order writes. Owner check: with an approved fresh Telegram launch, run `QA_CUSTOMER_WRITES=1 QA_TELEGRAM_INIT_DATA=... node scripts/qa/flows/telegram-mini.mjs` against the configured backend; a real order/payment and device checks remain for task 40.
- [x] [39 Mini App pairing, history and payment return](39-telegram-mini-history-payment.md) [38]
  - `npm run check`, `npm run build`, the production-build `telegram-pairing-payment.mjs` flow, and the existing checkout regression flow passed. Live pairing needs `QA_CUSTOMER_WRITES=1` plus fresh mini and website sessions; provider payment and device handoff remain manual checks for task 40.
- [x] [40 Customer accounts and Mini App release checks](40-customer-mini-release-checks.md) [39; backend78]
  - `npm run check`, `npm run build`, and production-build `npm run qa:customer-release` pass (10/10 automated checks). The responsive sweep reports 12 ignored Yandex script failures as directed by the owner. `/news/frittata` is an obsolete chef-blog route removed from page sweeps. Approved live writes and device checks remain pending in the release runbook.

- [x] [41 Category descriptions](41-category-descriptions.md) [40; backend80]
  - `npm run check` and `npm run build` pass. Category text becomes available after backend task 80 is deployed and iiko sync runs.

## Dish ordering blocks
- [x] [42 Visible blocked dishes and checkout protection](42-dish-ordering-blocks.md) [41] (needs citypark-go task 81)
  - Keep blocked dishes visible with normal content and no additional availability text or badges. Hide add/increment controls entirely rather than disabling them.
  - Keep blocked items already in persisted carts visible and removable, without additional availability text or increment controls. Never silently omit them from the submitted order.
  - Prevent checkout until blocked items are removed; attempted checkout uses the existing form-error mechanism to explain the required removal.
  - Refresh availability on cart/checkout entry, window focus and every 60 seconds while visible. Check fresh availability on submission and handle backend rejection authoritatively.
  - Test visible dishes without extra text or controls, persisted carts, removal, checkout rejection, unblock recovery and mobile/desktop behavior; run required frontend checks and relevant QA flows using existing services.
  - Check/build, existing menu and checkout flows, and the blocked-dish flow pass. The targeted flow covers 390/1440 px and both themes using a temporary admin-owned fixture, then restores its state; no orders are placed. Authenticated Mini App device/payment checks remain manual; it uses the same ordering components and server validation.

## Promo codes

Implementation plan: [Promo codes](../../../citypark-go/docs/tasks/PROMOCODES.md). Planned only; existing blockers remain unchanged.

- [x] [43 Promo codes in shared checkout](43-promo-checkout.md) [website42; backend89]
  - Added server-only authoritative quotes, apply/replace/remove, current-context pricing guards, kopeck totals, delivery-fee warnings and 409 quote refresh requiring another submission. Applied codes survive payment-return reloads; ownership/payment helpers are preserved. Check/build and production-build promo-checkout, checkout-and-contacts and telegram-mini regressions pass. Promo QA covers 390/1440 px, both themes, town/cart changes, overlap/cap, invalid/expired/minimum codes, delayed responses, failed transport and authorized validation-only 409s. Temporary campaigns are disabled; no orders, payments or iiko writes were made.
  - Owner will run authenticated Mini App checks (explicit handoff, 2026-10-31): supply an existing QA_MINI_SESSION and run QA_PROMO_WRITES=1 MENU_API_URL=<configured backend> node scripts/qa/flows/promo-checkout.mjs against the production build. Existing account-session checks can additionally use QA_CUSTOMER_SESSION. No provider authentication or payment is performed by this flow.
- [x] [44 Promo history and cross-surface QA](44-promo-history-qa.md) [website43; admin28]
  - Added one shared saved-snapshot breakdown for website/Mini App details and kopeck-accurate promo history totals; no-code presentation is preserved. Check/build, promo-history (guest pickup/delivery plus staff website inspection at 390/1440 px in both themes), promo-checkout, checkout-and-contacts, telegram-pairing-payment and unauthenticated customer-history checks pass. Temporary campaigns are disabled; QA orders 29–34 remain unpaid/pending. No provider payment, callback, iiko dispatch or staff message was sent.
  - Authenticated customer/Mini App history, second-account isolation and device/provider/restaurant acceptance were not run without supplied sessions; they remain release requirements, not claimed passes. See [promo release checklist](../PROMO_RELEASE.md) for actual results and rerun instructions.

- [x] [45 Promo-code visibility and automatic recalculation](45-promo-switch-and-debounce.md) [Backend task 90 and website task 44]
  - Check, build, checkout-and-contacts and promo-checkout flows pass. Website QA covers 390/1440 px, both themes, switch changes, debounce, cart/town changes, stale responses, transport errors and an authorized validation-only 409. Temporary campaigns were disabled and the original switch state restored. Authenticated Mini App checks remain pending a supplied QA_MINI_SESSION; run the promo flow with that session and approved temporary campaign/settings writes.

## Checkout location

- [x] [46 Automatic checkout location with DaData](46-checkout-geolocation.md) [website45]
  - Website and Telegram Mini App: automatic location request, editable address autofill, supported-town matching and manual fallback. Reuses DaData credentials; no backend or admin task required.
  - `npm run check`, `npm run build`, `checkout-geolocation.mjs`, `checkout-and-contacts.mjs`, `promo-checkout.mjs` with temporary campaigns, and `telegram-mini.mjs` pass. Campaigns were disabled by QA cleanup; no orders or payments were created. Authenticated Telegram Mini App geolocation and promo checks remain pending supplied session/device access.

## Motion

- [x] [47 CSS animations and hover effects](47-css-motion.md) [46]
  - Check and production build pass. Chromium checks of the generated production CSS pass in eight combinations of light/dark themes, mouse/touch input and normal/reduced motion, including disabled controls and dialog entry.

- [x] [48 Mobile cart opening and closing animations](48-mobile-cart-motion.md) [47]
  - Check, production build and cart-motion browser flow pass in both themes with normal/reduced motion. Verified animated opening/closing, close button, Escape, backdrop, repeated opening, focus restoration and checkout navigation; no backend writes.

## Delivery settings

- [x] [49 Delivery information from the backend](49-delivery-settings.md) [website48; backend91]
  - Use editable delivery facts and Белореченск’s existing tariff/threshold for captions and cart estimates; admin30 provides the editor.
  - Check/build and production delivery-settings, menu, checkout-and-contacts and cart-motion flows pass. Verified real facts at 390/1440 px in both themes, tariff estimates below/at threshold, independent settings/tariff cache refresh, zero threshold, pure fallback/validation cases, and restoration propagation. Authorized temporary settings/tariff writes were restored; no orders or payments created. Authenticated Mini App/device checks remain manual; shared provider/cart path checked statically.
  - Deviation: app wrapper lives in `src/app/delivery` rather than `src/app/providers`; Steiger rejects `providers` under `fsd/segments-by-purpose`. No rule overrides added.

## Local diagnostics

- [x] [50 Local DaData geolocation diagnostics](50-dadata-geolocation-diagnostics.md)
  - Opt-in development terminal logs show provider responses, HTTP status, normalized town candidates and rejection/failure reasons. API responses and selection rules are preserved; credentials are redacted and production logging is disabled. `npm run check`, `npm run build` and an isolated controlled-provider/handler harness pass, covering successful matching, unsupported settlement/region, duplicates, empty/malformed responses, provider/transport and delivery-list failures, logging guards and credential redaction. No live provider requests or backend writes were performed. Build emits existing dynamic-rendering diagnostics for `/` and `/menu` and completes successfully.

## Monitoring

- [x] [52 Sentry website monitoring](52-sentry.md) [51; backend93 for distributed verification]
  - Check, production build, monitoring QA and real-backend menu flow pass. SDK setup covers browser, Node and Edge initialization, request/navigation hooks, route/global error boundaries and strict privacy filters. Public source-map uploads succeeded and `.next/static` has no maps.
  - Fresh production-build browser/server errors (CITYPARK-MODERN-6 and -7) resolve to original TypeScript source with code context. Trace 8409f2e72c634806a9182b324130a770 links website→HTTP→Go and both errors. Temporary verification routes removed; owner restarted the clean backend and its former trigger returns 404. Final clean source-map upload passes; no token in public bundles. Next dev generated its documented AGENTS.md guidance block; retained it. Production rollout is separate and requires the normal owner-managed release/restart process; see docs/SENTRY.md.

- [x] [53 Address autofill Sentry diagnostics](53-autofill-sentry.md) [52]
  - Check/build, privacy/provider QA and production geolocation/browser failure flows pass. Development runs also verify effect replay, single capture/request and manual takeover/unmount suppression. Source maps uploaded; Sentry MCP confirms browser CITYPARK-MODERN-A and server CITYPARK-MODERN-9 with original TypeScript frames and allowlisted diagnostics. Coordinates, addresses and provider payloads remain excluded. No orders or payments; production deployment and native Telegram device permission checks remain separate.

## Payment retry

- [x] [54 Payment retry from order history](54-payment-retry.md) [website53; backend95]
  - Added shared payment retry on website and Mini App history/details, authoritative backend eligibility, isolated surface sessions, signed history origin and cart-preserving payment returns.
  - Check (typecheck/ESLint/Steiger), fresh production build, isolated action/crypto/status/return-routing harness, production payment-retry flow (390/1440 px, light/dark) and applicable Telegram payment regressions pass against the available backend. No backend writes performed.
  - Owner explicitly accepted the remaining manual checks and requested completion/commit (2026-11-05). Authenticated retry/returns with existing owned orders, stale backend rejection, provider success/failure, and native Telegram/external-browser checks are handed off to the owner; they are not claimed as automated passes. QA setup is documented in scripts/qa/flows/payment-retry.mjs.

## My addresses

Implementation plan: [My addresses](../../../citypark-go/docs/tasks/MY_ADDRESSES.md). Backend task 97 must pass before website 55; all backend tasks through 98 must pass before management integration in 56.

- [x] [55 Saved-address model and guest storage](55-guest-address-store.md) [website54; backend97]
  - Versioned guest storage, backend-aligned Unicode validation, duplicate/default rules, revision-aware import cleanup and shared checkout formatting complete. Address-store harness, check and production build pass (2026-11-05). Owner approved a temporary address-only `fsd/insignificant-slice` exception; remove once later tasks add independent consumers. Existing dynamic-rendering build diagnostics remain; no backend writes.
- [x] [55b Address API adapters and reusable fields](55b-address-api-and-fields.md) [website55; backend97]
  - Explicit web/Mini App server adapters, ownership guards, validated envelopes and safe errors complete. Shared controlled fields retain old towns and distinguish manual edits from selection; checkout uses the extracted suggestion control. Address API/guest-store harnesses, check, production build, keyboard suggestion checks at 390/1440 px and production geolocation flow pass (2026-11-05). Corrected the geolocation QA selector to ignore empty native options. No live backend mutations; authenticated Mini App/device geolocation remains pending supplied access.
- [x] [56 My addresses management screens](56-my-addresses-management.md) [website55b; backend98]
  - Shared guest/account address management, web/Mini App routes and navigation complete; removed the temporary address-only Steiger exception. Isolated management/API/guest-store harness, check, production build and guest browser flows (390/1440 px, both themes) pass.
  - Owner explicitly accepted manual verification (2026-11-05). Authenticated web/second-account/Mini App checks, legacy/unavailable fixtures, expired/changed-account browser acceptance and native Telegram checks are handed off to the owner, not claimed as passes. Required sessions and authorized mutation setup are documented in scripts/qa/flows/my-addresses.mjs. No backend mutations performed.
- [x] [57 Saved addresses and opt-in saving at checkout](57-checkout-saved-addresses.md) [website56; backend98]
  - Shared website/Mini App selection, one-time default priority, controlled address drafts and explicit guest/account save-before-order complete. Failed submissions preserve fields and save intent; ownership/session checks and isolated save errors prevent unintended account writes or order dispatch.
  - Checkout-address/guest-store/payment-retry harnesses, check, production build, expanded my-addresses (390/1440 px, both themes), checkout-geolocation, checkout-and-contacts and payment-retry flows pass (2026-11-05). Fixed native form-reset draft loss and synchronous geolocation-denial status race. Existing dynamic-rendering build diagnostics remain.
  - Authenticated web/second-account/Mini App, live address mutations and native Telegram/provider checks remain owner acceptance with supplied sessions/device access; they are not claimed as passes. No live orders, payments or account-address writes performed; browser tests save only disposable guest storage and submit invalid contact fields.
- [x] [58 Import guest addresses after login](58-import-guest-addresses.md) [website57; backend98]
  - Implementation ready (2026-11-05): consent prompt/manual trigger, per-surface coordinator, account/revision dismissal, sequential default-first import, guarded revision cleanup, partial-failure retries and current-session checks. Layout bridges recheck persistent navigation, cookie-derived account changes and restored/focused tabs; old contexts discard responses and queued work.
  - Isolated import/entity harnesses, check and production build pass. Expanded production my-addresses guest/anonymous/invalid-session checks pass at 390/1440 px in both themes; geolocation and payment-retry regressions pass. No live account-address writes, authentication proofs, orders or payments performed. Session/consent/import mutation setup and cleanup are documented in the my-addresses QA header. Server-side authentication in the bridges makes public layouts dynamic; existing content-fetch caches remain.
  - Owner explicitly accepted the manual handoff (2026-11-05). Real authenticated web/Mini App import, password/Google return-to-page and native Telegram relaunch/import/device acceptance are handed off to the owner and remain pending; they are not claimed as passes. Required session/consent setup is documented in scripts/qa/flows/my-addresses.mjs.
- [x] [59 Saved-address integration acceptance](59-my-addresses-acceptance.md) [website55–58 including55b; backend96–98 including96b/96c]
  - Automated gates pass (2026-11-05) (6 harnesses, check, build, guest my-addresses incl. new privacy and 44 px stages, menu, checkout-geolocation, payment-retry; backend address tests). Matrix, results and owner checklist in docs/MY_ADDRESSES_RELEASE.md. Authenticated web/second-account/Mini App, legacy fixtures, login return paths and native Telegram acceptance remain unrun; waiting for owner sessions or explicit handoff acceptance. customer-auth/telegram-mini flows not run.
  - Authorized web run (owner-approved 2026-11-05, two disposable dev-backend accounts): real-session persistence/default and import stages pass at 390/1440 px in both themes; accounts ended with 0 addresses and were logged out. Still unrun: Mini App, second-account isolation assertions, legacy fixtures, login return paths, native Telegram, deployed API-vs-docs check.
  - Owner accepted the manual handoff (2026-11-05): "i'll check mini manually, let's commit". Mini App, second-account isolation, legacy fixtures, login return paths, native Telegram and the deployed API-vs-docs comparison are handed off to the owner and remain pending; they are not claimed as passes. Checklist in docs/MY_ADDRESSES_RELEASE.md.
- [x] [60 Payment return pages use the UUID transaction id](60-moneta-uuid-return.md) [backend99]
  - Signed payment cookie stores the UUID taken from `redirectUrl`; return pages match it instead of the order number (needs citypark-go task 99; deploy together).
  - Completed (2026-11-05): UUID-bound signed payment context, creation/retry URL parsing, and UUID-preserving Mini success/failure forwarding; history keeps numeric order routes. Both harnesses, npm run check and production build pass. After the owner started the configured backend, payment-retry (390/1440 px, both themes) and checkout-and-contacts browser flows pass against a fresh production server. Authenticated owned-order/session checks and native Telegram/MONETA handoff remain manual acceptance; no live orders or payments created.

## Search and Alice discovery
- [x] [61 Indexing foundations](61-indexing-foundations.md) [60]

Remaining priorities and deployment acceptance: [Search discovery](../SEARCH_DISCOVERY.md).
  - Canonicals, Russian page descriptions, production metadata base, opt-in indexing and private-layout noindex implemented. Check/build pass; existing dynamic-rendering diagnostics remain. Deployed HTTP acceptance remains pending.

- [x] [62 Public sitemap](62-public-sitemap.md) [61]
  - Complete public sitemap and robots reference implemented using shared production origin. Harness, check, build and production HTTP checks pass: valid XML and 91 exact URLs from live backend lists. Backend failures propagate; no backend writes. Public deployment acceptance remains pending.

- [x] [63 Public server HTML audit](63-server-html-audit.md) [62]
  - Read-only production HTTP audit, check and build pass. Contacts/hours, 5 room details and 24 linked categories with 246 dishes verified from parsed server HTML, excluding scripts/styles/templates. No rendering fixes needed. Query-category canonical and fallback freshness limitations documented in SEARCH_DISCOVERY.md; public deployment acceptance remains pending.

- [x] [64 Hotel and restaurant structured data](64-business-structured-data.md) [63]
  - Server-rendered Hotel/Restaurant graph on contacts uses the same settings/hours as visible facts, stable canonical IDs and script-safe serialization. Isolated/security harness, check/build, live-backend HTML agreement and task 63 content regression pass. Existing dynamic-rendering diagnostics and deployed-site acceptance remain documented; no backend writes.

- [x] [65 Customer questions and content gaps](65-customer-questions.md) [64]
  - Five server-rendered contact FAQ answers use existing settings/hours and reference parking copy. Source review and eight owner follow-up topics documented in CUSTOMER_QUESTIONS.md. Check/build, raw HTML, 390/1440px light/dark screenshots, keyboard/no-overflow/phone-wrap and existing JSON-LD HTTP checks pass. No backend writes; deployment/indexing acceptance remains pending.

- [x] [66 Yandex Webmaster runbook](66-yandex-webmaster-runbook.md) [62; preparation through 65]
  - Owner runbook covers production readiness, genuine HTML verification file, sitemap submission, indexing troubleshooting and Alice visibility. Official sources and local links/configuration reviewed; check/build and diff checks pass. Existing dynamic-rendering build diagnostics remain. Registration, verification artifact, deployment and Webmaster submissions remain pending.

- [67 Production search acceptance](67-production-search-acceptance.md) — **Skipped for now by owner (2026-11-05)** [64–66; production deployment]
  - Blocked (2026-11-05, 22:12 UTC): cityparkvip.ru still serves the previous application's identical HTML shell at /, /contacts, /robots.txt and /sitemap.xml (HTTP 200 text/html). New frontend deployment is required before acceptance. Existing legacy Yandex verification meta tag needs continuity review; account status is unknown. Evidence in SEARCH_DISCOVERY.md. No deployment, account operations or backend writes performed.

- [x] [68 Page-specific Open Graph previews](68-open-graph.md) [61]
  - Complete Russian Open Graph/Twitter cards for public static pages and room/news details. Existing photos with public fallback for HTTP, SVG or missing images; HTTP backend news photo was unreachable from the crawler. Check, build and production crawler checks pass for ten public pages and room/news details, including query canonical and image responses. External social previews remain deployment-dependent; no backend writes. Missing-record/private-page browser regression checks were not run.
