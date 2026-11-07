# Task 33: Delivery-town regions

## Goal
Configure Краснодарский край or Республика Адыгея per delivery town.

## Depends on
backend 105; admin 32.

## Read first
`AGENTS.md`, `docs/tasks/CONVENTIONS.md`.

## Files to change
Delivery-price persistence/API (backend), town form/list (admin), or delivery-town fetching and address provider integration (website).

## Spec
Use regionCode RU-KDA or RU-AD. Default existing towns and new towns to RU-KDA. Omitted updates preserve the region. Assign Адыгея manually. Suggestions and geolocation use the configured region. Keep town names unique and address/order payloads compatible.

## Tests to write
Check both regions, defaults, preservation, invalid codes, and region mismatch in address matching.

## API documentation
Update delivery-price GET, POST and PATCH documentation and OpenAPI; regenerate admin types.

## Done when
Project convention checks pass, including relevant delivery/address regressions.

## Out of scope
Deployment and automatic town reassignment.
