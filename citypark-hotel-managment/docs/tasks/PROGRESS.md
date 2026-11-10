# Progress

## Hotel booking

- [ ] [01 Verify booking rules and freeze integration contract](01-booking-contract.md) [Nothing]
  - HTTP reference export implemented in existing Reports service: GET `/v1/booking-references`, existing publication authentication/permissions, JSON/no-store responses and generic failure handling. XML/handler/static checks pass; Designer, authentication and runtime regression acceptance remain pending owner execution. No deployment performed.
  - Owner requested removal of room owner dereferences: both script and HTTP export now include room/type data only; organization/building/complex data excluded. Runtime validation pending.
  - Reference export prepared: [instructions](../EXPORT_BOOKING_REFERENCES.md) and read-only BSL script for rooms/types/buildings/complexes without organization data (removed at owner request). Exported metadata models building ownership through hotel complexes; booking organization resolution is separate. Script not compiled/run in 1C; owner execution and mapping confirmation pending.
  - Discovery started (2026-11-09): inspected booking/staff posting, occupancy boundaries, cancellation/undo and tariff calculation. Drafts: [contract](../BOOKING_CONTRACT.md), [environment](../BOOKING_ENVIRONMENT.md). Explicit shared room locking was not found in the inspected posting paths; multiple tariff changes need regression verification. Contract remains unfrozen pending owner booking rules, stable live room mappings and platform/development-base details. No runtime configuration changes or production writes; tasks 02–03 remain dependency-blocked.
  - Owner confirmed (2026-11-09): guest phone required; guests choose arrival/departure dates only, without exact-time input; check-in after 12:00, check-out before 12:00; no separate development 1C information base exists. Actual 1C time constants remain unverified; remaining rules and proposed defaults await answers. Runtime implementation acceptance requires a separate development base.
  - Additional owner decisions (2026-11-09): only name/email/phone required for guest contacts; no child/extra-bed questions; unlimited maximum stay; six-month advance horizon; confirmation by email. Contract updated; live mappings, environment discovery and remaining technical/date policies still block freezing task 01.

  - Environment update (2026-11-09): owner reports 1С:Предприятие 8.3 (8.3.11.3133) and hotel timezone Moscow (`Europe/Moscow`). Owner reports hosting on Windows Server 2012; actual 1C session/server timezone configuration remains unverified.
  - Owner mapping/pricing update (2026-11-09): use admin-panel room prices in Go; all five website types mapped to 12 distinct room numbers in BOOKING_CONTRACT.md. Stable 1C data UUIDs and organization/building relationships still need lookup.
  - Same-day policy confirmed (2026-11-09): same-day bookings unavailable; earliest arrival tomorrow in Moscow time. Contract includes creation-time revalidation across Moscow midnight; no hourly cutoff needed.
  - Publication update (2026-11-09): owner confirmed an HTTP service is already published. Service identity, web server/publication path, TLS and authentication details remain unverified; no publication changes performed.

- [ ] [02 Implement 1C booking HTTP service](02-booking-api.md) [1C task 01]
  - Scope revision required (owner decision, 2026-11-09): Go calculates online prices; 1C owns inventory and persists the accepted backend price. Original 1C-authoritative pricing wording is superseded. Final rate source, price snapshot and staff check-in behavior must be frozen in task 01 before expanding implementation tasks.

- [ ] [03 Write manual booking deployment and rollback guide](03-booking-production-runbook.md) [1C 02; backend 107; website 73]
