# Customer accounts and Telegram Mini App release

This is a staged release checklist, not authorization to deploy or run live writes. Keep all values below as placeholders in documentation. The backend [customer backlog](../../citypark-go/docs/tasks/CUSTOMER_ACCOUNTS.md), [API contract](../../citypark-go/docs/API.md), and website [tasks 35–40](tasks/PROGRESS.md) define the implemented behavior.

Implementation tasks: website [35](tasks/35-customer-account-auth.md), [36](tasks/36-customer-recovery-google.md), [37](tasks/37-customer-account-history.md), [38](tasks/38-telegram-mini-entry.md), [39](tasks/39-telegram-mini-history-payment.md), [40](tasks/40-customer-mini-release-checks.md); backend [66](../../citypark-go/docs/tasks/66-order-ownership-migration.md), [67](../../citypark-go/docs/tasks/67-order-contact-safety.md), [68](../../citypark-go/docs/tasks/68-customer-identities.md), [69](../../citypark-go/docs/tasks/69-customer-sessions.md), [70](../../citypark-go/docs/tasks/70-customer-email-tokens.md), [71](../../citypark-go/docs/tasks/71-verified-email-auth.md), [72](../../citypark-go/docs/tasks/72-customer-password-reset.md), [73](../../citypark-go/docs/tasks/73-google-authorization.md), [74](../../citypark-go/docs/tasks/74-google-completion.md), [75](../../citypark-go/docs/tasks/75-telegram-authentication.md), [76](../../citypark-go/docs/tasks/76-customer-account-merge.md), [77](../../citypark-go/docs/tasks/77-telegram-account-pairing.md), [78](../../citypark-go/docs/tasks/78-customer-linked-methods.md).

## Server configuration

| Variable | Where it is read | Release value |
|---|---|---|
| `CUSTOMER_AUTH_ENABLED` | backend `internal/config`, customer routes | `false` until approved smoke checks pass; then `true` |
| `CUSTOMER_PUBLIC_ORIGIN` | backend `internal/config`, verification links and Google callback validation | `https://<public-site-host>` |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS_MODE` | backend `internal/config` and mail transport | Owner-managed mail server settings; `starttls` or `tls`; never put passwords in logs |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_CALLBACK_URL` | backend `internal/config` and Google auth | Callback exactly `https://<public-site-host>/account/google/callback`, registered exactly in Google console |
| `CUSTOMER_TELEGRAM_BOT_TOKEN` | backend `internal/config` and Mini App initData validation | Dedicated customer bot token, separate from staff `TELEGRAM_TOKEN` and notification settings |
| `MENU_API_URL` | Next server `src/shared/config/env.ts` | Backend API URL ending in `/api/v1`; server only |
| `CUSTOMER_PUBLIC_ORIGIN` | Next server `src/shared/config/env.ts` | Same public HTTPS origin as backend, no path |
| `TELEGRAM_MINI_APP_URL` | Next server environment | Optional `https://t.me/<customer-bot>/<app>` return link |

Keep secrets in server secret storage, never in `NEXT_PUBLIC_*`, HTML, browser storage, screenshots or QA output. Customer and mini sessions use `Secure`, `HttpOnly`, `SameSite=Lax` cookies and require HTTPS on the public origin. The service worker excludes `/account` and `/mini` private routes. Private responses must carry `Cache-Control: private, no-store` (or stricter); exclude those paths from CDN and reverse-proxy caches. Check this with the production flows and inspect the deployed proxy as well.

Create a dedicated customer bot. In BotFather, configure its Mini App and menu button to open `https://<public-site-host>/mini`; preserve the staff bot independently. Google sign-in and linking start on the regular website, outside Telegram's embedded browser. Do not expose the bot entry until the final gate.

## Staged rollout and rollback

1. Back up the database and verify restoration instructions and owner access.
2. Apply additive backend migrations, then deploy the backend with `CUSTOMER_AUTH_ENABLED=false`. Run the backend suites with the configured test database (`go test -p 1 ./...`), including staff Bearer login/access, notifications, Moneta callbacks and iiko order payload tests. Record test run, commit and evidence link; live notifications, payment callbacks and iiko delivery remain owner-run checks.
3. Configure SMTP, Google and dedicated bot credentials in server secret storage; validate exact HTTPS origins and redirect URI. Never log tokens, codes, email contents or provider payloads; log sanitized error categories only.
4. Deploy the production website. Check public HTTPS, headers and CDN cache exclusions. With the site running at `BASE_URL` and backend at `MENU_API_URL`, run `npm run check`, `npm run build`, then `npm run qa:customer-release`. The runner requires both URLs and aggregates child failures. It does not start the backend.
5. Obtain owner approval for live smoke checks before setting `QA_CUSTOMER_RELEASE_APPROVED=1 QA_CUSTOMER_WRITES=1` and providing approved fixtures. Run registration/verification, sign-in/recovery, Google, two-account isolation, Telegram authentication/pairing and order/payment checks; record actual results. `QA_PLACE_ORDER=1` additionally allows order creation. Never use a production customer account as a QA fixture.
6. Enable `CUSTOMER_AUTH_ENABLED=true` only after smoke checks pass. Repeat read-only checks and complete the manual matrix. Expose the BotFather Mini App/menu-button entry last.

Backend regression evidence to attach from the approved suite run: staff sign-in and Bearer access in [`TestStaffAuthRegression`](../../citypark-go/internal/app/customer_linking_test.go) and [`TestRequireRoleAdminOnAdminRoute`](../../citypark-go/internal/auth/middleware_test.go); staff notification in [`TestNotifierPaid`](../../citypark-go/internal/app/app_test.go); Moneta callbacks in [`TestCallbackSuccess`](../../citypark-go/internal/order/callback_test.go) and [`TestVerifyCallback`](../../citypark-go/internal/moneta/moneta_test.go); iiko delivery in [`TestIikoOrdersSuccess`](../../citypark-go/internal/app/iiko_orders_test.go). These are suite references, not a claim that they ran during this website task. Separately, the owner must approve and record one live notification, one Moneta payment/callback, and the resulting iiko payload/receipt against the configured services.

For rollback, disable the customer feature and bot entry points, then investigate and redeploy the prior website/backend as appropriate. Migration 76 merges identity, order and address ownership; a schema rollback cannot automatically unmerge those records. Preserve a backup and plan manual reconciliation if needed.

## Acceptance evidence

| Gate | Evidence to record | Current state |
|---|---|---|
| Automated production build | Commit, build/check logs, `qa:customer-release` output, BASE_URL and backend version (no secrets) | Passed on 2026-10-30: `npm run check`, `npm run build`, and 10/10 read-only release checks; accessibility 56 states with no violations; responsive 14 routes × 12 widths with 12 counted Yandex script exceptions directed by the owner |
| Approved live accounts/providers | Owner approval, fixture IDs, sanitized result and timestamp | Pending |
| Backend staff and integration regressions | `go test -p 1 ./...` output and named test links; separate owner-run notification/Moneta/iiko outcomes | Pending |
| Real device and payment handoff | Matrix below, screenshots or recording with secrets removed | Pending |

Earlier findings in [task progress](tasks/PROGRESS.md) include private-IP image guard failures on `/` and `/news` (tasks 24–25) and backend-dependent QA flows (task 34). The mobile target-size findings from task 25 pass after the quick-action button gap change. The owner directed QA to ignore only the Yandex Maps script load error on `/contacts`; the responsive runner counts and prints those exceptions, while other errors still fail. `/news/frittata` belongs to the removed chef blog and is excluded from page sweeps. The release runner returns nonzero on every failed child check.

## Manual device matrix

All rows are **pending**. Enter the actual app and OS version, date, result, sanitized evidence link and owner after execution. Test both successful and failed provider/payment paths without recording tokens or full contact data.

| Device/version | Scenario | Date | Result | Evidence | Owner |
|---|---|---|---|---|---|
| Android / pending | Launch, raw auth expiry, reopen after cookie loss and interrupted network | — | Pending | — | — |
| Android / pending | Viewport, keyboard, safe areas, Back button, ordering/contact | — | Pending | — | — |
| Android / pending | History, pairing initiated and confirmed on both sides, Google handoff to regular browser | — | Pending | — | — |
| Android / pending | External payment success, failure, cancel; return, reopen and reload | — | Pending | — | — |
| iOS / pending | Launch, raw auth expiry, reopen after cookie loss and interrupted network | — | Pending | — | — |
| iOS / pending | Viewport, keyboard, safe areas, Back button, ordering/contact | — | Pending | — | — |
| iOS / pending | History, pairing initiated and confirmed on both sides, Google handoff to regular browser | — | Pending | — | — |
| iOS / pending | External payment success, failure, cancel; return, reopen and reload | — | Pending | — | — |
| Desktop / pending | Launch, raw auth expiry, reopen after cookie loss and interrupted network | — | Pending | — | — |
| Desktop / pending | Viewport, keyboard, safe areas, Back button, ordering/contact | — | Pending | — | — |
| Desktop / pending | History, pairing initiated and confirmed on both sides, Google handoff to regular browser | — | Pending | — | — |
| Desktop / pending | External payment success, failure, cancel; return, reopen and reload | — | Pending | — | — |
