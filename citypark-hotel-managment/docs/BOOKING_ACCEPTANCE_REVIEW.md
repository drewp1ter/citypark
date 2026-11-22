# Task 02 acceptance review — 2026-11-22

Status: **task 02 complete under the owner-approved baseline exception
(2026-11-22)**. The owner explicitly stated: “I accept the baseline exception.”
Full Designer checking remains failing on the documented pre-existing production
BSP/reference/form diagnostics; it is not reported as clean. The bounded runtime
results and limits below are retained. Task03 and production deployment are not included.

The chronological [runtime record](BOOKING_RUNTIME_RESULTS.md) retains fixture
IDs, outcomes, failures, corrections and cleanup. Old pending notes there are
historical and superseded only by explicit later results.

## Required scenarios and evidence

| Scenario | Recorded outcome and practical limit |
| --- | --- |
| BookingValid | All five website room types, capacities1/2/4, native room allocation, configured times, guest counts, backend prices and Reserved movements verified. Synthetic booking cleanup recorded. |
| FullStayOneRoom | Alternating twin-room fragments: individual nights available, full interval unavailable; create409, no orphan booking/counterparty. Three later-future stays on sole room201 each remain blocked after subsequent posts; gaps available; spanning interval unavailable. Native movements inspected; all three canceled and HTTP release/recovery verified. |
| AdjacentStays / OverlapVariants | Recorded advisory overlap/adjacency variants; capacities and date/JSON validation. Leap2028, tomorrow/today/past, >90-night/>6-month dates exercised. Storage boundary3999 accepted;4000/9999 now400 after applied date fix. Native minimum/maximum15,2 price preservation and overflow rejection verified. Moscow-midnight predicate evaluated in native debugger, combined with live today/tomorrow HTTP checks; no real HTTP clock-crossing was run. |
| WebsiteStaffRace | Bounded staff-wins run: staff paused after lock, competing direct POST pending19.092s, then409; native winner and owner zero-orphan inspection. Cleanup verified. This does not establish every scheduling interleaving. |
| ConcurrentRoomMove | Explicitly excluded by owner. Failed three-way attempts retained as history; no pass claimed. |
| DuplicateKeyRace | Simultaneous identical POSTs return201/200, sameUUID, owner one-booking/one-counterparty inspection. |
| KeyPayloadConflict | Each changed stay/contact/price field409; normalized/reordered same request200 exact. Invalid fields/types/JSON/key/body and money overflow rejected. |
| PostingRollback | Owner forces Отказ inside posting; generic503, key404, availability released, owner zero booking/counterparty/movements. No permanent fault injection. |
| CommitTimeoutRecovery | Relay observes upstream201 but drops client response; GET/replay200 sameUUID, owner one document/counterparty/key. Canceled; recovery remains immutable. |
| CancellationUndoRace | Bounded undo-wins run with pending competing POST then409, key404, owner zero orphans and restored original Reserved. Serial occupied undo refusal also recorded. Inverse concurrent winner not separately orchestrated. |
| RepostSelfExclusion | Booking/check-in repost, own/basis exclusion, conflicting edit rollback, unpost/repost, checkout and checkout undo verified. One-staff move/API old-room release tested. Two-staff variant remains excluded. |
| BackendPriceAtCheckIn | Backend4000 vs ordinary4500 preserved; explicit booking/check-in staff edits persist; invalid edits rejected; canceled dialog unchanged. Checkout preserves saved rate, partial stay and staff override; posted charges and undo checked. Original API snapshots unchanged. |
| UnauthorizedAccess | Anonymous/wrong-password401; unrelated role403 all methods; no-store/private generic errors verified. |
| RestrictedRole | Service-only create/quote/recover pass; Reports403. Native effective rights deny settings/prices/cancel/checkout writes. Ordinary staff keyRead=true, Update=false, settingsUpdate=false, no FullRights/service role or privileged mode. No actual staff credential HTTP attempt or deliberately denied register write was made. |

## Final source and validation

- Owner full export `htl.zip`:577 BSL files byte-identical at18:28; XML UUID
  inventories preserved. [Nine serialization differences reviewed](BOOKING_EXPORT_RECONCILIATION.md).
- Subsequent small date-storage fix applied by owner;14 HTTP boundary probes pass
  and six existing-result recovery/replay checks pass. That one-file patch
  supersedes the corresponding BSL in the18:28 archive; no second full export is claimed.
- Export layout766 roots/custom-color scan and CRLF-aware diff checks pass.
- Owner final check screenshots still show baseline BSP/reference/form issues,
  consistent with the earlier23 compiler errors,51 no-reference diagnostics and
  two form issues. No booking compiler diagnostic visible. Screenshots do not
  establish an exact fresh text-log comparison or a clean full check.
- [Production baseline](BOOKING_PRODUCTION_BASELINE.md) includes matching source
  defects and owner-supplied native production check screenshots19:16–19:17.
  Visible diagnostics match the development baseline. No production access or
  writes by the agent; no exact fresh text-log/check-options comparison claimed.
  Do not add placeholder BSP modules to silence these failures.

## Final scoped review completed (2026-11-22)

Owner confirms debugging finished. Booking HTTP validation/transaction/recovery,
shared room locks, staff document hooks, price preservation, checkout mapping and
recordset immutability were reviewed. No additional blocking implementation
finding identified in this review. Existing checkout exception handling is retained;
normal checkout/undo was exercised, not every failure/interleaving. This is not
an exhaustive concurrency or security certification.

Reviewed commit scope is95 changed/new/deleted paths, with12 byte-identical
accumulation-register moves. Excludes user archives, screenshots, local base/log
inputs and sibling projects. The reviewed manifest is retained outside the repository at
/tmp/citypark-task02-commit-manifest.json; only those paths belong in the task02 commit.

Final local results:

- Service role validator:9 checks, pass.
- Six new metadata objects:6 passed,0 errors; each warns format2.5 predates
  the validator's tested2.17–2.21 range. Native import/runtime evidence is separate.
- Export layout:766 declared root objects present; custom-color scan clean.
- All58 changed/new XML files parse; both Python scripts compile in memory.
- Tracked XML/BSL BOM and existing line-ending pattern preserved, including
  embedded lone-LF text already present in original files.
- Twelve moved register files match HEAD bytes. Thirty-seven existing form/template
  XML files outside the two command forms differ only in serialized color nodes.
- Known development credential/raw authorization and private-key signature scan
  found no matches in proposed existing files. No archive/image/base dump in scope.
- CRLF-aware diff check passes. No new runtime test or production write was run
  in this review.

## Owner-approved baseline exception and closure (2026-11-22)

The owner explicitly stated: “I accept the baseline exception.” Treat the known
Designer BSP/reference/form diagnostics, reproduced by the owner's production
check and matched to production source, as pre-existing baseline issues outside
task02. Full configuration checking remains failing; this exception does not
turn it into a clean check or waive newly introduced booking errors.

All bounded runtime evidence/limits in the matrix and the owner's explicit
ConcurrentRoomMove exclusion remain recorded. The midnight check is native
expression evaluation plus live today/tomorrow requests, not a live clock-crossing.
No additional acceptance scope change is inferred.

Debugging finished, final scoped review complete, baseline exception accepted,
and task02 marked complete in PROGRESS.md. Commit title:
`task 02: implement 1C booking HTTP service`. Production deployment and task03
remain outside this completion.

Test history and immutable keys remain in the isolated base. The latest three
future-slice fixtures are canceled and independently released; no active occupancy
from those fixtures remains. Prior partial-checkout draft dismissal and17438 saved
rate restoration were requested but cannot be established by a configuration ZIP.
No claim of live midnight HTTP testing, exhaustive concurrency, actual denied-write
attempts, production validation or production readiness is made.
