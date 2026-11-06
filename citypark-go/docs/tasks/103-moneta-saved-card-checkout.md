# Task 103: Saved cards through MONETA checkout

## Goal
Enable authenticated customers to save/reuse cards on hosted MONETA checkout using MNT_SUBSCRIBER_ID.

## Depends on
99, 100. Independent of unfinished task 101; explicitly requested by the owner.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/moneta/moneta.go`, `internal/order/service.go`, `internal/app/payments.go`.

## Files to create
- `docs/tasks/103-moneta-saved-card-checkout.md`

## Files to change
- `internal/moneta/moneta.go`, `internal/moneta/moneta_test.go`: subscriber-aware signatures and URLs.
- `internal/order/service.go`, `internal/order/payment_retry.go`: derive account identity for creation and retries.
- `internal/order/ownership_test.go`, `internal/order/payment_retry_test.go`, `internal/order/payment_uuid_test.go`: identity and signing coverage.
- `internal/app/payments.go`, `internal/app/app_test.go`: pass identity to signer.
- `docs/API.md`, `internal/apidocs/openapi.yaml`, `docs/tasks/PROGRESS.md`: contract and progress.

## Spec
Extend `order.PaymentRequest` with `SubscriberID string`.
Signer signatures become:
```go
func (s *Signer) RequestSignature(transactionID, amount, subscriberID string) string
func (s *Signer) PaymentURL(base, transactionID, amount, successURL, failURL, subscriberID string) string
```
Sign MD5 of merchant + transaction + amount + currency + subscriber + test mode + secret.
The authenticated customer account maps to `citypark-user-<accountID>`. Guests and staff without customer context omit it.
Use the same mapping for new orders, duplicates and authorized payment retries. Never use delivery recipient identity or browser-provided identifiers.
Overwrite stale base-URL subscriber parameters; remove them when the subscriber is empty.
Existing callback verification, ownership, UUID lookup, prices, notifications and public API shapes remain unchanged.
Never log subscriber IDs or store card data. No database migration.
After account merge, new links use the surviving account; source saved cards are not migrated. Old signed links and valid delayed callbacks retain existing validation.

## Tests to write
- `TestSubscriberPaymentURL`: empty/populated/encoded identifiers, live/test signatures, stale parameter removal/replacement, unrelated query preservation.
- `TestDuplicateIsolation`: guests/accounts share recipient contacts but use isolated identities across creation, duplicates and contact changes.
- `TestRetryPayment`: repeated authorized retries retain account identity and saved price without mutations.
- `TestPaymentsAdapter`: amount, URL and signature include the subscriber.
Retain existing signature vectors, callback tampering and cross-account authorization tests.

## API documentation
Update POST /orders and POST /orders/{id}/payment-retry notes and the OpenAPI result description; no new routes or input fields.

## Done when
CONVENTIONS checks pass with changed-package coverage >=80%.
Before production release, a human must verify merchant support and MONETA demo acceptance: explicit choice to save or decline, same-account reuse, and isolation from another account. Missing save-card controls block release, not code completion.

## Out of scope
Frontend/billing UI, card management, PAYMENTTOKEN, raw card handling, guaranteed one-click payment, production deployment and live financial transactions.
