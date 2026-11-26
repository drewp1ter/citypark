# Task 113 — Approve paid orders before iiko submission

Record approval eligibility on payment and notify staff in Telegram. Provide an admin approval endpoint that saves
per-item exclusions and kitchen comments, then queues a durable iiko delivery with stable identifiers. Preserve the
original payment and order composition. Use the synchronized terminal-group stop list for the review interface.
Flag uncertain deliveries for staff review. Historical paid orders are outside the approval queue.

Update OpenAPI and the API guide. Verify callback replay, authorization, concurrent approval, payload calculation,
restart recovery and uncertain iiko responses with the backend tests.
