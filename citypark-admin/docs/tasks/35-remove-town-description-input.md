# Task 35: Remove town description input

## Goal
Remove «Описание» from the town edit/create dialog.

## Depends on
34.

## Read first
AGENTS.md, docs/tasks/CONVENTIONS.md, delivery-price dialog.

## Files to change
Delivery-price dialog.

## Spec
Remove the description TextField. Keep the existing value in form state and requests so editing another field does not erase stored descriptions. New towns continue sending an empty value. Keep other town controls intact.

## Tests to write
No new logic. Run existing delivery-price browser flows for create/edit persistence, validation and mobile layout in both themes.

## Done when
npm run check, npm run build and delivery-prices browser tests pass.

## Out of scope
Backend schema/API removal and changes to list descriptions or search.
