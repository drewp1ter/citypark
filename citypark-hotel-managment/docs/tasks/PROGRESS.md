# Progress

## Hotel booking

- [x] [01 Verify booking rules and freeze integration contract](01-booking-contract.md) [Nothing]
  - Completed documentation/discovery (2026-11-16): [contract v1](../BOOKING_CONTRACT.md), [environment/setup prerequisites](../BOOKING_ENVIRONMENT.md), five explicit room sets covering 12 UUIDs, source inspection, exact internal/public interfaces, technical defaults and downstream implementation paths/signatures/named scenarios. Go owns pricing from admin room rates; existing 1C booking/posting is reused with shared locks and price-snapshot preservation.
  - Owner confirmed name/email/phone; no child/extra-bed questions; dates only, noon boundaries in Moscow time; no same-day arrival; unlimited maximum stay and advance booking (revised 2026-11-16); email confirmation; guest booking with rate limits and no prior email verification. Platform 8.3.11.3133 on Windows Server 2012; published service requires credentials.
  - Date-policy revision (2026-11-16): removed the six-month arrival horizon and month-clamping requirement. Earliest arrival remains tomorrow in Moscow time; minimum one night. Acceptance includes dates beyond six months, stays over 90 nights, full-stay inventory and date/money representation limits.
  - Owner supplied references and authorized a direct read-only GET: HTTP 200, valid UTF-8 Cyrillic. Local reference export excludes organization/owner fields; deployed discovery response includes them. This revision difference is documented, not treated as booking acceptance. [Export instructions](../EXPORT_BOOKING_REFERENCES.md).
  - JSON examples, UUID/mapping consistency, documentation links and CRLF-aware diff checks pass. Booking scenarios, Designer and latest exporter runtime checks remain unrun. No booking writes or deployment performed.

- [ ] [02 Implement 1C booking HTTP service](02-booking-api.md) [1C task 01]
  - Contract dependency cleared. Runtime acceptance requires an identified isolated development base (owner previously reported none), service-account/reference setup and Designer/concurrency checks. Production booking writes remain unauthorized. Reuse the existing document workflow and preserve backend pricing through staff check-in.

- [ ] [03 Write manual booking deployment and rollback guide](03-booking-production-runbook.md) [1C 02; backend 107; website 73]
