# 75 Yandex ID sign-in

## Read first

Read the Yandex endpoint contract in the backend API guide and the account OAuth flow.

## Goal

Expose Yandex ID sign-in and linking on the customer website when enabled.

## Done when

- Backend task 110 is available on staging.
- `npm run check` and `npm run build` pass.
- Verify Yandex login, linking, cancellation, and flow-cookie cleanup in a browser.
  Verify the existing Google flow and the disabled Yandex controls.
