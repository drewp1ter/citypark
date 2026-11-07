# Task 34: Styled region select

## Goal
Replace the native delivery-town region select with HeroUI v3 Select.

## Depends on
33.

## Read first
AGENTS.md, docs/tasks/CONVENTIONS.md, installed HeroUI Select types, delivery-price dialog.

## Files to change
Delivery-price dialog, delivery-price browser tests.

## Spec
Use Select with Label, Trigger, Value, Indicator, Popover and ListBox. Preserve the two codes, required selection, controlled form state and field errors. Keep the icon inset using the component theme.

## Tests to write
Adapt selection and persistence assertions to the visible trigger and options. Verify both themes and mobile layout with the existing real-backend flows.

## Done when
npm run check, npm run build, delivery-prices Playwright tests pass.

## Out of scope
Backend changes.
