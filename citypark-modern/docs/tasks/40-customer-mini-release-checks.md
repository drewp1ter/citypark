# Task 40: Customer accounts and Mini App release checks

## Goal
Collect production-build regression flows and a concrete staged release runbook
for website accounts and the Telegram Mini App.

## Depends on
39; backend78.

## Read first
`docs/tasks/CONVENTIONS.md`, `../citypark-go/docs/tasks/CUSTOMER_ACCOUNTS.md`, `package.json`, `scripts/qa/flows/telegram-pairing-payment.mjs`.

## Files to create
- `docs/CUSTOMER_ACCOUNTS_RELEASE.md`
- `scripts/qa/customer-release.mjs`

## Files to change
- `scripts/qa/flows/customer-auth.mjs`, `customer-recovery-google.mjs`, `customer-history.mjs`, `telegram-mini.mjs`, `telegram-pairing-payment.mjs`: final integrated scenarios and explicit skips.
- `package.json`: qa:customer-release command.
- `.env.example`, `README.md`: frontend public-origin/server API setup and link to runbook.

## Spec
- This task assembles checks and documentation, not deployment authorization.
  Do not publish bot/site, configure real credentials, create orders, send emails
  or pay without the existing approval process. Do not weaken checks to mark done.
- Runbook must inventory backend variables CUSTOMER_AUTH_ENABLED,
  CUSTOMER_PUBLIC_ORIGIN, SMTP_HOST/PORT/USERNAME/PASSWORD/FROM/TLS_MODE,
  GOOGLE_CLIENT_ID/SECRET/CALLBACK_URL, CUSTOMER_TELEGRAM_BOT_TOKEN; frontend
  MENU_API_URL and its server-only public origin. Explain where each is read;
  keep all secrets server-side, use placeholders only. Staff TELEGRAM_* stays
  distinct. Document cookie HTTPS requirements and private-cache exclusions.
- Explain dedicated bot creation and BotFather Mini App/menu-button entry at
  public HTTPS /mini. Google authorized redirect must exactly match the regular
  website callback. Google sign-in/linking remains outside Telegram embedded UI.
- Stage release: database backup -> additive backend migrations -> backend with
  customer feature gated off -> credential config -> frontend deployment ->
  owner-approved smoke checks -> enable customer auth -> expose bot entry last.
  Record migration76 merge data cannot be undone by schema rollback; disable
  entry points for rollback rather than promising automatic account unmerge.
- Add manual matrix rows for Telegram Android,iOS,Desktop: launch/auth expiry,
  viewport/keyboard/safe areas/back, ordering/contact, history/pairing both sides,
  Google handoff, external payment success/failure/cancel, return/reopen/reload,
  cookie loss and interrupted network. Columns device/version,date,result,
  evidence,owner. Leave unrun checks explicitly pending; never claim execution.
- Runner invokes relevant production-build customer flows and existing menu,
  checkout,keyboard,a11y,responsive checks, prints expected/actual and aggregates
  failures with nonzero exit. Preserve current known unrelated findings separately
  with links/evidence; do not mask newly introduced failures. No implicit backend
  writes: require owner approval and QA_CUSTOMER_WRITES=1 for write paths; absent
  flag prints clear pending checks, not success for unexecuted scenarios.
- Document staff Bearer login/access, notifications, Moneta callbacks and iiko
  payload regression evidence from backend suites; live notification/payment/iiko
  checks are explicit owner-run actions. Log only sanitized error categories.
- Acceptance evidence must distinguish automated checks, approved live checks and
  manual device checks. Preparing the runbook may complete this task; production
  enablement requires its pending smoke/device gates to be satisfied separately.
- No new product UI or authentication implementation belongs in this task.

## Tests to write
- `customer-release.mjs`: production server/backend reachable; all existing
  customer flows run in approved mode; missing required env fails with actionable
  message; a failing child causes nonzero exit; absent write flag reports pending
  live checks. Never starts or writes to backend implicitly.
- Run `npm run qa:customer-release` plus check/build; review each emitted result.
- Confirm runbook paths/commands/env names against implemented tasks; verify
  placeholders contain no real secret and manual matrix remains honest.
No new unit tests for documentation; validate runner's exit behavior and flows.

## API documentation
No backend endpoint changes. Link the existing API contract and each customer task; do not duplicate secret-bearing examples.

## Done when
Run `npm run check` without errors/warnings and `npm run build`. Run the
named Playwright flow against a production build on :3111, with BASE_URL
and MENU_API_URL configured. Read output and fix discrepancies. Never
start the backend or mock an unavailable backend; ask the owner to start it.
Real backend writes/orders/payments require the project's approval process.
Tick only after checks pass; commit as `task 40: Customer accounts and Mini App release checks`.


## Out of scope
Actual deployment/BotFather changes, unapproved real orders/payments/emails, unrelated hotel/staff features.
