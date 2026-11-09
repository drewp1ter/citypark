# Task 03: Write manual booking deployment and rollback guide

## Goal
Write manual booking deployment and rollback guide for one-room, instantly confirmed hotel reservations without prepayment. 1C is authoritative for inventory and tariffs.

## Depends on
1C 02; backend 107; website 73

## Read first
- `AGENTS.md`
- `docs/tasks/PROGRESS.md`
- `../citypark-hotel-managment/docs/tasks/01-booking-contract.md` (resolve from this project root).

## Files to create
- `docs/BOOKING_DEPLOYMENT.md`

## Spec
Write exact owner-executed steps based on verified platform and hosting: backup and restore verification, development-base rehearsal, Designer import/check/update, restricted account setup, HTTP publication/TLS/network access, backend secrets and configuration, deployment ordering and disabled-by-default activation. Include screenshots/menu names or precise commands where verified. Never invent server paths or expose credentials. Include controlled production smoke booking and cancellation, occupancy verification, monitoring, troubleshooting and rollback. Preserve real reservations and idempotency records during rollback; restoring an old database after bookings exist requires reconciliation. Owner performs all production changes.

## Tests to write
Rehearse instructions on the development base. Record platform/version and results. Production execution remains owner-pending; documentation completion does not mean deployed.

## API documentation
Document any new endpoints in the project's API contract and OpenAPI where applicable. Freeze exact request/response shapes, Russian messages and retry semantics in the prerequisite contract before implementation; do not invent incompatible interfaces.

## Done when
Dependencies are complete; all specified scenarios pass. Follow project checks (Go: formatting, build, vet, serial tests and required coverage; website: npm run check, npm run build and booking browser QA; 1C: Designer checks and development-base runtime tests). Record blockers rather than claiming production validation. Documentation-only tasks require link and consistency checks, not runtime tests.

## Out of scope
Online payment, multi-room carts, guest self-service cancellation or changes, and automatic production deployment. Do not implement another task in this session.
