# Task 84: Promo RMS contract

## Goal
Document the supported RMS payload for the agreed total discount and eligible positions.

## Depends on
83

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](PROMOCODES.md)
- `internal/iiko/delivery.go`
- `internal/app/iiko_orders.go`

## Files to create
- `docs/tasks/IIKO_PROMOCODES.md`

## Files to change
None.

## Spec
Document the supported RMS payload for the agreed total discount and eligible positions.
Consult official iiko documentation for deliveries/create and the RMS discount dictionary. Record dated source
URLs, supported fields, external payment behavior and exact pickup/courier JSON fixtures in IIKO_PROMOCODES.md.
Use one configured RMS flexible-sum discount with saved total discount and selectivePositions. Keep original
prices and quantities; document stable unique position UUIDs and saved eligibility, including eligible lines with
zero backend allocation. Omit the discount entry for zero total discount. Backend and iiko per-line allocations
may differ; exact total discount, eligible positions and food payment must match. Include a binding cap,
overlapping targets, an ineligible line and quantities whose net unit prices would not divide evenly.
Document IIKO_PROMO_DISCOUNT_TYPE_ID, assign-amount/selective settings, payment compatibility and deployment
acceptance requirements. Preserve existing delivery treatment. Distinguish schema evidence from restaurant setup
and runtime acceptance. No real orders or code changes. If evidence cannot establish this revised RMS mapping,
record the specific remaining blocker; per-line rounding differences alone are no longer a blocker.
The resulting document is a mandatory input to task 89.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
No application tests: documentation only. Manually reconcile each fixture: original food minus discounts = iiko item total = food payment; MONETA includes delivery separately.

## API documentation
No endpoint changes in this task; consume the published backend contract.

## Done when
Verify source links, reconcile fixture arithmetic, document setup and unsupported cases. Tick only after the wire representation is fully established; otherwise record the remaining blocker. Restaurant setup and authorized runtime acceptance are release checks.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
