# Booking environment discovery

Status: incomplete; inspected local workspace on 2026-11-09. No information base
was connected, created, imported, updated or published.

## Evidence available locally

* `Configuration.xml`: configuration version `1.0.0.0`, compatibility mode
  `Version8_3_4`, extension compatibility mode `Version8_3_11`, managed application,
  managed data locks. These values do **not** identify the installed platform build.
* Existing HTTP service metadata: `HTTPServices/Reports.xml`, root URL `reports`,
  routes `/v1/sales` and `/v1/room_sales`. Actual publication URL, TLS, network and
  authentication setup cannot be inferred from this metadata.
* No `.v8-project.json` was found by the workspace search, including hidden files.
* `1cv8` is not available on the current shell's PATH. This does not establish
  whether a platform is installed elsewhere or on the owner's workstation/server.
* Exported metadata does not include live room catalog records, tariff values,
  arrival/departure constant values or the active server timezone.

## Owner-supplied setup details needed

| Item | Status |
| --- | --- |
| Exact platform version/build and operating system | Owner reports 1С:Предприятие 8.3 (8.3.11.3133), hosted on Windows Server 2012 (2026-11-09) |
| Separate development information base and file/server topology | Owner confirmed no separate development base exists (2026-11-09); topology unknown |
| Development platform executable/remote execution access | Unknown |
| Isolated development database identity and sanitized sample data | Unknown |
| Current publication server, reverse proxy and authentication method | Owner confirmed an HTTP service is already published (2026-11-09). Service identity, web server, publication path, TLS, reverse proxy and authentication method remain unverified. |
| Development HTTP base URL and TLS/network route from Go | Unknown |
| Hotel timezone, 1C session/server timezone | Owner confirmed Moscow (`Europe/Moscow`, 2026-11-09); actual 1C session/server configuration remains unverified |
| Organization/building/room-type/physical-room data UUIDs | Owner supplied room-number mapping for all 12 rooms; stable UUIDs and organization/building relationships still need lookup. See BOOKING_CONTRACT.md. |
| Online price source and 1C document pricing references | Owner confirmed admin-panel room prices calculated by Go. Actual rate data and required 1C document references remain to verify; 1C is not the online price calculator. |
| Check-in/out constant values and date policy | Owner confirmed date-only guest input, check-in after 12:00 and check-out before 12:00 (2026-11-09). Same-day bookings unavailable; earliest arrival tomorrow in Moscow time. Actual 1C constant values remain unverified; remaining date policies pending. |

Store credentials outside version control; this document should contain only
non-secret topology, version and test evidence.

## Prerequisites for development validation

1. Identify a separate development base; confirm its identity before any mutation.
   Isolate notifications, scheduled tasks and production integrations in the copy.
2. Record the exact platform build, deployment topology, timezone and current
   configuration revision; arrange a backup with a verified restore procedure.
3. Read the local `db-load-xml` and `db-run` skills before invoking automation.
   Their presence does not establish that the required platform/base is available.
4. Load/check the configuration in Designer, update the development database,
   then exercise the scenarios in [BOOKING_CONTRACT.md](BOOKING_CONTRACT.md).
5. Capture version, fixture IDs, observed movements, concurrent outcomes and
   cleanup results. Static XML checks do not establish runtime correctness.

## Validation record

Static source inspection: performed. Designer configuration checks, development
database update, HTTP publication, concurrency tests and runtime scenarios: **unrun**.
Owner confirmed required guest name/email/phone, date-only input, noon check-in/out
boundary, unlimited stay length, six-month booking horizon, no child/extra-bed
questions, email confirmation and the absence of a separate development base on
2026-11-09. Task 01 remains open pending remaining owner decisions and
environment/reference discovery.
Task 02 cannot be completed until task 01 is frozen and development runtime
validation is available. Task 03 additionally depends on backend 107 and website 73;
its deployment rehearsal is not possible with the currently known environment.
