# Task 82: Correct MONETA Assistant integration

## Goal
Implement the approved MONETA Assistant correction plan, including signed Check URL validation.

## Depends on
81.

## Read first
- docs/tasks/CONVENTIONS.md
- internal/moneta/moneta.go
- internal/order/service.go
- internal/order/handler.go

## Spec
Use https://docs.moneta.ru/assistant/v1/ as the protocol reference.
Correct the demo Assistant URL; preserve redirectUrl compatibility.
Verify received merchant, currency, mode, subscriber and signature fields.
Support GET and POST Pay URL at /webhooks/moneta and signed XML Check URL at /webhooks/moneta/check.
Check requests never mutate orders. Return 100 for payable orders with omitted amount, 402 for matching payable orders, 200 for recorded payment, and 500 for canceled, missing, nonpayable orders or amount mismatches.
Reject malformed forms and duplicate MNT fields; preserve existing callback idempotency and canceled-order behavior.
Update API.md, OpenAPI, environment example and merchant setup instructions.

## Tests to write
Independent signature vectors, optional fields, wrong signed fields, URL query handling, GET/POST callbacks and Check responses, all Check state mappings, malformed forms, duplicate callbacks, concurrency and database errors.

## Done when
All CONVENTIONS checks pass; changed internal packages have at least 80% coverage.
Perform a real demo payment with publicly reachable Pay and Check URLs before production rollout.

## Out of scope
Frontend contract changes, database migrations, new expiry policies, real merchant account configuration.

## Pay URL simulator follow-up
User-requested Python standard-library CLI: scripts/simulate_moneta.py ORDER_ID. Read credentials from .env/environment, resolve amount through signed Check URL (or --amount), and send a signed Pay notification. Provide --base-url, --env-file, --operation-id and --dry-run. Isolated Python tests must pass.
