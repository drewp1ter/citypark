# Task 02: Implement 1C booking HTTP service

## Goal
Implement 1C booking HTTP service for one-room, instantly confirmed hotel reservations without prepayment. 1C is authoritative for inventory and tariffs.

## Depends on
1C task 01

## Read first
- `AGENTS.md`
- `docs/tasks/PROGRESS.md`
- `../citypark-hotel-managment/docs/tasks/01-booking-contract.md` (resolve from this project root).

## Files to create
- Service metadata and BSL modules identified by task 01
- Idempotency metadata and restricted role identified by task 01

## Spec
Implement the approved contract using existing document filling, tariff and posting logic. Add least-privilege service access. Allocate and recheck a physical room transactionally; enforce shared locking with staff booking paths. Persist idempotency records atomically with reservations. Reject reuse with a different payload. Return only necessary guest-safe data. Preserve XML identifiers, encoding, BOM and BSL style. Never write to production during implementation.

## Tests to write
In development 1C: valid booking and register movements; occupied dates; adjacent stays; capacity; tariffs; simultaneous website/staff requests; duplicate key; changed payload; rollback; timeout-after-commit lookup; unauthorized access.

## API documentation
Document any new endpoints in the project's API contract and OpenAPI where applicable. Freeze exact request/response shapes, Russian messages and retry semantics in the prerequisite contract before implementation; do not invent incompatible interfaces.

## Done when
Dependencies are complete; all specified scenarios pass. Follow project checks (Go: formatting, build, vet, serial tests and required coverage; website: npm run check, npm run build and booking browser QA; 1C: Designer checks and development-base runtime tests). Record blockers rather than claiming production validation. Documentation-only tasks require link and consistency checks, not runtime tests.

## Out of scope
Online payment, multi-room carts, guest self-service cancellation or changes, and automatic production deployment. Do not implement another task in this session.
