# Booking environment discovery

Status: task 01 discovery is complete; task 02 implementation draft and owner-run development validation are in progress (2026-11-16).
Source inspection and one authorized HTTP GET were performed. No information base
was created, imported, updated or published by the agent.

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
| Separate development information base and file/server topology | Owner originally reported none (2026-11-09). On 2026-11-16 the owner supplied a separate Windows file base in untracked `1c.txt` and confirmed CityParkHotelDev points to it; runtime validation remains unrun |
| Development platform executable/remote execution access | Owner chose owner-run Windows commands. `1c.txt` supplies the common launcher; version-specific `bin/1cv8.exe` still needs confirmation. No remote execution configured. |
| Isolated development database identity and sanitized sample data | File-base identity supplied in `1c.txt`; owner confirms isolation. Scheduled-task/integration isolation and fixture state must be checked before development mutations. |
| Current publication server, reverse proxy and authentication method | Owner confirmed an HTTP service is already published (2026-11-09). Reports publication GET verified at `http://mbc.cityparkvip.ru:8081/CityParkHotelDev/ru_RU/hs/reports/v1/booking-references`; owner confirmed username/password authentication. Web server, reverse proxy and production TLS remain unverified. Credentials are not recorded here. |
| Development HTTP base URL and TLS/network route from Go | Owner confirmed CityParkHotelDev targets the supplied isolated file base. New Booking service publication and runtime access remain to verify; production transport remains separate. |
| Hotel timezone, 1C session/server timezone | Owner confirmed Moscow (`Europe/Moscow`, 2026-11-09); actual 1C session/server configuration remains unverified |
| Organization/building/room-type/physical-room data UUIDs | Owner supplied reference response on 2026-11-16: 12 unique room UUIDs, three 1C room-type UUIDs, one building/complex and one organization candidate. See BOOKING_CONTRACT.md; booking-organization selection remains unverified. |
| Online price source and 1C document pricing references | Owner confirmed admin-panel room prices calculated by Go. Actual rate data and required 1C document references remain to verify; 1C is not the online price calculator. |
| Check-in/out constant values and date policy | Owner reports production check-in 13:00 (2026-11-18); production check-out remains unconfirmed. On 2026-11-20 owner authorized using existing 1C check-in/out constants, superseding noon boundaries. Date-only input, calendar-night pricing and earliest arrival tomorrow in Moscow remain unchanged. Verify both actual values and align development constants with production before runtime acceptance; do not change production constants for the integration. |

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

## Supplied HTTP/export evidence (2026-11-16)

Owner supplied JSON with `schemaVersion: 1` and all 12 expected rooms. Printed
numbers and UUIDs are usable, all supplied deletion marks are false. Three website
types share one 1C room type, so allocation must use explicit room sets.
The pasted Cyrillic labels contained question marks. A direct authorized GET by
the agent on 2026-11-16 returned HTTP 200 and `application/json; charset=utf-8`;
all 7,398 response bytes parsed as UTF-8 JSON, with intact Cyrillic room/type
names. This resolves the server-encoding concern for the observed deployment;
the earlier client/display/save conversion remains unidentified. The result includes owner and
organization fields removed from current local code; deployment revision needs
comparison. One read-only authenticated GET was executed by the agent; no credentials were saved to repository files.

## Validation record

Static source inspection: performed. Designer configuration checks, development
database update, HTTP publication, concurrency tests and runtime scenarios: **unrun**.
Owner confirmed required guest name/email/phone, date-only input, configured check-in/out
boundaries (revised 2026-11-20; earlier noon rule superseded), unlimited stay length and advance booking (revised 2026-11-16), no child/extra-bed
questions, email confirmation and the absence of a separate development base on
2026-11-09. Guest booking with rate limits and no prior email verification was
confirmed on 2026-11-16. Task 01 contract/documentation is complete.
Task 02 requires runtime validation on the isolated development base. On
2026-11-16 the owner confirmed the file-base identity through `1c.txt` and explicitly
confirmed that CityParkHotelDev points to it. Owner-run import, Designer checks,
fixture/integration isolation checks and runtime scenarios are still unrun; see
[BOOKING_RUNTIME_RESULTS.md](BOOKING_RUNTIME_RESULTS.md). Task 03 additionally depends on backend 107 and website 73;
its deployment rehearsal is not possible with the currently known environment.
