# Task 69: Saved address recipient contacts

## Goal
Store recipient name and phone per saved address and use them at checkout.

## Depends on
59; backend 102

## Read first
`docs/tasks/CONVENTIONS.md`, `AGENTS.md` and the address entity/model.

## Files to create
None.

## Files to change
Address entity, address management/import, checkout, PhoneField and address QA harnesses.

## Spec
Add `recipientName` and `recipientPhone` to saved-address writes and responses, retaining `name` as the address label. Name is trimmed, 2–80 Unicode code points; Russian phone is normalized to +7XXXXXXXXXX. New and edited entries require both. Legacy entries remain readable with empty contacts. Preserve existing duplicate and merge survivor rules. Guests retain version 1 storage with backward-compatible empty defaults. Carry contacts through import; incomplete imports fail validation without deleting the guest entry. Checkout selections fill contacts; automatic defaults preserve touched contact fields, legacy empty values preserve current inputs. Explicit checkout saves include contacts. Display Russian recipient labels and saved contacts. Backend/frontend deployment must be coordinated because old clients omit required contacts.

## Tests to write
Validate normalization, missing/invalid contacts, persistence and edits. Extend existing address API, storage, import and checkout checks, including legacy entries and address selection.

## API documentation
Update customer address create/update and response contracts in API.md and OpenAPI; no new routes.

## Done when
Required conventions checks pass, plus address QA harnesses for the frontend. Backend changed packages meet 80% coverage.

## Out of scope
Shared profile contact changes and deployment.
