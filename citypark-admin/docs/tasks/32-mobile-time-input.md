# Task 32: Fix time-field keyboards on mobile

## Goal
Allow typing a colon in category ordering schedules and working hours on mobile.

## Scope
Request the standard text keyboard for the existing time fields. Keep accepted formats,
category end time `24:00`, empty category limits and validation unchanged.

## Done when
- `npm run check`
- `npm run build`
- `npm run e2e -- e2e/working-hours.spec.ts e2e/menu.spec.ts`
- The mobile flows assert text keyboard mode, save/reload times, preserve category
  `24:00` and empty limits, and restore original values.
