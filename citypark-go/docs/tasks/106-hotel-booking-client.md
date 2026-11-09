# Task 106: Implement 1C booking client

## Goal
Implement 1C booking client for one-room, instantly confirmed hotel reservations without prepayment. 1C is authoritative for inventory and tariffs.

## Depends on
1C tasks 01–02

## Read first
- `AGENTS.md`
- `docs/tasks/PROGRESS.md`
- `../citypark-hotel-managment/docs/tasks/01-booking-contract.md` (resolve from this project root).

## Files to create
- `internal/hotelbooking/client.go`
- `internal/hotelbooking/client_test.go`

## Spec
Implement context-aware authenticated HTTP client and validated server-only configuration for the frozen contract. Bound responses and timeouts; sanitize errors and logs. Do not retry ambiguous creates with a new key. Disable booking when integration is unconfigured. Preserve current room content and reporting APIs.

## Tests to write
Use httptest for successful quotes/creation/lookup, invalid JSON, upstream errors, timeout, cancellation, credential isolation and ambiguous creation recovery. Follow existing config tests.

## API documentation
Document any new endpoints in the project's API contract and OpenAPI where applicable. Freeze exact request/response shapes, Russian messages and retry semantics in the prerequisite contract before implementation; do not invent incompatible interfaces.

## Done when
Dependencies are complete; all specified scenarios pass. Follow project checks (Go: formatting, build, vet, serial tests and required coverage; website: npm run check, npm run build and booking browser QA; 1C: Designer checks and development-base runtime tests). Record blockers rather than claiming production validation. Documentation-only tasks require link and consistency checks, not runtime tests.

## Out of scope
Online payment, multi-room carts, guest self-service cancellation or changes, and automatic production deployment. Do not implement another task in this session.
