# Task 65: Customer questions and content gaps

## Goal
Provide concise, crawlable answers to verified customer questions and record questions needing owner facts.

## Depends on
63 and 64.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/contacts/ui/contacts-page.tsx`, `src/entities/content/model/blocks.data.ts`, `docs/SEARCH_DISCOVERY.md`.

## Files to create
- `src/pages/contacts/ui/contacts-questions.tsx`: server-rendered FAQ using existing cards, links and tokens.
- `docs/CUSTOMER_QUESTIONS.md`: source review, approved facts and unresolved owner questions.
- `scripts/qa/customer-questions.mjs`: raw HTML and browser acceptance.

## Files to change
ContactsPage includes the FAQ using its existing fetched settings/hours. Update task progress and search release notes.

## Spec
Add an always-visible section on /contacts with id questions and heading «Частые вопросы». Five questions: “Где находится «Сити Парк»?”, «Есть ли парковка?», «В какое время работают ресторан и доставка?», «Где посмотреть номера, цены и правила проживания?», «Где посмотреть меню доставки и цены на блюда?». Address, hours and hotel/restaurant phone links use the same settings/hours as the page. Parking answer repeats the exact existing reference copy: «Для гостей — бесплатная парковка у здания.». Link to existing /rooms, /hotel and /menu pages; do not duplicate mutable hotel rules or delivery fees. Exact final Russian UI copy is recorded in ContactsQuestions. No new backend calls, CMS fields, structured-data types or API changes. Retain existing fallbacks. Review existing hotel/delivery content and list missing facts for owner follow-up, without inventing answers.

## Tests to write
Raw HTML: FAQ contains all five questions, current backend address/hours/phone values and correct links after removing scripts/styles/templates. Browser: FAQ and answers visible at 390/1440px in light/dark themes; no horizontal overflow; links focusable by keyboard; telephone links stay on one line; capture screenshots. Existing task 64 structured-data HTTP acceptance remains valid.

## API documentation
No API changes.

## Done when
`npm run check`, `npm run build`, `node scripts/qa/customer-questions.mjs` and `node scripts/qa/business-json-ld-http.mjs` pass against a production server (BASE_URL defaults to http://localhost:3112). Review screenshots. Source review and owner questions are recorded in CUSTOMER_QUESTIONS.md.

## Out of scope
Publishing unresolved policies, changing existing hotel/legal copy, refund/availability guarantees, backend mutations and Webmaster registration.

## Results
Production build and all named HTTP/browser acceptance checks passed. Screenshots at 390/1440px in both themes were reviewed; phone wrapping was corrected and checked. Source facts and eight owner follow-up topics are documented in CUSTOMER_QUESTIONS.md. Existing dynamic-rendering diagnostics remain during build; deployment/indexing checks are pending.
