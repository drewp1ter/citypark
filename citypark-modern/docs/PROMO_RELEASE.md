# Promo release checks

Task 44 verification, 2026-10-31. This records implementation checks, not campaign activation approval.
The contract is [PROMOCODES.md](../../citypark-go/docs/tasks/PROMOCODES.md); the restaurant acceptance
procedure is [IIKO_PROMOCODES.md](../../citypark-go/docs/tasks/IIKO_PROMOCODES.md).
Keep production campaigns disabled until every release acceptance below has a recorded pass.

## Actual results

| Check | Result |
|---|---|
| Website `npm run check`, `npm run build` | Pass |
| `promo-history.mjs` against the existing backend and production website | Pass for guest pickup/delivery creation and staff inspection of website details, 390/1440 px, light/dark |
| Saved snapshots after campaign percentage/cap edit and disabling | Pass: original subtotal, code, discounted food, delivery, payable amount and per-line discounts unchanged |
| Kopeck arithmetic | Pass: 0.01 ₽ cap; original line subtotals sum to header subtotal; line discounts sum to header discount; food = subtotal − discount; payable = food + delivery |
| MONETA URL amount | Pass: generated `MNT_AMOUNT` equals saved discounted food plus saved delivery for pickup and paid delivery. URL inspected only; provider never opened |
| Legacy/no-code detail | Pass: existing wording, whole-ruble formatting and absence of promo rows preserved |
| `promo-checkout.mjs` | Pass: guest checkout, 390/1440 px, light/dark; current authoritative quotes, codes, cart/town changes, capped discounts, races and transport errors. Validation-only 409 branch not rerun (already recorded in task 43) |
| `checkout-and-contacts.mjs` | Pass: validation, pickup, delivery, failed return, preserved form/cart and existing contact flow; real order submission disabled |
| `telegram-pairing-payment.mjs` | Pass: unauthenticated gate, forged payment return, reload and cart preservation, both themes; authenticated branches not run |
| `customer-history.mjs` | Pass: unauthenticated redirects, private/no-store headers, malformed ID, storage and layout at both widths/themes; live account branches not run |
| Actual customer/Mini App history and cross-account ownership | Pending: no `QA_CUSTOMER_SESSION`, `QA_MINI_SESSION` or distinct `QA_CUSTOMER_SESSION_2` supplied. Staff inspection is not customer authentication or ownership acceptance |

The owner authorized temporary QA data. This task created unpaid guest orders 29–34 (including fixtures
from early QA failures). All remain pending. All temporary `QHISTORY_` and checkout campaigns were disabled.
The final creation/edit/disable run used orders 33 (pickup) and 34 (delivery). No payment callback, charge,
provider authentication, iiko dispatch or staff message was triggered. No backend/database was started.
Screenshots were inspected for the mobile dark promo breakdown; automated layout checks cover both widths/themes.
The existing date formatter displays timestamp dates incorrectly; that unrelated behavior was not changed.

## Repeatable automated checks

Use the configured backend from `.env`, never a replacement service. Start a production website on `BASE_URL`
(default `http://localhost:3111`). `promo-history.mjs` always signs in with the existing QA staff credentials
(`QA_ADMIN_EMAIL`/`QA_ADMIN_PASSWORD`, development defaults if omitted) to inspect saved snapshots.

```bash
set -a
source .env
set +a
npm run check
npm run build
# In another terminal: npm start -- -p 3111
QA_PROMO_ORDER_IDS=33,34 node scripts/qa/flows/promo-history.mjs
node scripts/qa/flows/checkout-and-contacts.mjs
node scripts/qa/flows/telegram-pairing-payment.mjs
```

`QA_PROMO_ORDER_IDS` and optional `QA_LEGACY_ORDER_ID` reuse saved fixtures without modifying campaigns/orders.
For a new run, obtain explicit approval for writes and set `QA_PROMO_WRITES=1`. It creates guest orders and,
when supplied, orders owned by `QA_CUSTOMER_SESSION` and `QA_MINI_SESSION`, then changes/disables the campaign
in `finally`. It never completes a payment. Supply a distinct `QA_CUSTOMER_SESSION_2` to verify foreign-order
404s. The sessions must already be valid; the script does not register accounts or authenticate providers.
Existing paired web/mini sessions may share ownership; the script honors the backend's actual access result.

Rerun `promo-checkout.mjs` using approved `QA_PROMO_WRITES=1` or existing `QA_PROMO_FIXTURES` as documented
in its header. Supply the same web/mini sessions to cover authenticated quote flows.
Never store session values in this document or QA logs.

## Required before activation

- Re-run backend formatting/build/vet, serial full suite and required coverage, and admin check/build plus
  promo campaign/order Playwright flows against the release revision. Backend task 89 and admin task 28
  record earlier passes; those are prerequisite records, not checks rerun by this website task.
- With valid customer and Mini App sessions, run both promo flows for pickup/delivery, both widths/themes,
  account lists/details and guest/foreign exclusion. Record real results, including second-account isolation.
- Complete website guest/account and authenticated Telegram checkout and successful/failed payment returns
  on supported devices. Verify return reloads, cart preservation/clearing, explicit handoff, Back button and
  ownership checks. The automated forged-return regression does not prove a real provider return.
- With explicit restaurant/provider authorization, reconcile MONETA's paid amount (discounted food + delivery),
  iiko's food-only payment (discounted food), saved eligible positions and the configured flexible-sum RMS
  discount. Accept per-line RMS rounding only under the shared contract; total discount/payment must agree.
- Check Telegram/staff notification subtotal, promo code, discount, discounted food, delivery and payable
  against the saved snapshot and both customer history surfaces. No live notification/iiko checks ran here.
- Record device/OS, session surface, sanitized order references, provider results and restaurant sign-off.
  Activate a campaign only after every prerequisite and authorized acceptance check passes.
