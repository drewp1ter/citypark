# Task 87: Promo pricing and quotes

## Goal
Provide authoritative side-effect-free quotes using one reusable pricing calculation.

## Depends on
85

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](PROMOCODES.md)
- `internal/order/service.go`
- `internal/order/handler.go`

## Files to create
- `internal/promo/pricing.go`
- `internal/promo/pricing_test.go`
- `internal/order/quote.go`
- `internal/order/quote_test.go`

## Files to change
- `internal/order/handler.go`
- `internal/apidocs/openapi.yaml`
- `docs/API.md`

## Spec
Provide authoritative side-effect-free quotes using one reusable pricing calculation.
Export promo.Line with ProductID, CategoryID int64, Amount int and UnitPrice money.Kopecks; Pricing with
Subtotal, EligibleSubtotal, Discount, Total money.Kopecks and LineDiscounts map[int64]money.Kopecks.
Expose Calculate(c Campaign, lines []Line, now time.Time) (Pricing,error). Use passed clock time, checked
integer arithmetic and the shared eligibility/allocation rules. Keep availability checks in the order layer.
Expose Service.Quote(ctx context.Context,in QuoteInput) (Quote,error) with the shared wire shape. Merge repeated
products; load fresh catalog data and campaign snapshot consistently; resolve town and free-delivery pricing.
Register public POST /orders/quote with existing optional authentication and private/no-store response behavior.
No orders, customer rows, reservations or payment links are created. Invalid codes return field errors.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- TestCalculate: 1/99 percent, odd kopecks, cap, exact minimum, below minimum, no match, mixed union and overlap.
- TestAllocate: stable tie break, non-divisible quantities, sum exactly equals discount, checked overflow.
- TestCampaignWindow: start inclusive/end exclusive, disabled, no bounds, fixed clock.
- TestQuote: delivery/pickup, discount crosses free threshold, no-code compatibility, duplicate items merged.
- TestQuoteCatalogChanges: category move/new member, price change, unavailable product → current rules.
- TestQuoteErrors: malformed body, unknown town/product, invalid code and minimum → exact errors, no writes.
Use clock.Fixed, table tests and testutil.NewDB; assert quote leaves orders/customers unchanged.

## API documentation
Add POST /orders/quote request/result/error schemas and examples to API.md and OpenAPI.

## Done when
Run all CONVENTIONS.md checks: gofmt -l . (empty), go build ./..., go vet ./...,
go test -p 1 ./..., and go test -p 1 -cover for every changed internal package (at least 80%).
Load TEST_DB_DSN from .env.development; do not start PostgreSQL. Update API.md/OpenAPI where applicable.
Tick only this implemented task and use the required task commit after all checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
