# Task 01: Verify booking rules and freeze integration contract

## Goal
Verify booking rules and freeze integration contract for one-room, instantly confirmed hotel reservations without prepayment. 1C is authoritative for inventory and tariffs.

## Depends on
Nothing

## Read first
- `AGENTS.md`
- `docs/tasks/PROGRESS.md`
- `../citypark-hotel-managment/docs/tasks/01-booking-contract.md` (resolve from this project root).

## Files to create
- `docs/BOOKING_CONTRACT.md`
- `docs/BOOKING_ENVIRONMENT.md`

## Spec
This is a prerequisite discovery/documentation task, not permission to deploy. Inspect booking document filling/posting, occupancy registers, tariffs, cancellation and staff booking paths. Identify platform version, publication topology and a separate development information base; availability is currently unknown. Document setup prerequisites without credentials.
Resolve with the owner: required guest/contact fields, children and extra beds, check-in/out times, maximum stay/advance horizon, confirmation delivery and abuse prevention. Record concrete choices before marking complete.
Map the five website room-type IDs to physical 1C rooms using stable references. Require one physical room to be free for the entire stay. Define hotel-local date boundaries and tariff calculation including extras and changed-price consent.
Specify website → Go → authenticated 1C HTTP service contracts for availability/quote, creation and lookup by idempotency key. Define exact fields, examples, statuses and Russian errors. Confirmation requires a committed 1C reservation. Repeated creation must return the same reservation; timeout after commit must be recoverable without duplication. Define locking shared with staff posting paths to prevent overbooking.
Expand downstream tasks with exact file paths, signatures and test names from the approved contract before clearing their dependency. Do not mark unresolved product decisions as approved.

## Tests to write
Document a scenario matrix for concurrent website/staff bookings, full-stay availability, adjacent stays, tariffs, cancellations and timeout recovery. No runtime code in this task.

## API documentation
Document any new endpoints in the project's API contract and OpenAPI where applicable. Freeze exact request/response shapes, Russian messages and retry semantics in the prerequisite contract before implementation; do not invent incompatible interfaces.

## Done when
Dependencies are complete; all specified scenarios pass. Follow project checks (Go: formatting, build, vet, serial tests and required coverage; website: npm run check, npm run build and booking browser QA; 1C: Designer checks and development-base runtime tests). Record blockers rather than claiming production validation. Documentation-only tasks require link and consistency checks, not runtime tests.

## Out of scope
Online payment, multi-room carts, guest self-service cancellation or changes, and automatic production deployment. Do not implement another task in this session.
