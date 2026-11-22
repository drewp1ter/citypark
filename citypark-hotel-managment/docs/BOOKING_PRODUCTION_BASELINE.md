# Production-export baseline comparison

Evidence date: 2026-11-17. The owner supplied an XML configuration export from
production at local `hotel-orig/` (the message spelled it `horel-orig`). This is an
owner-supplied reference export, not a direct production-system inspection. The
agent did not change that directory or connect to production.

## Owner-native production check supplied (2026-11-22)

Owner identifies screenshots2026-11-22_19-16.png and19-17.png as configuration
checking on the production database. The agent did not connect to or modify that
base. These provide native check evidence in addition to the earlier production
source export.

Visible diagnostics match the development baseline: unresolved report/form
handler references, АвансовыйОтчет incorrect data path, missing BSP identifiers
in ПанельАдминистрированияБСП/ПанельАдминистрированияБСПВМоделиСервиса, and the
SORC import-form unresolved metadata reference. The23 visible missing-identifier
locations follow the same form names and line/column positions as the earlier
baseline, including ПоддержкаИОбслуживание(45,49), (423,3), (478,64), (488,36),
(165,15), ОбщиеНастройки(240,3), and the remaining administration forms shown.
No new booking-module diagnostic is visible in the development check screenshots.

This supports classifying the reported failures as pre-existing production
baseline diagnostics, not newly introduced booking errors. Full checking still
fails in both reported environments. Screenshots alone do not establish a byte-
exact full diagnostic comparison; no fresh production text log, selected check
options or platform version screenshot was supplied with this message. No clean
check or automatic waiver/completion is inferred. Baseline remediation remains
separate from the booking implementation review.

## Existing configuration-check failures

The reference also lacks all ten common modules named by the remaining 23 compiler
errors. None is declared in its `Configuration.xml`, and none has a definition in
its `CommonModules/` directory. All ten affected form BSL modules are **byte-for-byte
identical** between the reference and working exports. Each of the 23 logged
identifier/line-number pairs matches the corresponding production-export source
line. This establishes that these source dependency defects predate booking.
It does not claim that a fresh native full check of the reference was performed.

The form XML files for the incorrect data path in `АвансовыйОтчет` and unresolved
metadata reference in `ЗагрузкаНоменклатурыИзКонфигурацииSORCКафе` are also
byte-for-byte identical. Their expanded native diagnostics and remediation remain
unresolved. The 51 no-reference diagnostics have not been waived or declared safe.

There are no genuine versions of the ten missing BSP modules in this reference to
restore. The earlier hypothesis that they were omitted from our repository copy
is superseded by this comparison. Do not add dummy modules, remove their callers,
or copy unrelated library versions merely to silence the full configuration check.

## Metadata and implementation scope

- Reference: **760** root declarations. Working export: **766**. The six additions
  are exactly the Booking HTTP service, `ОнлайнБронирование` common module, three
  online-booking registers and the restricted service role.
- All **760 root metadata UUIDs match**. Existing catalog, document and register
  root-definition XML files are byte-for-byte identical. The new registers add
  independent storage; no existing business-object definition was replaced.
- Both compatibility settings are `Version8_3_4`.
- Every original BSL module is present in the working tree. Only the seven
  booking/staff locking and price-preservation modules plus the previously
  developed Reports discovery module differ. The production copy is not copied
  over the current implementation.
- One existing HTTP metadata difference is recorded: the `booking_references`
  URL template/method under Reports uses production UUIDs
  `ecc28040-e15b-407a-9a48-e24b7d75c18c` /
  `7cb648b2-3b17-4387-aa4f-94fc6babbea8`, while the prior task-01 source uses
  `00b0af5b-b864-48b9-bd04-b72e8b7bc826` /
  `eae1d961-52a5-4979-bd8c-430a10d9631f`. The Reports service root and its three
  original report routes retain their UUIDs. This is an existing discovery-revision
  difference, unrelated to the 23 BSP errors; no UUID reconciliation was applied
  as part of this comparison.

The previous local import-compatibility repairs (register folder, static template
and form colors) remain in the working source. The full configuration check still
fails. No production deployment, runtime pass or task completion is inferred from
metadata/source equality.

## Next development validation

The latest owner check has no booking-module compiler diagnostics. With the
production source baseline now established, continue the already-authorized work
in the **isolated file base**: apply the development database configuration update,
then start Enterprise and exercise setup/service/staff workflows described in
[BOOKING_RUNTIME_RESULTS.md](BOOKING_RUNTIME_RESULTS.md). Capture any database-update
or startup error before proceeding to reservation scenarios. This does not waive
or turn the known full-check failures into passes.

The owner has not yet reported a development database update or startup result.
Booking runtime, least-privilege access, posting, price preservation, concurrency
and timeout recovery remain unrun. The full-check baseline defects remain an
explicit completion limitation; task 02 is unchecked and uncommitted.
