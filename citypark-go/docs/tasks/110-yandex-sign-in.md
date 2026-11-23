# 110 Yandex ID sign-in

## Read first

Read the customer Google flow in `docs/API.md` and its backend implementation.

## Goal

Support customer web login and linking with Yandex ID using provider-bound OAuth
attempts. Keep the existing customer session and account ownership rules.

## Done when

- The migration, API contract, and configuration support Yandex.
- Go formatting, build, vet, and serial tests pass against the configured test DB.
