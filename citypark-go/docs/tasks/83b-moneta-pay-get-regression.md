# Task 83b: Repair the MONETA GET callback regression test

## Goal
Resolve the existing TestPayGET failure without weakening payment validation, so the required full suite can validate task 85.

## Depends on
82 (implemented MONETA callback contract).

## Read first
- `docs/tasks/CONVENTIONS.md`
- `docs/tasks/82-moneta-assistant.md`
- `internal/order/check_test.go`
- `internal/order/service.go`

## Files to create
None.

## Files to change
- `internal/order/check_test.go`: correct the valid GET callback fixture and cover malformed GET callbacks.
- `docs/tasks/PROGRESS.md`: record validation and tick only this task after all checks pass.

## Spec
Current TestPayGET sends only MNT_TRANSACTION_ID to an in-process router, then expects 200 SUCCESS, paid status, payment ID 555 and one notifier call. PaymentConfirmed compares the received amount with the persisted food total plus delivery; the fixture omits that amount and operation ID. Restarting the running backend cannot affect this test.

Compare the fixture with task 82 and existing valid POST callback tests. Fix the valid GET request to supply MNT_TRANSACTION_ID for the seeded order, MNT_OPERATION_ID=555 and MNT_AMOUNT=790.00 (the seeded order's payable amount). Construct query parameters with url.Values and retain existing success, persisted payment and notifier assertions. Use the existing fake Payments verifier consistently with other handler tests; do not introduce real credentials or callbacks.

Add explicit GET rejection coverage for omitted amount and for an amount different from the persisted payable amount: expect HTTP 400 with plain FAIL, unchanged pending status/payment ID and no payment notification. Do not assume that production rejects a missing operation ID unless that behavior is established by the implemented contract; changing that validation is outside this test-fixture repair.

Do not weaken handler/service checks, alter expected successful payment semantics, skip tests or suppress the failure. If the investigation establishes a production defect rather than an invalid success fixture, record the precise blocker and stop instead of expanding this task.

## Tests to write
- `TestPayGET`: complete valid GET callback → HTTP 200 SUCCESS, paid order, payment ID 555, one notification.
- `TestPayGETInvalidAmount`: table cases for omitted amount and wrong amount → HTTP 400 FAIL, unchanged pending order, no notification.

Use existing order test fixtures, fake payment verification, testutil.NewDB and serial database tests. No real payment or backend HTTP requests.

## API documentation
No endpoint behavior changes; no API.md/OpenAPI changes required for this test-only repair.

## Done when
Load TEST_DB_DSN from .env.development. Never start PostgreSQL.

- `go test -count=1 -p 1 -run 'TestPayGET' ./internal/order`
- `gofmt -l .` prints nothing.
- `go build ./...`
- `go vet ./...`
- `go test -count=1 -p 1 ./...`
- `go test -p 1 -cover ./internal/order` reports at least 80% coverage.

Tick task 83b only after these pass; commit as `task 83b: Repair MONETA GET callback regression test`. Stage only this task's changes, leaving task 85 implementation separate. Do not mark task 85 complete in this task; its own checks and commit follow separately.

## Out of scope
Production payment changes, real demo payments, credentials, promo implementation, changes to tasks 82/83 completion or historical blocker notes, and unrelated fixes.
