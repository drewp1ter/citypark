# Task 84: Promo iiko contract

## Goal
Document a verified iiko representation for the agreed line discounts.

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
Document a verified iiko representation for the agreed line discounts.
Consult official iiko documentation for the deployed integration. Record dated source URLs, supported payload
fields, external payment behavior, required settings and exact JSON fixtures. Decide and document how capped
proportional discounts and quantities that do not divide evenly into unit prices reconcile to the kopeck.
Include both pickup and courier fixtures, ineligible lines, overlapping targets and a cap. Preserve existing
delivery treatment. Document required configuration and an exact mapping from saved line discounts to payload.
If official evidence cannot establish a supported representation, leave this task blocked with the reason.
Do not send real iiko orders or change code. The resulting document is a mandatory input to task 89.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
No application tests: documentation only. Manually reconcile each fixture: original food minus discounts = iiko item total = food payment; MONETA includes delivery separately.

## API documentation
No endpoint changes in this task; consume the published backend contract.

## Done when
Verify source links, reconcile fixture arithmetic, document setup and unsupported cases. Tick only after the wire representation is fully established; otherwise record the blocker.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
