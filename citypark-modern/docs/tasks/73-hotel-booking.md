# Task 73: Implement hotel reservation flow

## Goal
Implement hotel reservation flow for one-room, instantly confirmed hotel reservations without prepayment. 1C is authoritative for inventory; Go calculates prices from admin-managed room rates.

## Depends on
backend 107; 1C task 01

## Read first
- `AGENTS.md`
- `docs/tasks/PROGRESS.md`
- `../citypark-hotel-managment/docs/tasks/01-booking-contract.md` (resolve from this project root).

## Files to create
- Booking route/page and form paths fixed by task 01
- `scripts/qa/flows/hotel-booking.mjs`

## Spec
Replace today-only availability with full-stay availability from 1C and Go price quotes. Preserve selected room type, dates and exact occupancy through search and booking. Replace ambiguous 3–4 occupancy selection with exact counts according to the contract. Add guest form and confirmation using existing design tokens and Russian copy. Use server-side API calls only; preserve inputs on errors and stable idempotency through retries. Display price changes for explicit reconfirmation and distinguish unavailable, unknown and pending outcomes. Preserve existing room content APIs; do not infer availability from today counts. Implement agreed confirmation delivery only as specified in the contract.

## Tests to write
Production-build QA at 390/1440px in both themes: search, selection, exact guests, quote, validation, confirmation, double submit, sold-out race, changed price, timeout recovery, outage, keyboard navigation and input preservation. Booking writes only against an approved development environment.

## API documentation
Document any new endpoints in the project's API contract and OpenAPI where applicable. Freeze exact request/response shapes, Russian messages and retry semantics in the prerequisite contract before implementation; do not invent incompatible interfaces.

## Done when
Dependencies are complete; all specified scenarios pass. Follow project checks (Go: formatting, build, vet, serial tests and required coverage; website: npm run check, npm run build and booking browser QA; 1C: Designer checks and development-base runtime tests). Record blockers rather than claiming production validation. Documentation-only tasks require link and consistency checks, not runtime tests.

## Out of scope
Online payment, multi-room carts, guest self-service cancellation or changes, and automatic production deployment. Do not implement another task in this session.

## Frozen implementation specification (2026-11-16)

Follow [Booking contract v1](../../../citypark-hotel-managment/docs/BOOKING_CONTRACT.md), including its exact HTTP shapes, implementation map, paths, signatures and named acceptance scenarios for this task. The contract supersedes the original 1C-pricing wording. Runtime validation and environment gates remain mandatory; this expansion does not mark this task complete.
