# Task 39: Refresh stop-list markers while the menu is open

## Depends on

Backend task 112 and admin task 38.

## Changes

- Refetch the menu products every 15 seconds while the admin menu is open in the foreground. Use the refreshed product data for dish and category indicators.
- Extend the menu browser regression to simulate a stop-list change from the backend while the same page stays open.

## Done when

Run the admin checks and build and `npm run e2e -- e2e/dish-visibility.spec.ts`. Verify against a running backend with task 112.
