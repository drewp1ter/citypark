# Booking service development checks

Status (2026-11-22): **task 02 complete under the owner-approved baseline
exception**. Extensive isolated-development HTTP/native checks, final scoped
review and fixture cleanup are recorded below. Full Designer checking still
fails on documented pre-existing production issues; the owner explicitly stated:
“I accept the baseline exception.” No clean full-check claim is made.
See [final acceptance review](BOOKING_ACCEPTANCE_REVIEW.md) and
[source export reconciliation](BOOKING_EXPORT_RECONCILIATION.md).
Older dated pending statements below describe the state at that time; consult
later evidence and owner scope decisions before treating them as current gates.
Task03 and production deployment are outside this completion.

## Environment and current evidence

The owner selected owner-run Windows commands and confirmed that the file base
identified in local `1c.txt` is isolated. The owner also confirmed that the
`CityParkHotelDev` HTTP publication points to that base. `1c.txt` is user-supplied,
untracked input and is not part of this change. Its executable is the common
`1cestart.exe` launcher; the version-specific `bin/1cv8.exe` must be located below.
Previously reported platform: **8.3.11.3133, Windows Server 2012**. Confirm the actual
build in Designer before recording a runtime result. No remote execution is configured.

Local evidence:

- New metadata passes the repository metadata validator with its warning that
  format 2.5 is below its tested 2.17–2.21 range. Generated modern history callbacks
  were removed to match the surrounding export; existing UUIDs are preserved.
- The new service role passes the role validator. Validation of the modified
  `АдминистраторГостиницы` role reports the existing `AllFunctionsMode` right as
  unsupported by the validator. This old platform right is unchanged; Designer
  acceptance on 8.3.11 is required rather than deleting it to satisfy a newer tool.
- XML identity/reference checks and CRLF-aware diff checks are local structural
  checks. They do not compile BSL or establish locking, rights or posting behavior.
- Initial HTTP-checker syntax-only evidence was superseded by the subsequent
  development HTTP executions and native document/register inspections below.

## Import failure and layout repair (2026-11-17)

The owner's first full XML import failed with `Неизвестный объект метаданных`
for existing accumulation registers, including `ВзаиморасчетыЗаПроживание`,
`ОстаткиТМЦ`, `ДенежныеСредства`, `Продажи`, `ВзаиморасчетыСПодотчетниками`,
`ВзаиморасчетыСПоставщиками` and `ПрочиеЗатраты`.

Confirmed cause: their seven XML definitions and five associated module/form
files were stored in `.codex/AccumulationRegisters/`, outside the configuration's
required root `AccumulationRegisters/` directory. All 12 existing files were
moved to the required directory without changing a byte; SHA-256 comparisons
before/after match. No register was recreated or assigned a new UUID.

`python3 scripts/check-export-layout.py` reproduced the seven missing-object
errors before the move, then passed all **766** root declarations after it. The
seven metadata definitions pass the local metadata validator, with its existing
format-2.5 compatibility warning. Repeat this layout check before packaging or
loading the export. These results do not establish successful Designer import.

Cancel the failed import, extract `citypark-hotel-booking-dev-fixed.zip` into a
**new empty directory**, and retry loading from its `citypark-hotel-managment`
folder. Confirm `AccumulationRegisters` is next to `Configuration.xml`. A new
archive is supplied rather than overwriting the owner's earlier ZIP/7z files.
The configuration-only archive excludes `.codex`, documentation, scripts and
local setup inputs; those remain in the working tree. Its CRC/contents and all
766 root XML declarations were checked before handoff. Import and subsequent
Designer/runtime scenarios remain unrun until the owner reports their results.

## Template color import repair (2026-11-17)

The next owner-run import failed while reading `Template.xml`: the platform could
not convert `{http://v8.1c.ru/8.1/data/ui/style}ИзмененноеЗначениеРеквизитаЦвет`
to `Color`. The reference occurs in the existing object-version comparison
spreadsheet `InformationRegisters/ВерсииОбъектов/Templates/СтандартныйМакетПредставленияОбъекта/Ext/Template.xml`.
A scan of spreadsheet templates found eight references to four configuration
style items, all in this template.

Replaced these eight color nodes with the exact values already declared by their
style items: web `Blue`, `LightCyan`, `LightPink`, and RGB `#CCFFCC`. Web palette
values carry an explicit namespace declaration on the color element. Built-in
style references and the four style-item metadata files remain unchanged. XML
node comparison confirms that only the eight color values changed; BOM and CRLF
were preserved. This preserves the declared appearance, but future changes to
those four style items will no longer automatically affect this template. Review
the added/changed/deleted value colors in the object-version comparison after a
successful import; that visual regression check is **unrun**.

The repository MXL structural validator passes (it also passed the original
file, so this is not proof of XDTO acceptance). All 766 root metadata declarations
still pass the layout check. Owner retry in Designer 8.3.11.3133 is required.
`citypark-hotel-booking-dev-fixed-v2.zip` includes this template correction and
the accumulation-register layout repair. Extract it into a new empty directory,
then retry from its `citypark-hotel-managment` source folder. Earlier archives
are retained and do not include this correction. Task 02 remains unchecked.

## Form color import repair (2026-11-17)

The next owner-run import failed in `Form.xml` on the same XDTO conversion class:
`{http://v8.1c.ru/8.1/data/ui/style}ПоясняющийТекст` could not be converted to
`Color`. The previous template-only scan did not cover forms.

A full XML scan found **140** remaining references to five custom color style
items in **36** forms: `ПоясняющийТекст` (128), `ФонУправляющегоПоля` (8),
`НедоступныеДанныеЦвет` (2), `ПросроченныеДанныеЦвет` (1), and
`РезультатУспехЦвет` (1). Replaced all of them with their exact declared values:
`#455185`, `#FFE8B3`, `web:LightGray`, `#B22222`, and `#008000`, respectively.
The web namespace was already declared in the affected forms.

XML-node comparison confirms only color text values changed; form structure,
attributes, IDs, BOM and original line endings are preserved. This includes two
conditional-appearance `value` elements typed as `v8ui:Color`. Built-in style
references and all style-item metadata remain unchanged. As with the spreadsheet
repair, changing these custom style items later will no longer automatically
update the affected static form properties. UI/conditional-appearance regression
checks after successful import remain **unrun**.

Used the repository form validator before and after each change. All 36 runs
return the same pre-existing errors for export format 2.5 (the tool reports old
form controls as unavailable before format 2.10). Captured diagnostics with
`MaxErrors=5` are identical before/after; this is **not** a validator pass or native
Designer acceptance. The tool warns that its tested formats are 2.17–2.21.

Extended `scripts/check-export-layout.py` with a project-specific legacy custom
color reference guard. A regression fixture reproducing the screenshot's
`style:ПоясняющийТекст` fails the guard; its declared color value and built-in
styles pass. The real export passes all 766 root declarations and has no remaining
`style:` references to any configuration-defined color item. This guard does not
claim to validate all XDTO types or compile forms/BSL.

`citypark-hotel-booking-dev-fixed-v3.zip` contains this complete form-color repair,
the template repair and the accumulation-register directory repair. The archive's
CRC, root declarations and current file contents were verified. Extract it into a
new empty directory and retry from `citypark-hotel-managment`. Native import,
configuration checks, database update and runtime acceptance remain pending.

## Owner import progress after v3 (2026-11-17)

The owner supplied a new Designer screenshot showing the XML load's warnings
dialog with **Принять enabled**, rather than the earlier fatal XDTO conversion
errors. This establishes progress to the warning/acceptance stage, not completed
import, compilation, database update or runtime acceptance.

The six displayed warnings concern an existing organization-form help-image link,
one additional-column declaration for `ПользовательИнфБазыРоли.Роль` in the
external-user form, and four additional-column declarations for `ВнешнийВид.Ключи`
in the user-settings forms. The relevant `AdditionalColumns` declarations are
present unchanged in HEAD; they were not introduced by booking or the color fixes.
They may affect those existing administration forms and are not claimed harmless
or runtime-tested. No warning-source declarations were removed to silence them.

Next owner step: accept this import in the isolated base, then run the full
configuration check including server modules and managed forms. Capture the
result before updating the database configuration. The owner has not yet
reported accepting this dialog or a successful configuration check.

## Owner configuration-check failure (2026-11-17)

The owner reached configuration checking after v3 and supplied a readable UTF-8
`config-check.txt`. It reports 24 compiler errors: one new booking-module date
constructor error, corrected in source to use `12, 0, 0`, and 23 errors in
unchanged BSP administration BSL referencing ten absent common modules. It also
reports 51 no-reference diagnostics and two existing metadata/form diagnostics.
The full check did not pass. See the exact correction, baseline evidence and
missing-module counts in [BOOKING_CONFIGURATION_CHECK.md](BOOKING_CONFIGURATION_CHECK.md).
The small date-fix patch is ready for owner recheck; no new Designer pass,
database configuration update or runtime acceptance is claimed. Task 02 remains
blocked on those checks and genuine missing dependency recovery.

## Updated owner log: four string-split errors (2026-11-17)

The owner replaced `config-check.txt`; rereading confirms four new-module
`СтрРазделить` compiler failures and the same 23 BSP compiler failures, for 27 total.
The date-constructor error is no longer reported. Configuration compatibility is
8.3.4, so replaced all four native split calls with the existing server helper
`СтроковыеФункцииКлиентСервер.РазложитьСтрокуВМассивПодстрок`, explicitly preserving
empty parts using its inverted skip-empty flag (`Ложь`). No compatibility setting
or library module was changed. The small compatibility patch includes the date
fix and supersedes the date-only patch. See [the updated check report](BOOKING_CONFIGURATION_CHECK.md).
Corrected-module Designer recheck, the full configuration check and runtime
scenarios remain pending; existing BSP errors remain unresolved.

## Latest owner check: no booking compiler diagnostics (2026-11-17)

The owner supplied another updated log (23,935 UTF-8 bytes). The date and all four
string-split diagnostics have cleared; the run reports no new booking-module
compiler diagnostics. The same 23 BSP missing-module errors, 51 no-reference
diagnostics and two existing form/metadata diagnostics remain. This is bounded
native compilation evidence for the booking fixes, not a successful full
configuration check or runtime acceptance. The missing dependency baseline is
still required; task 02 remains unchecked and uncommitted. See
[the latest check analysis](BOOKING_CONFIGURATION_CHECK.md).

## Production source baseline confirmed (2026-11-17)

The owner supplied `hotel-orig/`. All ten missing BSP common modules are absent
from that reference too; the ten affected BSL form modules and all 23 logged
source lines are identical. The two existing metadata-diagnostic form XML files
also match. All 760 existing root UUIDs match; the working tree adds exactly the
six planned booking objects. The Reports discovery endpoint has a separately
recorded prior identity/revision difference. See [BOOKING_PRODUCTION_BASELINE.md](BOOKING_PRODUCTION_BASELINE.md).

The full check remains failing, with no booking compiler diagnostics in the latest
owner run. Continue the already-authorized isolated-base database update/startup
and runtime checks with this explicit baseline recorded; this does not mark the
full check passed or waive any task completion gate. Database update and startup
results are not yet reported. No production operation was performed.

## Owner F7 restructuring warnings (2026-11-18)

The owner pressed F7 and supplied the information-restructuring dialog. It shows
two non-unique-code warnings for `Справочник.СтатьиДвиженияДенежныхСредств`:
`00000000005` and `00000000006`; the **Принять** button is enabled. The source
predefined-data definitions assign those codes to the `Приход` and `Расход`
folders respectively. The screenshot does not identify all duplicate records or
explain when/how the duplicates arose.

Compared the catalog root XML and its `Ext/Predefined.xml` against `hotel-orig`:
both are byte-for-byte identical. Code settings are string length 11, whole-catalog
series, uniqueness checking and automatic numbering enabled. All predefined IDs
and codes are unchanged; no catalog source or data edits were made for this step.
This is evidence of duplicate-code detection in the isolated base during the
update, not proof that duplicate records have been inspected or corrected.

Next owner step in the already-confirmed isolated base: accept the restructuring
and report its result; if it completes, start Enterprise and report any startup
error. Keep the financial catalog records intact for this booking validation;
any data cleanup needs separate record/reference inspection. The owner has not
yet reported acceptance, successful database update or startup. No production
operation or booking write was performed by the agent.

## Owner Enterprise startup and web-publication probes (2026-11-18)

Following the F7 stage, the owner reports: “app was started without errors.” Record
**Enterprise startup: PASS (owner report)**. This is not a pass for booking setup,
service access, posting or any of the named runtime scenarios; the database-update
acceptance was not separately evidenced beyond the owner's reported sequence.
The known full-check baseline defects remain recorded.

The agent then performed unauthenticated read/advisory HTTP probes, with redirects
disabled, a 10-second timeout, no credentials, no response-body logging and no
reservation-create request:

| Development route | Method | Observed status |
| --- | --- | --- |
| `/CityParkHotelDev/ru_RU/hs/booking/v1/availability` | POST with future stay | 404, HTML |
| `/CityParkHotelDev/hs/booking/v1/availability` | POST with empty body | 404, HTML |
| `/CityParkHotelDev/ru_RU/hs/reports/v1/booking-references` | GET | 500, plain text |

These are failed routing/publication diagnostics, not authentication or booking
acceptance passes. A 404 does not alone distinguish unpublished service, wrong
publication routing or stale configuration. The Reports 500 was not diagnosed;
response bodies were intentionally omitted. Desktop startup does not establish
that the web publication's service/session setup is healthy.

Next owner step: in Designer, open **Администрирование → Публикация на веб-сервере…**,
select the existing **CityParkHotelDev** development publication and verify the
**HTTP сервисы** tab includes enabled **Booking**. Apply the development
publication's settings, then recheck the service route. Do not redirect that
publication to the production base. The operator workflow is documented in
[official 1C publication instructions](https://its.1c.ru/db/content/updinfo/src/arautomation20/2.5.15.40/index.htm).
Publication success is not yet reported. After the route is available, continue
dedicated-user/role and room-settings setup below before the authenticated HTTP
checker. Settings remain disabled until reference verification.

## Owner publication retry: native HTTP 500 (2026-11-18)

The owner reported completing the publication step and requested a retry. Repeated
unauthenticated development probes now return **500** for Booking availability
both with and without the locale prefix (previously 404). The existing Reports
reference endpoint also returns **500**. All responses are plain text, with no
WWW-Authenticate challenge or API no-store header. No credentials were supplied,
no redirects followed, no reservation-create route called and no raw response
body persisted or logged.

The native error response is 360 bytes and references **8.3.13.1513**. Safe
classification detects infobase-connection/version/use-required wording. It does
not identify the loaded web-extension build. The response is not the Booking
handler's JSON error envelope, so platform/publication/session startup must be
resolved before testing the HTTP contract. A web-extension/infobase version mismatch
is the current inference, not a verified DLL/configuration diagnosis. The observed
server header is Apache/2.4.34 (Win32).

Requested the owner's read-only Apache `LoadModule` line containing `wsap24.dll`,
plus confirmation whether that Apache instance also serves production. This is
needed to identify the actual platform DLL and scope any repair: changing a shared
Apache module or restarting a shared instance can affect other publications. No
module path, server configuration or production setting was changed. The installed
platform list includes 8.3.13.1513, but the currently loaded web-extension version
and post-startup desktop version have not been supplied.

Official platform guidance requires publishing with the version used for the
information base and matching the web-extension bitness to the web server:
[1C general publication procedure](https://kb.1ci.com/1C_Enterprise_Platform/Guides/Administrator_Guides/1C_Enterprise_8.3.22_Administrator_Guide/Chapter_8._Setting_up_web_services_for_1C_Enterprise/8.3._Publication_types/8.3.1._General_publication_procedure/).
HTTP authentication, availability, reservation and remaining runtime scenarios
have not passed. Task 02 remains open; the next dependency is the actual web
runtime configuration, not another application-code patch.

## Latest publication retry: authentication challenge (2026-11-18)

At the owner's next retry request, unauthenticated Booking availability probes
(with and without `/ru_RU`) and the existing Reports reference GET all return
**401 with WWW-Authenticate**, rather than the prior 500. No credentials were
supplied or guessed, no redirects followed and no reservation-create endpoint
called. No application-code change was required for this observation.

This verifies that these anonymous requests are now challenged at the publication
boundary. It does not establish successful authenticated infobase connection,
Booking handler dispatch, configuration/availability or version alignment behind
the authentication layer. The previous version-related 500 is no longer exposed
to anonymous requests; its absence after a valid login is not yet verified. The
full `UnauthorizedAccess` scenario (wrong credentials/unrelated role) remains
incomplete.

Next: create/map the dedicated development service user and assign
`ОнлайнБронированиеСервис` (without full/hotel-administrator rights). Keep its
password in the owner's Windows environment. Then configure a verified room-type
record and run the authenticated availability/checker workflow below. No user/
role setup, authenticated call, room setting or reservation result has yet been
reported. Task 02 remains open.

## Authenticated probes: reports pass, booking role denied (2026-11-18)

The owner explicitly authorized Basic authentication using the supplied `ws`
account. The password was used only in request memory; it is not stored in this
report, the repository, artifacts or diagnostics. Requests target only the
owner-confirmed isolated CityParkHotelDev base, with redirects disabled and
bounded timeouts/responses. No reservation-create request was sent.

| Authenticated probe | Result |
| --- | --- |
| Reports `GET /v1/booking-references` | 200 JSON, schemaVersion 1, no-store, 12 rooms |
| Booking `POST /v1/availability` | 403 JSON, FORBIDDEN, no-store |
| Booking `GET /v1/reservations/{fresh-random-UUID}` | 403 JSON, FORBIDDEN |

The lookup path checks the explicit booking role without consulting room settings.
Its FORBIDDEN response therefore isolates the current blocker to the service role
being unavailable to this authenticated account, rather than a room-setting
responsible-user mismatch. Assign `ОнлайнБронированиеСервис` (display name
`Сервис онлайн-бронирования`) to the intended development booking account and
recheck; no role was modified by the agent. A least-privilege account/complete
RestrictedRole scenario is not claimed passed merely because Reports succeeds.

The authenticated Reports response was also compared to the frozen contract:
all **12 unique room UUIDs**, printed numbers, room-type UUIDs and both room/type
non-deletion flags match exactly. This does not validate live inventory,
organization selection, building relationships, noon constants or booking prices.
The earlier native version-related 500 is absent from these authenticated probes;
no loaded DLL/build comparison was performed or inferred.

Next owner action: grant the intended account the explicit role, then retry
availability. Room-settings and responsible-user setup may still be needed after
that gate is cleared. Authentication and reference reads now have native HTTP
evidence; availability, creation/replay, register and concurrency acceptance remain
pending. Task 02 is not complete.

## Authenticated retry: role accepted, runtime exception unresolved (2026-11-18)

After the owner requested another retry, the same authorized account returned:

| Probe | Observed result |
| --- | --- |
| GET recovery with a fresh UUID | 503 `BOOKING_UNAVAILABLE`, JSON, `Cache-Control: no-store` |
| POST availability, single-balcony, 2026-12-01 through 2026-12-03, one guest | 503 `BOOKING_UNAVAILABLE`, JSON, `Cache-Control: no-store` |
| GET recovery with `not-a-uuid` | 400 `INVALID_REQUEST` |
| POST availability with malformed JSON | 400 `INVALID_REQUEST` |

The invalid-input probes confirm the explicit role gate passes and handler input
validation runs. The valid-key recovery path does not read booking settings, so
missing/disabled room settings alone cannot explain both 503 responses. The
handler deliberately hides unexpected exception details. Register access or
another runtime failure remains possible; the actual exception has not been
observed. Do not grant full rights or change code based on an assumed cause.

Next evidence: capture the exception raised during fresh-key recovery in the
isolated base using Designer's debugger (break on errors) or an existing
technological-log EXCP event. In the local source, inspect `ПолучитьРезультат`
in `CommonModules/ОнлайнБронирование/Ext/Module.bsl`, especially record-manager
creation, key assignment and `Прочитать()`. Supply the failing source line and
exception text with any credentials/contact data removed. A configuration check
log cannot identify this caught runtime exception.

No reservation creation, configuration mutation or credential persistence was
performed. Booking acceptance and task 02 remain incomplete.

## Temporary diagnostic response: read-only lookup parameter (2026-11-18)

Owner temporarily changed the development HTTP handler's unexpected-error message
to the original exception description. Authenticated fresh-key recovery returned
503 with `Индексированное значение доступно только для чтения`. Availability for
single-balcony, 2026-12-01 through 2026-12-03, one guest returned 503 with the
explicit exception text `BOOKING_UNAVAILABLE`. These are distinct failures.

Source inspection found that `ПолучитьРезультат(Ключ)` reassigns its by-reference
parameter, while its HTTP caller passes the indexed read-only URL-parameter
collection value directly. Changed the signature to
`ПолучитьРезультат(Знач Ключ)` so normalization assigns only to a local copy.
The source change preserves all other bytes, including BOM/CRLF. Native recheck
and HTTP verification of the fix are pending; expected fresh-key result is 404
`RESERVATION_NOT_FOUND`. Availability's explicit setup/validation failure remains
to be isolated after lookup works. No reservation was created.

The diagnostic handler change exists only in the owner's isolated database; the
repository handler retains the generic public message. Restore the owner's
generic message after debugging. No diagnostic messages containing real guest
data should be returned, and production is outside this test's scope.

## Owner update retry: same lookup error, failing line still unknown (2026-11-18)

After the owner reported updating the configuration, one authenticated fresh-key
GET still returned 503/no-store with `Индексированное значение доступно только для
чтения` in approximately 0.07 seconds. No reservation creation request was sent.
The value-copy source fix is not yet verified in the running web session. Stale
runtime code or another failing assignment remains possible. Requested a temporary
development diagnostic message using `"DEV-DIAG-2: " + ОписаниеОшибки()` in the
unexpected-error branch, to identify the source line and confirm the handler
revision is active. Keep this diagnostic out of the repository/public handler and
restore the generic message after diagnosis.

## Debugger attached: corrected lookup reaches expected not-found branch (2026-11-18)

The owner supplied a debugger screenshot showing the active signature
`ПолучитьРезультат(Знач Ключ)` and execution stopped at line 83 on the intentional
`RESERVATION_NOT_FOUND` exception. This confirms the value-copy fix is active,
the debugger attaches, and the fresh-key lookup successfully reaches the empty
register-result branch. The HTTP client had timed out after 15 seconds while
execution was paused, so the completed HTTP 404 response itself is not yet
verified. No reservation creation was requested.

After the owner resumed execution, an authenticated availability POST for
single-balcony, 2026-12-01 through 2026-12-03, one guest also timed out after 15
seconds. Await the debugger's highlighted line/exception to identify the actual
setup failure; a timeout alone is not proof of the stopping location.

## Availability breakpoint identifies missing/disabled setup (2026-11-18)

Owner debugger screenshot shows `BOOKING_UNAVAILABLE` at line 188 in
`ПолучитьНастройки`, under `Если Не Запись.Выбран() Или Не Запись.Включено`.
The preceding request was single-balcony, 2026-12-01 through 2026-12-03, one guest.
This confirms the settings record is absent or disabled; the screenshot does not
distinguish which. Record creation/enabling under a development administrator is
still pending. Organization, responsible-user mapping, room/type/building and
noon settings must be verified before enabling. HTTP availability and reservation
acceptance remain incomplete; no reservation creation request was sent.

## Configured check-in/out policy revision (2026-11-20)

Owner reports production check-in is 13:00 and authorized updating logic and
contract to use hotel configuration. Production check-out remains unconfirmed;
the service reads its configured value without assuming noon. This section
supersedes all earlier noon-only setup instructions and acceptance requirements.

The booking module now parses date-only input at midnight, validates date order
and calendar-night pricing, then applies the existing hotel helpers
`УправлениеЗаселением.РассчитатьВремяЗаселения` and
`ПолучитьДатуИВремяВыселения` once per availability/new reservation request.
Removed the noon-only guard. Queries, document dates and period rows receive the
same derived timestamps; price quantity remains calendar nights. Idempotent
replay still returns the saved result before settings/times are reapplied.
The previously confirmed lookup value-copy fix is included.

Local checks: BOM/CRLF preserved; export layout/color validation and calendar
arithmetic examples checked. These do not compile or run the changed BSL.
Designer compilation/update and all affected runtime assertions remain pending.

Owner handoff: replace only the ОнлайнБронирование common-module text from
`citypark-hotel-booking-configured-times-fix.zip`, check in Designer and update
only the isolated base. The archive contains the complete updated common module,
including prior date/string-split/lookup fixes; it does not replace the temporary
HTTP diagnostics. Resume any paused request before updating. Restore development
constants to verified production values if they were changed during debugging.
Do not modify production. Restore the HTTP handler's generic unexpected-error
message after diagnosis.

Required native cases: 13:00 arrival/12:00 departure for one and multiple nights;
leap-day and year transitions; changed constant minutes/seconds; adjacent stays
with a gap, equal boundaries and an actual overlap; one physical room for the
whole interval; exact document/period/register timestamps and unchanged backend
prices through staff check-in. Change constants after creation and verify same-key
replay/recovery returns the original result without modifying the document; a new
key must use the current settings. No reservation writes were performed by the
agent during this revision. Task 02 remains unchecked.

## Room-validation query failure after configured-time patch (2026-11-20)

Owner reports applying the configuration. Availability request for single-balcony,
2026-12-01 through 2026-12-03, one guest timed out after 15 seconds. Debugger
screenshot identifies a query-execution exception at line 230 in `ПолучитьНомера`.
Execution reached room validation after the settings/user and time-application
steps. The popup contains only the outer method-call error; its nested cause has
not been captured. This is not successful availability or timestamp acceptance.

Source review found an invalid hierarchy-dependent condition: the query uses
`НЕ ЭтоГруппа`, while `Catalogs/ГостиничныеНомера.xml` declares
`Hierarchical=false`. Removed only that condition; room ID/type, deletion marks,
building/complex deletion checks and duplicate detection are retained. No role
expansion or metadata changes were made. This fixes a source/schema mismatch;
confirmation that it is the observed exception's cause requires native retry.

The module-only `citypark-hotel-booking-room-query-fix.zip` includes this fix and
all prior compatibility/lookup/configured-time changes. Module BOM/CRLF and archive
bytes checked; export-layout validation passes. Owner Designer update/retry is
pending. If it still fails, capture the nested query error via the temporary full
HTTP diagnostic after resuming execution. No reservation creation was sent.

## Room query executes; configured room fails validation (2026-11-20)

After owner reports applying the query fix, availability for single-balcony,
2026-12-01 through 2026-12-03, one guest again timed out while debugging. Owner
screenshot shows execution stopped at the intentional BOOKING_UNAVAILABLE on
line 231, after `Запрос.Выполнить().Пустой()`/duplicate detection. The revised
query without ЭтоГруппа is visible and no query-execution exception is reported.
For this one-room type, the result array is initially empty and duplicate detection
cannot explain rejection of its sole UUID; the query therefore returned no row.
Actual mismatch (room/type/deletion or owner chain) has not been observed. Next:
inspect the configured room reference/type and deletion flags, including its
building/complex, without weakening validation. No reservation was created;
successful availability and full native acceptance remain pending.

## First successful authenticated availability (2026-11-20)

The owner debugger watch identified room 201's actual 1C type as
`Двухместный (обычный)`, while the setup record contained `Двухместный (малый)`.
Room, building and complex deletion flags were all false. Owner was instructed
to correct the setup record's room-type reference, preserving the room catalog
and the website single-balcony capacity of one guest. Owner reports updating
the room type; no direct database/export comparison was performed by the agent.

Retried POST availability for single-balcony, 2026-12-01 through 2026-12-03, one
guest returned HTTP 200 in approximately 1.69 seconds, `Cache-Control: no-store`,
with `available: true` and the same room type/date/guest fields. This is the first
successful advisory availability check. It confirms the tested path now passes
setup/user/room validation and completes availability. It does not prove exact
13:00/checkout timestamps, physical inventory state, posting, price preservation,
restricted-role acceptance, concurrency or idempotency. All those native checks
remain pending. No reservation creation request was sent.

## Pre-creation checks and JSON storage fix (2026-11-20)

Owner authorized continuing development tests. Fresh probes: anonymous POST
availability returns 401; authorized single-balcony availability for 2026-12-01
through 2026-12-03/one guest returns 200, available=true, Cache-Control no-store.
No reservation creation was dispatched.

Pre-write source review found both idempotency JSON resources, ЗапросJSON and
РезультатJSON, declared String(10), too short to persist the request/result.
Expanded each to String(16384), variable length and nonindexed. Only the two
length text nodes changed; UUIDs, remaining bytes, UTF-8 BOM/CRLF and other fields
are unchanged. Metadata skill validation passes with the existing old-format
warning; export-layout scan passes. A maximum-field-size reference JSON fits the
new bound, including escaped Cyrillic; this is not native persistence testing.
Patch: citypark-hotel-booking-json-storage-fix.zip (one register XML).

Owner must update these resource lengths in development Designer and apply F7
before any reservation test. No production changes. For automated negative tests,
disable stop-on-error and ordinary breakpoints temporarily so intentional 400,
404 and 409 exceptions do not pause the request. Reenable on an unexpected failure.
Next: default protocol checker, then one synthetic reservation at a test backend
snapshot of 400000 kopecks/night for the two-night fixture, normalized replay,
GET recovery and changed-payload conflict. Retain/record the recovery key before
creation, inspect native document/register state, and use staff cancellation for
cleanup. Remaining native/concurrency tests are still unrun.

## Protocol checks pass; first creation fails in key-record hook (2026-11-20)

Owner reports increasing JSON resource lengths and disabling debugger stops.
Ran the stdlib checker against the isolated development publication, fixture
single-balcony/2026-12-01 through 2026-12-03/one guest, test backend price
400000 kopecks per night (800000 total, two calendar nights). Anonymous denial,
availability, unknown-key 404 and unknown-field 400/error-envelope checks all pass.

Recovery key printed before first creation:
`9ba701ee-86da-4491-a9ac-56253fc632b4`.
Synthetic guest: name `Тест онлайн-бронирования`, email
`booking-check@example.invalid`, phone `+70000000000`. Retain this exact normalized
payload/key for further retry; do not start a fresh-key creation.

First POST creation returned 503. GET recovery returned 404. Retried the exact
same key/payload to capture development diagnostics: 503 at common-module line 67,
`Записать`, inside the record-set `ПередЗаписью` handler; nested common-module
line 451 INVALID_REQUEST is raised while constructing the key UUID. Thus the
manager-backed write presents an invalid key to the record-set hook; the actual
hook filter value was not observed. No guest/booking success response was returned.

Changed only the new-record write path to create an explicit register record set,
set `Отбор.Ключ` to the already validated/locked key, add one row with the same key
and resource values, then write that filtered set inside the existing transaction.
The lookup/replay manager reads, immutable-record guard, key/room locks and rollback
remain intact. This avoids the manager's old-filter handling after reading a
missing key. Owner native compilation/update and successful persistence remain
pending. Patch: `citypark-hotel-booking-key-write-fix.zip`, latest complete common
module, preserving prior fixes and BOM/CRLF. JSON resource length changes are
separate metadata and must remain applied.

After the diagnostic attempt, GET recovery again returns 404 and advisory
availability remains 200/available=true. These are bounded rollback observations,
not proof of no orphan counterparties/documents or a full PostingRollback pass.
Native inspection for failed synthetic attempts is required. No confirmed
reservation, replay/recovery success or concurrency acceptance is claimed.
After owner applies the patch, recover this key first and retry its exact payload;
inspect native document, counterparty, occupancy and saved JSON before cleanup.

## First confirmed reservation and replay/recovery checks (2026-11-20)

Owner reports applying the key-write module fix. Recovered the existing key first:
HTTP 404/no-store. Retried the exact previously documented synthetic payload/key
and received a confirmed reservation with UUID
`4cd27d6c-e52b-11f1-82c4-e46f13aa4449`.
Recovery key remains `9ba701ee-86da-4491-a9ac-56253fc632b4`.
Fixture: single-balcony, 2026-12-01 through 2026-12-03, one guest, RUB,
2 nights × 400000 kopecks = 800000 kopecks (4000 RUB/night, 8000 RUB total).

Assertions passed:

- Successful creation response was gated to HTTP 200 or 201 and exact accepted
  stay/price plus confirmed status. The console summary accidentally used the JSON
  `status` field over its HTTP-status field, so the exact initial success code was
  not retained. Strict first-create-201 acceptance must be captured in a later
  justified creation test; do not infer that assertion from this log.
- Normalized/reordered same-key POST: HTTP 200, exactly the saved JSON result.
  Name whitespace, email domain case and formatted phone normalize to the original.
- GET recovery: HTTP 200, exactly the saved result and UUID.
- Changed guest name with same key: HTTP 409 IDEMPOTENCY_CONFLICT.
- Subsequent availability for the same stay: HTTP 200, available=false.
- All observed authenticated responses carry Cache-Control: no-store.

One confirmed synthetic reservation now remains for native inspection. Agent did
not cancel it or delete recovery history. Successful HTTP behavior alone does not
verify all posting/register effects, absence of failed-attempt orphans, service
least privilege, native constant values or concurrency. Owner must inspect room
201, configured arrival/departure timestamps, posted/open status, two-night daily
period rows, exact backend price/guest count, occupancy movements, counterparty
and one complete immutable key/request/result record. Inspect possible earlier
failed-attempt artifacts too. Do not cancel before recording native evidence;
use staff cancellation for cleanup and verify saved-key recovery afterwards.

## Owner document screenshot: configured times and price preserved

Owner supplied screenshot `2026-11-21_00-49.png` of booking ОЛ-00015928 dated
21.11.2026 00:42:37 (displayed server/local time). It shows synthetic counterparty
Тест онлайн-бронирования, room 201, building Корпус "Ситипарк", responsible ws,
blank comment, arrival 01.12.2026 13:00:00 and departure 03.12.2026 12:00:00.
The Стоимость tab has one daily Сутки row with identical period boundaries,
quantity 2,000, price 4 000,00 and amount 8 000,00. Room 201 appears occupied on
the background schedule for the corresponding interval.

This confirms owner-observed booking form values and cost preservation on opening
for the current synthetic fixture, including configured non-noon arrival and
two calendar nights despite 47 elapsed hours. It does not establish posting
status (the visible posting buttons do not prove that), exact occupancy-register
rows, guest count, Проживание table values or immutable-key record count/content.
The screenshot confirms development checkout 12:00; production checkout remains
unconfirmed. Link to the preceding HTTP result UUID is based on fixture identity,
not a reference UUID visible in this screenshot. Native inspection continues;
do not mark BookingValid or BackendPriceAtCheckIn complete from this alone.

## Owner period-row and occupancy-register evidence

Screenshots `2026-11-21_00-50.png` and `2026-11-21_00-53.png` show booking
ОЛ-00015928's Заселение tab with one Сутки period, quantity 2, start
01.12.2026 13:00:00 and end 03.12.2026 12:00:00. This quantity denotes tariff units,
not guest count. The document form menu has no movements command.

Screenshot `2026-11-21_00-56.png` shows the СостояниеНомеров list with the selected
record for booking ОЛ-00015928 dated 21.11.2026 00:42:37, row number 1, room 201,
state Reserved. Arrival is visibly truncated after 01.12.2026 13:0… and departure
after 03.12.2026 11:5…. Thus a corresponding reserved occupancy movement is
owner-observed; exact timestamp seconds and total filtered row count are not yet
verified. Source `InformationRegisters/СостояниеНомеров/Ext/RecordSetModule.bsl`
sets `ДатаВыселения = ДатаВыселения - 1`, so expected inclusive register end is
03.12.2026 11:59:59 for document checkout 12:00:00. This source behavior explains
the visible minute prefix; do not claim screenshot verification of the hidden
seconds. Next: widen date columns/inspect row, then verify one complete recovery
record and native guest count. Native posted flag, rollback-orphan inspection,
check-in price preservation and concurrency acceptance remain incomplete.

## Exact occupancy timestamps confirmed by owner

Screenshot `2026-11-21_00-57.png` widens the date columns of СостояниеНомеров.
Selected booking ОЛ-00015928, row 1, room 201, state Reserved visibly stores
01.12.2026 13:00:00 through 03.12.2026 11:59:59. These exact values match the
13:00 arrival and inclusive register end (document checkout at 12:00 minus one
second). The prior hidden-seconds uncertainty is resolved for this row. The list
is not filtered to the booking, so exclusive total-row count and document posted
flag are not independently established. Next native evidence: filtered immutable
recovery key record, full JSON contents and guest count. Other required runtime
and concurrency scenarios remain incomplete; keep this reservation for inspection.

## Native recovery record and advisory adjacency/overlap checks

Owner screenshots `2026-11-21_00-59.png` and `2026-11-21_01-01.png` show one listed
recovery record for key 9ba701ee-86da-4491-a9ac-56253fc632b4, linked explicitly to
Бронирование ОЛ-00015928 от 21.11.2026 0:42:37, with the synthetic contacts.
Owner then copied both full ЗапросJSON and РезультатJSON in chat. Paragraph marks
in the pasted representation denote line breaks. Request matches the original
fixture including its canonical key, guest, date-only stay and 400000/800000
kopeck price. Result matches saved reservation UUID
4cd27d6c-e52b-11f1-82c4-e46f13aa4449, confirmed status, same stay, guests=1 and
full price fields; no guest contacts occur in the result. This resolves the
native full-JSON/link visibility checks; HTTP replay/recovery equality was already
verified. The document's native guest-count field itself remains unobserved.

Six additional advisory availability POSTs for single-balcony/one guest all
returned HTTP200/no-store with expected results against this active reservation:

| Stay | Expected/observed available |
| --- | --- |
| 2026-11-30 to 2026-12-01 (adjacent before) | true |
| 2026-12-03 to 2026-12-04 (adjacent after) | true |
| 2026-12-01 to 2026-12-03 (identical) | false |
| 2026-12-02 to 2026-12-03 (contained) | false |
| 2026-11-30 to 2026-12-02 (overlap from before) | false |
| 2026-12-02 to 2026-12-04 (overlap from after) | false |

No additional reservation creations or edits were sent. These verify advisory
queries for this fixture, not transactional conflicting-create rejection, equal
clock boundaries, fragmented multi-room inventory or concurrent posting.
Next owner check: from the staff workspace use Заселение → Забронированный номер
and select ОЛ-00015928 to open the filled check-in form, initially without saving
or posting it. Inspect guest count, dates and inherited backend cost before
choosing recalculation/posting scenarios. Full native acceptance remains pending.

## Owner unsaved staff check-in: inherited period and price preserved

Owner screenshots `2026-11-21_01-22.png`, `2026-11-21_01-22_1.png` and
`2026-11-21_01-23.png` show Заселение (создание), blank document number, based on
booking ОЛ-00015928, synthetic counterparty and room201. Dates are
01.12.2026 13:00:00 through 03.12.2026 12:00:00. Проживание contains one Сутки
row for this interval, quantity2; Стоимость contains the corresponding row,
quantity2, price4000 RUB and amount8000 RUB. Thus opening/filling a staff check-in
preserves the tested online period and backend snapshot in the unsaved form.
Forced recalculation, saving/posting and native guest-count assertions remain
pending. The screenshot does not verify that the test backend price differs
from the actual 1C tariff; full BackendPriceAtCheckIn acceptance is incomplete.

Гости is empty: it is a separate guest-registration table, not the numeric
КоличествоГостей attribute. No form control for that numeric attribute exists
in the exported check-in form XML, so its value requires debugger/native data
inspection rather than reading the empty table.

All three screenshots include an equipment connection failure for a TESA
lock device, port-settings error999. Source check-in ПриОткрытии explicitly
initializes equipment. This is observed integration initialization, not a booking
or price-calculation error; successful hardware isolation/operation is not claimed.
No card creation, payment or hardware-setting changes were requested/performed.
Keep the test check-in unsaved/unposted while checking recalculation. Next
controlled UI step: change arrival clock13:00 to13:01 and leave field, then restore
13:00 and leave field; both invoke the date/recalculation handler. Verify final
cost still2×4000=8000 and original dates, then inspect numeric guest count and
choose subsequent native posting/cleanup scenarios. This step is not yet run.

## Owner check-in form after requested recalculation step

Following the requested temporary arrival-clock edit13:00→13:01→13:00, the owner
supplied `2026-11-21_01-27.png`. Check-in remains in creation mode, blank number,
with an unsaved-change asterisk. Final room201, arrival01.12.2026 13:00:00 and
departure03.12.2026 12:00:00 are unchanged. Стоимость still contains one Сутки
row, quantity2, price4000 RUB and amount8000 RUB for the same interval. This is
owner evidence of final price/period preservation following the requested UI
recalculation sequence; the intermediate field value/call trace was not supplied.
Actual 1C tariff price difference and numeric guest count remain unverified, so
full BackendPriceAtCheckIn acceptance remains incomplete.

Owner explicitly directs ignoring TESA for the booking tests. Do not pursue
hardware troubleshooting or settings changes as part of this task. Next native
step is posting this synthetic check-in in the isolated base via Провести, keeping
the form open for any validation error and subsequent occupancy/price inspection.
No save/post outcome is reported yet. Do not trigger payment or checkout while
collecting this evidence; existing reservation/key fixture remains retained.

## Staff check-in receives number; recovery remains unchanged

Following the instruction to click Провести, owner screenshot
`2026-11-21_01-30.png` shows saved check-in ОЛ-00017433 dated21.11.2026 00:42:37,
no unsaved asterisk, room201, original Dec1 13:00→Dec3 12:00 period, and daily
cost2×4000=8000 RUB. Responsible is the staff user МБ Консалтинг. The messages
panel is empty. This is owner evidence of the resulting numbered/saved form;
check-in occupancy movements and explicit posted status still need inspection.
No payment or checkout was requested. TESA troubleshooting remains excluded.

Agent read-only checks after this screenshot: recovery GET for retained key
9ba701ee-86da-4491-a9ac-56253fc632b4 returns HTTP200/no-store and exactly the
original confirmed booking JSON (UUID4cd27d6c-e52b-11f1-82c4-e46f13aa4449,
same dates, guests1 and full price). Availability for the original stay returns
HTTP200/no-store, available=false. Thus staff action has not altered immutable
HTTP recovery and the advisory stay remains blocked; this alone does not prove
which register movement makes it unavailable. Next owner evidence: open the
check-in form's Состояние номеров navigation link and inspect rows for registrar
Заселение ОЛ-00017433, expecting Busy for room201 and the tested boundaries.
Full check-in/self-exclusion/reposting, guest-count and concurrency acceptance
remain incomplete until native assertions are recorded.

## Staff check-in Busy movement observed

Owner screenshot `2026-11-21_01-32.png` shows Состояние номеров navigation within
check-in ОЛ-00017433, with one visible row for this document, room201, state Busy,
line1. Arrival prefix is01.12.2026 13:0… and departure prefix03.12.2026 11:5…;
exact seconds are clipped. This confirms a posted occupancy effect and that the
check-in did not reject its own booking basis for the tested stay. Original
configured dates/prices and immutable HTTP result were separately verified.
Full timestamp seconds on this Busy row, native guest count, actual tariff-price
difference and self-reposting are not yet verified. Next owner step: return to
Основное and click Провести again without edits, then inspect whether the same
single Busy movement remains and no self/basis conflict appears. This is a
bounded RepostSelfExclusion subtest; other edit/unpost/undo paths and concurrency
remain pending. Do not mark the entire scenario complete from this first post.

## Owner same-check-in repost evidence

After the requested repeat Провести without edits, owner screenshot
`2026-11-21_01-34.png` shows the same check-in ОЛ-00017433 and one visible Busy
row for room201, with an empty messages panel. This supports successful
same-document repost/self-exclusion for the fixture; intermediate execution is
owner-driven, not instrumented. Exact Busy timestamp seconds remain clipped.
This does not complete all RepostSelfExclusion edit/unpost/undo variants.

Next owner step: cancel posting of this development check-in and inspect its
occupancy list for zero rows, then check recovery/availability before reposting.
Source review: check-in posting closes/reposts its booking basis; unposting
clears check-in occupancy and settlement movements but does not reopen the
booking. Therefore do not assume unposting restores the earlier Reserved row;
record actual lifecycle state and availability. This is preexisting workflow
behavior and has not been changed here. No new HTTP booking writes were sent.

## Check-in unposted; preparing competing booking fixture

Owner screenshot `2026-11-21_01-35.png` shows zero occupancy rows for check-in
ОЛ-00017433 and an empty messages panel after requested unposting. Agent GET
recovery for original key9ba701ee-86da-4491-a9ac-56253fc632b4 returns HTTP200/no-store
and exactly the original saved result. Availability for Dec1–3 returns
HTTP200/no-store, available=true. This matches the preexisting workflow: check-in
posting closes its booking basis, and unposting does not reopen it. Immutable
creation recovery is not a live booking-status API.

Next test: competing synthetic reservation on the now-free room201/same dates,
then owner attempts to repost original check-in; it must reject the other booking.
New recovery key recorded BEFORE dispatch: `42bdbc5a-a1a8-4485-8993-5aff1d51f6aa`.
Fixture: single-balcony Dec1–3/one guest, name Тест конфликта онлайн-бронирования,
email booking-conflict@example.invalid, phone +70000000001, RUB2×400000=800000
kopecks. Dispatch/result pending. Retain key for recovery; do not generate a new
key on transport failure. After rejection inspection, cancel only the competing
booking through staff workflow, inspect release, then restore original check-in.
This serial conflicting-repost test is not evidence of concurrent race safety.

## Competing booking created: strict 201 captured

The prepared competing fixture key `42bdbc5a-a1a8-4485-8993-5aff1d51f6aa` received HTTP201,
Cache-Control no-store, confirmed reservation UUID`e47d906c-e532-11f1-82c4-e46f13aa4449`,
exact fixture stay/guest count and price. This independently captures strict
first-create201, resolving the earlier console-output collision limitation.
GET recovery returns HTTP200 with exactly the saved result. Same-stay
availability returns HTTP200/available=false. No extra creation keys were used.

Original check-in ОЛ-00017433 must remain unposted until owner tests reposting
against this conflicting booking. Expected: reject posting and retain zero Busy
rows for that check-in, keep competitor Reserved movement and immutable records
intact. Owner should send the validation message. Then inspect and cancel the
competing fixture only through staff workflow; do not delete key records or
reopen the original booking basis as an ad hoc workaround. Test cleanup pending.

## Competing repost rejected, but nested transaction error exposed

Owner screenshot `2026-11-21_01-38.png` shows an attempted Провести of original
check-in ОЛ-00017433. Staff message identifies the competing reservation:
`Номер Reserved с 01.12.2026 13:00:00 по 03.12.2026 11:59:59. Документ:
Бронирование ОЛ-00015929 от 21.11.2026 1:36:58`. Thus competitor UUID
e47d906c-e532-11f1-82c4-e46f13aa4449 is identified by fixture/time as native
booking ОЛ-00015929. A separate modal reports `В данной транзакции уже происходили
ошибки!`. Conflict detection is observed, but a clean conflict-rejection pass is
NOT recorded because of this transaction error. Original check-in zero-movement
state and absence of settlement side effects still require native reinspection.

Agent read-only probes after failure: both original and competitor keys recover
with HTTP200/no-store and their expected UUID/confirmed status. Availability for
the competing stay remains HTTP200/no-store/false. These are bounded checks, not
full rollback proof.

Source review found explicit НачатьТранзакцию/ОтменитьТранзакцию/ЗафиксироватьТранзакцию
inside check-in ОбработкаПроведения, which already runs in the platform's document
write transaction. On occupancy refusal, the handler canceled the transaction
and returned into the platform write path. This is a source-level explanation
for the extra error; native confirmation remains pending. Fixed only this handler:
remove explicit nested begin/commit/rollback, retain shared lock and basis posting,
and return immediately when availability sets Отказ. Platform-managed rollback
then retains atomicity of basis/document/register writes. Later object procedures
and original encoding/BOM/CRLF are byte-identical. No role or HTTP changes.

Patch: citypark-hotel-checkin-conflict-transaction-fix.zip, one file
Documents/Заселение/Ext/ObjectModule.bsl. Owner must apply it to the DOCUMENT OBJECT
module, check/update isolated base in Designer, then retry original check-in
posting while competitor ОЛ-00015929 remains active. Expected: occupancy rejection
without the nested-transaction error, no original Busy/settlement rows, competitor
movement intact and both recoveries unchanged. Afterwards perform competitor
staff cancellation, release verification and original check-in successful re-post.
No cleanup or successful native retest claimed yet. This serial test does not
establish concurrent race safety.

## Fresh conflict test requested by owner

Owner reports the previous conflict test failed and requests a new test. The
previous transaction-error case remains failed/incomplete; latest check-in
object-module patch application and native retry were not confirmed. Earlier
fixtures must remain tracked (original booking ОЛ-00015928/check-in ОЛ-00017433,
competitor ОЛ-00015929 and both recovery keys). Their current native lifecycle
state was not re-inspected and no deletion/cancellation was performed by agent.

Fresh availability for single-balcony, 2026-12-10 through 2026-12-12, one guest
returns HTTP200/available=true. Preparing one new base reservation on these dates.
Recovery key recorded BEFORE dispatch: `7ac6fc48-f7de-4909-a053-c6ba1ab48525`.
Synthetic guest Тест конфликта 2 — исходная бронь,
booking-conflict-2-base@example.invalid, +70000000002, price RUB2×400000=800000
kopecks. Dispatch pending. Owner must apply the latest Заселение DOCUMENT OBJECT
module patch and F7 before testing native check-in posting. A fresh fixture alone
cannot fix the known nested-transaction defect. New competitor will be created
only after this base booking has a check-in that is posted, then unposted.

## Fresh base reservation confirmed

Fresh fixture key `7ac6fc48-f7de-4909-a053-c6ba1ab48525` returns HTTP201/no-store, confirmed
reservation UUID`945c469c-e535-11f1-82c4-e46f13aa4449`, exact Dec10–12 stay/one guest and RUB2×4000=8000
price. One fresh base booking has been created; its native document number is
not yet known. No new competing booking exists for this fresh fixture yet.
Next owner step after latest object-module patch/F7: staff Заселение →
Забронированный номер → guest Тест конфликта 2 — исходная бронь/Dec10–12;
post the filled check-in and provide its number/screenshot. Do not select the
earlier Dec1–3 fixture. Later steps (unpost, competing booking, reject repost,
cleanup) remain pending. Prior failed test is not reclassified as a pass.

## Fresh check-in receives number ОЛ-00017434

Owner screenshot `2026-11-21_01-57.png`, following requested Провести, shows
check-in ОЛ-00017434 dated21.11.2026 1:56:12, basis booking ОЛ-00015930 with the
same timestamp, guest Тест конфликта2 — исходная бронь, room201, dates
10.12.2026 13:00:00 through12.12.2026 12:00:00. One Сутки period row quantity2
and total8000 RUB are visible; messages panel is empty. Native posted flag/Busy
movement and successful installation of the transaction patch are not independently
shown. Agent subsequent availabilityPOST returns200/no-store/available=false for
Dec10–12. New competitor has not been created yet.

Screenshot also displays К оплате=-7000 RUB despite total8000 RUB. No agent payment
request was made. Record this unexplained native settlement/payable discrepancy;
underlying ledger rows, preexisting payments/credits and calculation path have
not been inspected. Do not claim settlement/pricing acceptance from the visible
total alone; investigate separately after retaining the conflict fixture state.

Next: owner unposts only fresh check-in ОЛ-00017434 (not earlier ОЛ-00017433),
shows empty occupancy list. Then agent verifies release/recovery and creates one
fresh competitor on Dec10–12 before owner attempts conflict repost. No cancellation,
checkout/payment or changes to old fixtures are requested at this stage.

## Fresh check-in unposted; competing fixture recorded before dispatch

Owner screenshot `2026-11-21_02-01.png` shows zero occupancy rows and no messages
for fresh check-in ОЛ-00017434 after requested unposting. Original fresh base-key
recovery returns HTTP200 with UUID945c469c-e535-11f1-82c4-e46f13aa4449/confirmed;
Dec10–12 single-balcony availability returns HTTP200/available=true.

Fresh competitor recovery key recorded BEFORE creation: `3d112903-4a2b-4dc7-ac4d-cc074f8bee0a`.
Fixture: same Dec10–12/one guest, name Тест конфликта 2 — конкурирующая бронь,
email booking-conflict-2-other@example.invalid, phone+70000000003,
RUB2×400000=800000 kopecks. Dispatch pending. Retain key on failure and recover
before retry; never create another key automatically. The original fresh check-in
must remain unposted until the competitor is confirmed; next owner step will
attempt reposting ОЛ-00017434 and capture normal occupancy rejection/any error.
Prior fixtures and unresolved payable discrepancy remain tracked, not cleaned up.

## Fresh competitor confirmed; conflict repost pending

Fresh competitor key`3d112903-4a2b-4dc7-ac4d-cc074f8bee0a` creates reservation
UUID`945c469e-e535-11f1-82c4-e46f13aa4449` with HTTP201/no-store, exact fixture stay/price and
confirmed status. GET recovery returns200 with identical saved result;
availability Dec10–12 returns200/false. Native document number is not yet known.
Original fresh check-in ОЛ-00017434 is the target of the pending conflict repost.
Owner should Основное→Провести; expected normal occupancy refusal naming this
competitor, no Busy movement for the original check-in, no nested-transaction
error. Keep competitor active until rollback/immutable records are inspected.
Native patch effectiveness and this conflict test remain unverified until outcome.

## Fresh conflict repost correctly refused without transaction error

Owner screenshot `2026-11-21_02-03.png` after attempted Провести of fresh check-in
ОЛ-00017434 shows the normal platform refusal `Не удалось провести Заселение
ОЛ-00017434 от 21.11.2026 1:56:12`. Staff conflict detail identifies Reserved room
10.12.2026 13:00:00 through12.12.2026 11:59:59, competitor booking ОЛ-00015931 dated
21.11.2026 2:02:38. The earlier nested-transaction error is absent. This verifies
the clean refusal/message subtest after the transaction patch; it does not yet
verify absence of all failed-posting side effects or concurrent race safety.
Fixture/time links competitor native number ОЛ-00015931 to saved UUID
945c469e-e535-11f1-82c4-e46f13aa4449 (UUID not visible in screenshot).

Post-rejection agent checks: fresh base key7ac6fc48-f7de-4909-a053-c6ba1ab48525
and competitor key3d112903-4a2b-4dc7-ac4d-cc074f8bee0a each return HTTP200/no-store
with exact original confirmed result/stay/guests/price and expected UUID. Dec10–12
availability returnsHTTP200/no-store/false. No new reservation writes were sent.
Next owner evidence: dismiss refusal and inspect check-in's occupancy list for
zero rows. Settlement movement absence remains separately unverified; native
payable display -7000 RUB is still unresolved. Then cancel only competitor
ОЛ-00015931 through staff workflow, verify release and immutable recovery, and
retry original fresh check-in successfully. Do not delete key records.

## Failed fresh repost leaves zero occupancy rows

Owner screenshot `2026-11-21_02-05.png` shows fresh check-in ОЛ-00017434 with an
empty Состояние номеров list after the clean refused repost, while the staff
message still identifies Reserved competitor ОЛ-00015931. This verifies no Busy
occupancy row was left by that failed posting. Combined with prior recoveries
and availability, the serial conflicting-repost/occupancy rollback subtest passes.
Settlement-register absence, full rollback/orphan checks and concurrency remain
unverified; do not generalize this to all transactions or races.

Next staff operation in isolated base: close current check-in form, choose
Отмена бронирования in hotel workspace and select only competitor ОЛ-00015931
(guest Тест конфликта2 — конкурирующая бронь, room201, Dec10–12). Post its
cancellation document and retain screenshot/number. Do not delete the reservation
or immutable key and do not cancel base ОЛ-00015930. Then agent checks release
and unchanged recovery before asking owner to repost original ОЛ-00017434.
Cancellation has not been performed/reported yet; all fixture cleanup stays pending.

## Fresh competitor cancellation releases room and preserves recovery

Owner screenshot `2026-11-21_02-08.png` shows an unsaved Отмена бронирования
based on competitor ОЛ-00015931; first attempted posting is rejected because
Причина отмены is empty. This is required-field validation, not a booking-code
failure. Owner then supplies `2026-11-21_02-09.png`: numbered cancellation
ОЛ-00015931 dated21.11.2026 2:02:38, same booking basis, room201, reason
`Test success`, responsiblews, no unsaved marker and empty messages panel.
The cancellation and booking share displayed number/date but are different
metadata document types; do not conflate their references.

Post-cancellation agent checks: base key7ac6fc48-f7de-4909-a053-c6ba1ab48525 and
competitor key3d112903-4a2b-4dc7-ac4d-cc074f8bee0a returnHTTP200/no-store with
exact original results (UUID945c469c-e535-11f1-82c4-e46f13aa4449 and
945c469e-e535-11f1-82c4-e46f13aa4449, confirmed, original stay/guests/price).
Dec10–12 availability isHTTP200/no-store/true. Thus cancellation releases the
advisory fixture inventory and does not rewrite creation recovery. The confirmed
status is immutable historical creation status, not live cancellation status.

Next owner step: repost original fresh check-in ОЛ-00017434 on Основное→Провести;
expect success now that competitor is canceled. Inspect one Busy row, original
period/cost and no errors. Then cancellation-undo conflict and native settlement
assertions can be exercised. Restore does not imply full cleanup of old fixtures.
Serial cancellation/recovery checks do not establish cancellation-undo race safety.

## Fresh check-in restores Busy after competitor cancellation

Owner screenshot `2026-11-21_02-14.png` after requested reopen/Провести shows
fresh check-in ОЛ-00017434 with one visible occupancy row, room201, Busy,
exact start10.12.2026 13:00:00 and inclusive end12.12.2026 11:59:59, empty messages.
This confirms successful restore/post after competing booking ОЛ-00015931 was
canceled, with the correct boundaries. The serial chain (unpost → competitor
create → conflicting repost refused/no Busy → competitor cancel → original post
Busy) now has bounded native/HTTP evidence. This is not a concurrent race pass,
and settlement/payable, complete guest-count and unrelated-role tests remain open.

Next test: owner opens the CANCELLATION document Отмена бронирования ОЛ-00015931,
then Ещё→Отменить проведение while check-in ОЛ-00017434 remains posted. Undo tries
to reopen/repost the competitor's booking; it must be rejected because room201
is now Busy. Capture error/conflict details, then verify cancellation remains
posted, original Busy remains and immutable keys recover unchanged. Do not
unpost original check-in, delete keys or change dates to force cancellation undo.
This is serial cancellation-undo conflict, not CancellationUndoRace acceptance.

## Cancellation undo exposes swallowed-write transaction failure

Owner screenshot `2026-11-21_02-16.png` attempting Отменить проведение of
Отмена бронирования ОЛ-00015931 shows `В данной транзакции уже происходили
ошибки!`. Clean cancellation-undo conflict rejection is FAILED, not accepted.
Check-in ОЛ-00017434 was last observed posted with one Busy row. Post-failure
agent checks both fresh recovery keys: HTTP200/no-store with exact original
confirmed results, expected UUIDs/stay/guests/price; same-stay availability is
HTTP200/no-store/false. This does not independently prove cancellation posted
flag, native basis status or absence of settlement side effects.

Source cancellation undo sets its basis open and tries to post it, catches the
write exception by only setting Отказ, and continues while the platform
transaction has already failed. Cancellation posting also contains explicit
nested begin/commit/rollback and a swallowed-write exception. Updated only the
two handlers in ОтменаБронирования object module:

- Keep the shared ordered room/basis lock. Before undo writes anything, read the
  basis and test its interval using the existing availability helper, excluding
  that basis itself. An occupied room sets Отказ and returns before a failed write.
- Allow the platform to own cancellation posting/unposting transactions; remove
  explicit nested transaction control and swallowed posting exceptions.
- Preserve the existing privileged basis-write behavior for undo but restore the
  caller's previous privilege mode on success and failure. Unexpected write
  errors are rethrown for platform rollback, not swallowed.

Object filling and later BeforeWrite/BeforeDelete hooks are byte-identical;
BOM/CRLF preserved. Local source checks verify conflict guard precedes status
change/write and cleanup rethrows; export-layout scan passes. These are not
Designer compilation/native transaction proofs.

Patch `citypark-hotel-cancellation-undo-conflict-fix.zip` contains only
Documents/ОтменаБронирования/Ext/ObjectModule.bsl. Owner should dismiss the dialog,
apply to that DOCUMENT OBJECT module, check/F7 in development, then retry undo
of the SAME cancellation ОЛ-00015931 while check-in ОЛ-00017434 remains posted.
Expected normal occupancy refusal, original Busy intact, canceled competitor
not restored, both saved recoveries unchanged. Do not create more fixture bookings
or weaken validation. Then test undo success when room is free and restore/cleanup
through staff workflow. Native patch acceptance and concurrent undo race remain
pending. Unresolved payable balance and old fixture cleanup stay open.

## Cancellation undo now returns normal refusal

Owner screenshot `2026-11-21_02-23.png` after cancellation-object patch instructions
shows normal refusal `Не удалось сделать непроведенным Отмена бронирования
ОЛ-00015931 от 21.11.2026 2:02:38`, replacing the prior failed-transaction modal.
Busy/conflict detail is not visible in this screenshot, so record observed normal
refusal only, not independent proof of its exact branch or native status flags.
Last observed check-in ОЛ-00017434 remains the intended conflict fixture.

Agent probes after this refusal: both fresh keys returnHTTP200/no-store with exact
original saved JSON; Dec10–12 availability returnsHTTP200/no-store/false. These
checks show recovery invariance and continued blocking, but not absence of all
native basis/settlement mutations. Next: owner unposts check-in ОЛ-00017434 only,
verifies its occupancy list empty, then tests undoing SAME cancellation when the
room is free. Expected successful undo restores competitor booking Reserved
movement. This tests both allowed and denied branches; no new fixtures needed.
Old fixture cleanup, unpaid-balance discrepancy and concurrency remain open.

## Owner cannot locate check-in after restart; native base/list identity pending

Owner reports check-in ОЛ-00017434 absent from Заселение list even after search
suggestions, and that Enterprise was restarted after the last change. Restart
alone does not establish document deletion or fixture loss. Agent current HTTP
checks still find both fresh booking recovery records (HTTP200, expected UUIDs,
confirmed) and Dec10–12 availability is200/false. This confirms HTTP-side fixture
history and blocked interval persist, not direct existence/posted state of native
check-in ОЛ-00017434. Last verified native state was its Busy movement.

Requested current Enterprise Справка→О программе (file-base identity) and current
Заселение list/filter screenshot to distinguish wrong information base from list
filters or actual lifecycle changes. Pause dependent native undo operations until
target is identified. No extra fixture creations, deletions, or posting changes
performed by agent. The allowed cancellation-undo test remains pending.

## Fresh check-in located and unposted for allowed cancellation undo

Owner screenshot `2026-11-21_02-35.png` identifies check-in ОЛ-00017434 by title
and shows empty occupancy rows/messages. This resolves locating the target
check-in; earlier wrong-base/list-filter cause was not determined. Agent subsequent
GETs for both fresh keys return200/no-store, expected UUID/confirmed status;
Dec10–12 availability returns200/no-store/true. The room is now free for the
allowed cancellation-undo test. These GETs verify identity/status; earlier full
payload equality checks remain the detailed recovery evidence.

Next owner operation: open CANCELLATION Отмена бронирования ОЛ-00015931 via
Все функции→Документы→Отмена бронирования, Ещё→Отменить проведение. Expected
success now that the conflicting Busy movement is gone, restoring competitor
booking ОЛ-00015931 Reserved state for Dec10–12. Keep original fresh check-in
unposted during undo/inspection. Capture result; no new fixture creation needed.
Successful undo and resulting native register state have not yet been reported.

## Allowed cancellation undo succeeds (owner report + HTTP checks)

Owner reports successful unposting/undo of cancellation ОЛ-00015931 after fresh
check-in ОЛ-00017434 was unposted and room-free availability verified. No new
screenshot of restored competitor register row or cancellation flag is supplied.
Agent follow-up: Dec10–12 availability changes fromtrue tofalse (HTTP200/no-store),
consistent with the competitor booking being restored; both fresh keys return
HTTP200/no-store with EXACT original saved results/UUIDs/stay/guests/price.
This supports successful allowed-undo behavior and unchanged recovery history.
Denied-undo and allowed-undo branches now have bounded serial evidence; the
concurrent CancellationUndoRace, exact native restored row and settlement effects
are still unverified.

Next cleanup: repost the SAME cancellation Отмена бронирования ОЛ-00015931,
leaving fresh check-in ОЛ-00017434 unposted, then agent verifies room release and
recovery again. Do not delete recovery records. Earlier Dec1–3 competitor
ОЛ-00015929 remains a separate cleanup item; original unposted check-in
ОЛ-00017433 and base booking ОЛ-00015928 are retained historical test documents.
Current Dec10–12 competitor is active until the cancellation is reposted.

## Fresh cancellation repost cleanup verified; duplicate-key race prepared

Owner reports reposting cancellation ОЛ-00015931. Agent Dec10–12 availability
isHTTP200/no-store/true; both fresh recoveries remain EXACT original saved JSON,
HTTP200/no-store. Thus this fixture's advisory occupancy has been released while
history remains. Fresh check-in ОЛ-00017434 remains unposted per owner workflow;
no test documents/key rows were deleted. Earlier Dec1–3 competitor ОЛ-00015929
still needs separate native cleanup; payable/settlement issues remain open.

Next authorized isolated test: two client-concurrent identical creation requests
on one key. Preflight Dec15–17/single-balcony/one guest isHTTP200/available=true.
Recovery key recorded BEFORE dispatch: `83de36b9-507a-46d2-86c9-a269fec9af78`.
Synthetic name Тест одновременного повтора, booking-duplicate-race@example.invalid,
phone+70000000004, RUB2×400000=800000 kopecks. Each request will use this SAME
payload/key; expect responses201/200 and the same UUID. A client-side barrier
will synchronize dispatch; actual server scheduling/interleaving is not
instrumented. Native one-document/counterparty/key count and cleanup will still
need inspection. No dispatch result is claimed in this preparation entry.

## Duplicate-key HTTP concurrency assertions pass

Client-barrier dispatch sent two identical POST reservations requests for retained
key`83de36b9-507a-46d2-86c9-a269fec9af78`, Dec15–17 fixture, in two Python worker threads.
Observed HTTP statuses[201, 200], both no-store, and
exactly identical confirmed result JSON/UUID`bacdd225-e53b-11f1-82c4-e46f13aa4449` with expected
stay/guest count/price. Durations[0.498, 0.555] seconds.
GET recovery returns200 with exactly that saved result; availability is200/false.
One winning successful creation response201 and one replay200 are verified.
No new keys/retries were dispatched on errors.

This passes client-concurrent duplicate-key HTTP assertions for the current file
base/publication. Actual server overlap is not instrumented; native one-document,
one-counterparty and one-key counts/rollback side effects remain to inspect.
One expected reservation remains for inspection/normal staff cancellation. Native
number unknown; guest Тест одновременного повтора, room201, Dec15–17. Do not
mark full DuplicateKeyRace or all concurrency scenarios complete from HTTP alone.

## Owner duplicate-key counts and planned different-key room race

Owner reports one booking and one counterparty for guest Тест одновременного
повтора, following duplicate-key201/200 test. This is owner-reported native count
evidence; no list screenshot or dedicated filtered key-count was supplied.
Recovery equality and same UUID were verified over HTTP. That fixture's booking
and key are retained pending normal staff cancellation; no cleanup claimed.

Next independent isolated test: client-concurrent creation using DIFFERENT keys
for the same one-room website type single-balcony, Dec20–22/one guest. Preflight
HTTP200/available=true. Keys recorded BEFORE dispatch:
A `2a8273d3-93a5-4d38-904e-7cbc39cf507c`, synthetic Тест гонки номера A,
booking-room-race-a@example.invalid, +70000000005;
B `18506a95-adec-48e5-9481-77f5c94fc2f8`, synthetic Тест гонки номера B,
booking-room-race-b@example.invalid, +70000000006.
Each priceRUB2×400000=800000 kopecks. Expected statuses201/409 ROOM_UNAVAILABLE,
one confirmed result/winner recovery200 and losing key recovery404, then blocked
availability. Retain BOTH keys on transport errors; do not generate replacements.
Client barrier synchronizes dispatch but server interleaving is not instrumented.
Native document/counterparty/key counts and cleanup still require inspection;
this is HTTP/HTTP contention, not concurrent website/staff acceptance.

## Different-key one-room HTTP race assertions pass

Two client-barrier-synchronized POSTs for Dec20–22/single-balcony/one guest,
different retained keys, return statuses[409, 201] with
no-store; durations[0.566, 0.387]seconds. Winner guest
Тест гонки номера B, key`18506a95-adec-48e5-9481-77f5c94fc2f8`, UUID`bacdd227-e53b-11f1-82c4-e46f13aa4449`,
confirmed exact stay/price. Loser guestТест гонки номера A,
key`2a8273d3-93a5-4d38-904e-7cbc39cf507c`, HTTP409 ROOM_UNAVAILABLE with expected Russian envelope.
Winner recovery200 exactly equals saved result; loser recovery404 has expected
envelope. Subsequent advisory availability200/false. Thus HTTP assertions for
one-room different-key client contention pass. No retries/new keys were sent.

Owner native assertions still required: exactly one booking and counterparty for
winner guest and zero for loser; one successful-key record and no losing-key row;
correct room/state/period/price movements. No native counts supplied yet. Client
simultaneous dispatch is not proof of specific server interleaving or staff race
safety. One expected race-winner booking is retained for inspection and normal
staff cancellation; native number unknown. Dec15–17 duplicate test and earlier
Dec1–3 fixture cleanup also pending. Full task acceptance remains incomplete.

## Owner confirms room-race counts; 28 invalid-input checks pass

Owner reports the different-key room test passed after being asked to verify
winner Тест гонки номера B has one booking/one counterparty and loser A has zero
of each. Record as owner confirmation in context, not direct agent database count.
Losing-key404 and winning saved result equality were verified over HTTP. Native
key-row count, movements, all teardown and server interleaving remain limited.

Agent ran 28 invalid-input HTTP checks, all HTTP400 INVALID_REQUEST with exact
Russian envelope and Cache-Control no-store:

- Availability: guests0, over-capacity2 for single-balcony, fractional1.5, string1,
  booleantrue; equal/reversed dates; invalid calendar date; past arrival;
  noncanonical date; unknown room type; unknown stay field (12 cases).
- MalformedJSON, array root, null root, empty body, over16384-byte body (5).
- Malformed recovery key (1).
- Creation price: USD, zero nightly price, fractional kopecks, representational
  money overflow, calendar-night mismatch, total mismatch (6).
- Creation contacts: blank name, invalid email, short phone, newline in name (4).

Creation-invalid cases reused the EXISTING winning key18506a95-adec-48e5-9481-
77f5c94fc2f8 and its exact saved fixture apart from one invalid field; no fresh
keys or additional reservations were created. Post-test recovery200 equals the
exact original saved JSON. This establishes tested negative validation only;
successful boundary values, Moscow midnight, valid leap dates, other capacities,
large accepted stays/horizons and other required scenarios remain unverified.

Next native check: open unposted check-in ОЛ-00017434 and inspect its
Взаиморасчеты за проживание navigation list; expected no recorder movements after
unposting. Earlier К оплате=-7000 RUB requires separate diagnosis; no payment
requests were made. Source shows debt lookup queries settlement by document
reference; an unsaved empty reference is one hypothesis for a stale displayed
balance, not an observed cause. Inspect ledger/refresh before changing code.

## Unposted fresh check-in settlement list is empty

Owner screenshot `2026-11-21_02-54.png` shows the Взаиморасчеты за проживание
navigation list within Заселение ОЛ-00017434 empty. This is native evidence that
this displayed document-related settlement view contains no rows after unposting.
Exact navigation filter is not shown, so do not extrapolate to all ledger data.
The empty occupancy view was separately confirmed. TESA messages are ignored
per explicit owner instruction; no equipment work is included.

Source review confirms ПриОткрытии refreshes payable through
ОбновитьСтатусЗадолженности, while ПослеЗаписи only updates availability of form
controls. The debt helper queries settlement turnovers by document reference.
A new unsaved form has an empty reference, so querying empty-reference historical
rows could explain the earlier -7000 display persisting after first save; this
remains a hypothesis until ledger/empty-reference values are inspected.
Next ask owner for current Сумма and К оплате on Основное of the now-reopened,
unposted but saved ОЛ-00017434. Keep it unposted and do not pay; no source changes
are justified solely by the current empty-list screenshot. Native numeric guest
count, payable diagnosis, remaining boundaries/roles/races and cleanup still open.

## Reopened payable correct; unsaved-reference display guard added

Owner screenshot `2026-11-21_02-55.png` of saved, unposted check-in ОЛ-00017434
shows total8000 RUB and payable8000 RUB, original Dec10–12/13:00→12:00 period.
The settlement navigation list was previously empty. This resolves the incorrect
negative value for this reopened form, not proof that the initial unsaved path
was correct. Earlier -7000 did not persist on reopening. No payment was made
by agent or requested in the test workflow.

Source showed ОбновитьСтатусЗадолженности querying settlement even for unsaved
Объект.Ссылка, while ПриОткрытии calls it and ПослеЗаписи did not refresh it.
Thus empty-reference settlement rows could affect a new draft and leave a stale
display after first save; actual historical empty-reference ledger values were
not inspected, so this cause remains a source-backed explanation rather than
verified ledger arithmetic.

Updated only Заселение FORM module: when reference is empty, display Объект.Сумма
and return before querying settlement. Existing referenced documents retain the
normal debt calculation. ПослеЗаписи refreshes the payable value after updating
control availability. No shared ledger helper, register, price or payment writes
were modified. BOM/CRLF preserved; structural/source and export-layout checks
are not native BSL compilation or payment regression proof.

Patch citypark-hotel-checkin-payable-display-fix.zip contains one FORM module,
Documents/Заселение/Forms/ФормаДокумента/Ext/Form/Module.bsl. Owner must update
Документы→Заселение→Формы→ФормаДокумента→Модуль (not object module), check/F7,
then open a NEW unsaved check-in from existing active booking guest
Тест одновременного повтора (Dec15–17). Expected total/payable8000 before save;
retain draft unsaved for first screenshot. Current old check-in can remain
unposted. No new reservation fixtures needed. Native guard/after-save acceptance
and referenced-document payment behavior remain pending.

## New unsaved check-in payable guard verified by owner

Owner screenshot `2026-11-21_03-04.png` after form-patch instructions shows
Заселение (создание), blank number, synthetic guest Тест одновременного повтора,
basis Бронирование ОЛ-00015932 dated21.11.2026 2:40:13, room201, Dec15 13:00
throughDec17 12:00, one Сутки period quantity2, total8000 RUB and payable8000 RUB.
This supplies the previously unknown native number for duplicate-key fixture
(UUIDbacdd225-e53b-11f1-82c4-e46f13aa4449/key83de36b9-507a-46d2-86c9-a269fec9af78),
linked by fixture identity; UUID not visible in screenshot.

The previously negative payable display is absent in this new unsaved form;
empty-reference guard's intended UI result is owner-verified. After-save refresh
still needs testing. Next owner step: click Записать only (not Провести/Оплатить),
keep the same form open and capture assigned number/total/payable. Expected8000
for both, preserving dates/price without closing/reopening. This creates one
unposted check-in draft for the existing booking, not another reservation.
TESA remains explicitly outside task scope. Complete payment/settlement behavior,
referenced payments/credits, guest-count and other remaining acceptance are not
established by this screenshot.

## Payable remains correct immediately after first save

Owner screenshot `2026-11-21_03-06.png`, following requested Записать-only step
without closing/reopening, shows numbered check-in ОЛ-00017435 dated21.11.2026
2:40:13, basis ОЛ-00015932, guest Тест одновременного повтора, room201, Dec15
13:00→Dec17 12:00, Сутки quantity2. Total/payable both8000 RUB, no unsaved marker,
messages panel empty. Together with blank-number preceding screenshot this
verifies intended payable UI values before and immediately after first save.
No payment was requested; posted status is not established by the numbered form.

Next owner step: click Провести on this same saved check-in and inspect
Взаиморасчеты за проживание. Existing daily posting logic should produce two
charge movements4000 RUB each (8000 total), linked to ОЛ-00017435. Inspect row
amounts, recorder/calculation-document references and absence of unexpected
payment/extra rows; after-save UI should still show8000/payable8000 in the
no-payment fixture. Native ledger amounts/periods and payment behavior remain
unverified until screenshot. No new reservation creation is required. Existing
fixture teardown and other runtime/role/concurrency requirements remain pending.

## Posted check-in settlement charges match backend snapshot

Owner screenshot `2026-11-21_03-07.png` after requested Провести of check-in
ОЛ-00017435 shows exactly two visible positive settlement rows. Both registrar
and Документ расчетов are Заселение ОЛ-00017435 dated21.11.2026 2:40:13:
row1 at15.12.2026 13:00:00, amount4000 RUB; row2 at16.12.2026 12:00:00,
amount4000 RUB. Messages panel is empty. Observed charges total8000 RUB and match
the accepted backend price snapshot. The noon boundary for the second daily
charge matches the existing daily-tariff helper; do not misinterpret it as an
extra/missing night. No payment row is visible in this document navigation list.

This verifies actual native charge amounts and calculation-document association
for the fixture, beyond earlier form display. Actual catalog tariff difference,
referenced-document payment/credit regression and all other scenario assertions
remain unverified. Next cleanup/assertion: owner unposts only ОЛ-00017435,
checks both settlement and occupancy navigation lists empty, then agent checks
Dec15–17 availability and immutable duplicate-key recovery. No new fixtures or
payments are needed. Other outstanding native and concurrency tests remain open.

## Duplicate fixture unposted: both native lists empty, HTTP release verified

Owner reports both settlement and occupancy lists for ОЛ-00017435 are empty
following requested unposting. This is owner-reported native evidence (no new
screenshots supplied). Agent verifies duplicate-fixture key83de36b9-507a-46d2-
86c9-a269fec9af78 GET returns200/no-store with EXACT original confirmed JSON/UUID
bacdd225-e53b-11f1-82c4-e46f13aa4449, Dec15–17/one guest/price; availability for
that stay returns200/no-store/true. Combined with earlier posted two4000 charge
rows, this verifies the fixture's post/unpost financial/occupancy subchecks to
that extent. Broader payment/credit, tariff-difference and race assertions remain
unverified. No payment or new reservation was created during this check.

Remaining active fixture cleanup requested next, using staff cancellation only:
old competing booking ОЛ-00015929 (guest Тест конфликта онлайн-бронирования,
Dec1–3) and room-race winner guest Тест гонки номера B (Dec20–22, native number
not yet captured). Keep unposted check-ins and immutable recovery records intact.
Do not cancel unrelated base/production bookings or infer the winner's number.
Fresh Dec10–12 competitor cancellation remains posted per prior owner report;
Dec10–12 release was verified. After remaining cancellations, recheck all four
fixture intervals and recovery records. Full task/runtime/role/concurrency
acceptance still incomplete; no completion/commit implied by cleanup.

## Active fixture intervals released; recovery history preserved (2026-11-21)

Owner reports completing remaining staff cancellations (old competitor
ОЛ-00015929/Dec1–3 and room-race winner Тест гонки номера B/Dec20–22).
Agent verifies all four fixture stays Dec1–3, Dec10–12, Dec15–17 and Dec20–22
returnHTTP200/no-store/available=true. Six saved creation recovery records
(original, old competitor, fresh base, fresh competitor, duplicate race and
room-race winner) each returnHTTP200/no-store with EXACT original saved JSON,
including dates/guest count/price/UUID. Losing room-race key
2a8273d3-93a5-4d38-904e-7cbc39cf507c remains404 RESERVATION_NOT_FOUND/no-store.
No new bookings or key edits/deletions were sent in this verification.

This verifies active fixture interval release and preservation of recovery
history, not complete native orphan/rollback cleanup or deletion of historical
documents. Unposted check-ins/canceled bookings/history intentionally remain;
failed initial attempts still need native no-orphan confirmation if required.
Cancellation native numbers for final winner cleanup were not supplied.

Next mandatory access acceptance: inspect currently assigned development roles
for HTTP userws in Designer→Администрирование→Пользователи→ws. Earlier this
account could invoke Reports, so dedicated-role-only execution and denial of
unrelated service access have not been demonstrated. Request checked-role list
before narrowing permissions, preserving a separate administrator login. Then
exercise booking availability/lookup and reports denial as appropriate. Current
creation history can support read-only recovery checks; no new fixtures needed
for this stage. Exact necessary session/BSP rights must be recorded rather than
using full rights to mask errors. Remaining native numeric guest count, actual
price-vs-tariff proof, successful extended boundary cases, staff/web race and
fragmented inventory/rollback injection also remain open. Temporary owner HTTP
DEV-DIAG-2 response must be restored to generic errors before any final acceptance.
Task02 remains unchecked/uncommitted; no production actions or task03 work.

## Dedicated service-role-only read checks; creation fixture prepared

Owner screenshot `2026-11-21_03-15.png` showed ws assigned Администратор системы,
Полные права and Сервис онлайн-бронирования. Earlier successful create tests
therefore do NOT establish restricted-role execution. Owner reports unchecking
the first two, retaining only dedicated service role. Agent fresh authenticated
requests: Booking availability Dec10–12 returns200/no-store/true, saved recovery
returns200/no-store/expected original UUID, Reports booking-references returns
403 plain text (body omitted). Thus required read paths work and unrelated Reports
access is denied with owner-reported restricted assignment. No extra session/BSP
rights were added for these checks.

Required creation under restricted rights remains to test. Preflight Dec25–27,
single-balcony/one guest returnsHTTP200/available=true. Recovery key recorded
BEFORE dispatch: `5824ae7e-dfb9-4c79-8ea8-1f34bb740e6c`. Synthetic guest Тест ограниченных прав бронирования,
booking-restricted@example.invalid, +70000000007, RUB2×400000=800000 kopecks.
Dispatch pending. Retain same key on error/recovery; never regrant full rights to
mask failures. Any missing object operation must be identified and granted only
if required by the booking workflow. Native count/movements and cleanup remain
necessary after a successful restricted creation. Separate admin login retained.

## Restricted creation fails at one missing session read; scoped role fix

First creation under owner-reported dedicated-role-only ws returns503/no-store.
Development diagnostic identifies ОнлайнБронирование line285 `Документ.Заполнить`,
its ОбработкаЗаполнения, then ЗаполнениеДокументовВызовСервера line80 reading
ПараметрыСеанса.АвторизованныйПользователь: Нарушение прав доступа. This exact
operation is missing; do not restore administrator/full-rights roles.

After failure, retained restricted key5824ae7e-dfb9-4c79-8ea8-1f34bb740e6c recovers
404 RESERVATION_NOT_FOUND; Dec25–27 availability200/true. No confirmed booking
exists for this attempt; no-orphan native proof remains pending.

Using role-edit, added ONLY SessionParameter.АвторизованныйПользователь:Get=true
to ОнлайнБронированиеСервис. The unchanged base БазовыеПрава role supplies this
same Get right, confirming the platform XML spelling. Existing dedicated rights
and role UUID/wrapper remain untouched; semantic XML comparison after removing
the single added object equals previous rights tree; BOM/CRLF preserved.
Role validator passes; native compilation and restricted creation retry pending.
Patch citypark-hotel-booking-session-right-fix.zip contains one Rights.xml.

Owner can apply in development Designer Общие→Роли→ОнлайнБронированиеСервис,
rights tree Параметры сеанса→АвторизованныйПользователь, enable only Получение
(Get); no Set right. Check/F7, keep ws only in dedicated service role, then
recover/retry EXACT retained fixture key/payload. Any later permission failure
must be identified individually. Read/recovery and Reports403 already work;
full RestrictedRole scenario still incomplete. Temporary diagnostics remain
pending removal before final acceptance; no production changes.

## Dedicated-role creation/replay/recovery works with single session Get

Owner reports applying the scoped Get permission. Agent first recovers retained
restricted-test key5824ae7e-dfb9-4c79-8ea8-1f34bb740e6c:404/no-store. Retrying its
exact Dec25–27/single-balcony/one-guest fixture now returnsHTTP201/no-store,
confirmed UUIDc809ade1-e541-11f1-82c4-e46f13aa4449, exact accepted stay and
RUB2×400000=800000 kopeck price. Exact-payload replay returnsHTTP200 identical
saved result; GET recovery200 identical; changed guest name same key409
IDEMPOTENCY_CONFLICT; Reports booking-references403 plain text. The only added
session/BSP right was SessionParameter.АвторизованныйПользователь:Get, bundled
inside the dedicated role. No full-rights/admin roles were regranted by agent.

Owner-reported service-only assignment plus runtime creation/read/replay/denial
checks support the required restricted HTTP workflow. Full RestrictedRole still
needs native one-document/counterparty/key count and prohibitions on direct
settings/room-price edits/cancellation/checkout or equivalent runtime evidence;
unrelated authenticated-role access to every method remains separately incomplete.
Do not overstate source permission restrictions as native tests.

One new synthetic reservation remains: guest Тест ограниченных прав бронирования,
room201, Dec25–27, 8000 RUB; native number unknown. Owner should inspect one
booking/one counterparty, then under separate development administrator cancel
only that booking via Отмена бронирования with test-completion reason. Keep
history and prior unposted check-ins intact. Agent will verify release and saved
recovery afterwards. No production actions or payment requests.

## Owner restricted-fixture native counts confirmed

Owner explicitly confirms Dec25–27 has one booking and one counterparty after
being asked to inspect guest Тест ограниченных прав бронирования. Record as
owner-reported native counts for successful restricted creation/replay fixture,
not direct agent database inspection. Native number/key-row count or movement
screenshot not supplied. Cancellation has not yet been reported. Request staff
Отмена бронирования of only this restricted-test booking under separate dev
administrator, reason Завершение тестирования, post it and retain immutable
key/history. After report, agent will verify release and exact recovery.
No new writes or HTTP probes performed in this acknowledgement.

## Restricted fixture cleanup and valid date boundaries verified

Owner reports staff cancellation of restricted-test booking. Agent Dec25–27
availability returns200/no-store/true; key5824ae7e-dfb9-4c79-8ea8-1f34bb740e6c
recovery returns200/no-store and EXACT original confirmed result (UUID
c809ade1-e541-11f1-82c4-e46f13aa4449, stay, guests1, RUB2×400000=800000 kopecks).
Historical recovery is intentionally retained; native cancellation number not
supplied, no-orphan history proof still separate.

Four positive advisory validation checks under restricted account return200,
exact stay parameters, boolean available=true and no-store:
1) Dec30–31 2026, one night; 2) Feb28–Mar1 2028, valid leap-day/two nights;
3) Jan1–Apr12 2027,101 nights; 4) Nov21–23 2027, arrival over six months ahead.
No reservation creation sent. These verify accepted advisory date input and no
artificial 90-night/six-month cap, not native long-stay price/posting correctness.

Next negative authorization test: owner temporarily changes only development
userws assignment from Сервис онлайн-бронирования to Базовые права, retaining a
separate administrator login. Then agent tests all three Booking HTTP methods
with valid Basic credentials: availability, saved-key recovery and a SAME
previously saved payload/key POST (so an unexpected permission failure cannot
create a new reservation). Expect authenticated403 for each; distinguish native
plain-text denial from application FORBIDDEN envelope. Restore ws dedicated
service role afterwards before further service tests. No full-rights role is
requested, and no unrelated data edits are authorized by this test instruction.
Negative-role test not yet run. Direct settings/price/cancel prohibitions and
remaining native/staff races still incomplete; temporary DEV-DIAG-2 response
must be removed before acceptance.

## Authenticated unrelated-role denial: all Booking methods403

Owner reports readiness after temporary ws role-change instruction. Using the
same valid Basic credentials, all three Booking method probes returnHTTP403:
POST availability, GET saved-key recovery, POST reservations with exact existing
restricted-test saved payload/key5824ae7e-dfb9-4c79-8ea8-1f34bb740e6c. Thus no new
key/reservation was used for the negative test. Responses are platform-generated
text/plain;charsetUTF-8, not application JSON; Cache-Control is absent. Native
body omitted. Do not claim handler FORBIDDEN envelope/no-store assertions here.

This supports denial of all methods under owner-reported unrelated-only
Базовые права assignment. Native assignment not rescreenshoted. Earlier anonymous
401 and dedicated-role200/201 cases complement this boundary check. Missing/wrong
credentials and every direct-object restriction are not all tested by it.

Next owner step: restore ws ONLY Сервис онлайн-бронирования (remove temporary
Базовые права), save. Dedicated role already contains the single required session
Get right; keep admin/full-rights roles unchecked. Then agent verifies restored
availability/recovery/Reports denial. Negative-role stage has no new fixture to
clean up. Temporary HTTP DEV-DIAG-2 still requires removal before final acceptance.

## Dedicated role restored; temporary HTTP diagnostics cleanup prepared

Owner reports restoring ws to only Сервис онлайн-бронирования. Agent verifies
Dec25–27 availability200/no-store/true, saved restricted fixture recovery200/no-store
EXACT original JSON, and Reports booking-references403. This confirms service
access and unrelated Reports denial after the negative-role test. No new bookings
or other data writes were sent.

Owner development handler still requires restoring the temporary DEV-DIAG-2
message branch. Repository HTTPServices/Booking/Ext/Module.bsl already has generic
unexpected503 message Бронирование временно недоступно and no diagnostic marker
or ОписаниеОшибки call. Packaged that unchanged generic module in
citypark-hotel-booking-http-cleanup.zip, one file, CRC/byte verified. This archive
changes no shared/business/object/role modules and does not undo their fixes.

Owner must replace HTTP-сервисы→Booking→Модуль from this archive, check/F7 in
isolated base. Next controlled availability503 test can temporarily disable
single-balcony setup under development administrator, verify generic envelope/no
internal details/no-store, then reenable and verify200; not yet run. Do not
inject posting failures or use production for this cleanup. Task02 still requires
remaining documented native/race/boundary/permissions checks before completion.

## Clean HTTP handler owner-applied; read-only protocol regression passes

Owner reports applying generic HTTP handler. Reran booking-http-check.py WITHOUT
creation against restricted ws/Dec25–27 fixture. Anonymous denial, availability,
unknown-key404 and unknown-field400 (exact error envelopes/private headers) pass.
No reservation creation requested. Normal checks do not independently prove the
unexpected503 branch no longer contains owner diagnostics.

Next controlled test under separate development admin: temporarily disable ONLY
single-balcony settings Включено; keep references/times/role intact. Then agent
POST availability should return exact503 BOOKING_UNAVAILABLE with generic
Бронирование временно недоступно/no-store and no DEV-DIAG/module/line text.
Reenable the SAME record immediately after observation and verify200. Current
record remains enabled until owner reports disabling; no server/data mutation
was performed by agent. This is isolated development only, no fixture creation.

## Generic503 response verified; reenable setup pending

After owner requests checking the disabled-setup response, authenticated
single-balcony Dec25–27 availability returnsHTTP503, application/json;charsetUTF-8,
Cache-Control:no-store and exact body:
`{"statusCode":503,"code":"BOOKING_UNAVAILABLE","message":"Бронирование временно недоступно"}`.
No DEV-DIAG/module/source-line/exception details appear. Controlled503 privacy
regression passes and confirms generic branch is active in owner development
handler. No posting failure was injected and no reservation was created.
Next required step: owner rechecks Включено on SAME single-balcony settings record
and saves (no F7), then agent verifies restored200. Do not leave fixture setup
disabled inadvertently. Remaining documented acceptance gates are unchanged.

## Setup reenabled after generic503 test; prepare double-twin coverage

Owner reports rechecking Включено on single-balcony. Agent availability Dec25–27
returnsHTTP200/no-store/exact stay/available=true; retained restricted recovery
returns200/no-store/EXACT original saved JSON. Controlled generic503 test setup
is restored; no new reservation created.

Next coverage gap: only single-balcony one-room/capacity1 path has native runtime
fixture evidence. Double-twin needs independent verified settings row to test
capacity2, four-room allocation and whole-stay/fragmented inventory. Frozen
contract maps rooms203/206/302/303 to UUID suffixes6/9/b/c and shared 1C type
fb170ab1… (Двухместный (обычный), same as201). Owner development administrator
must create/edit ONLY double-twin settings, retaining single-balcony unchanged:
existing verified booking organization, responsiblews, matching room type,
НомераJSON exact four-UUID array from BOOKING_CONTRACT.md, enabled after checking
references. This is data setup, noF7. Configuration row not yet confirmed or
probed; no twin reservations created yet. Service role/account remains restricted.
Full task acceptance still includes other mapped types/capacities, native guest
count, staff-vs-web race, failure injection and remaining gates.

## Double-twin setup probe: responsible-user mismatch (2026-11-21)

After owner reports setup ready, advisory Jan10–12 2027 double-twin/guests2
returns HTTP403 with exact FORBIDDEN envelope and no-store. Same-date
single-balcony/guests1 returns200/available=true; double-twin/guests3 returns400
with exact INVALID_REQUEST envelope/no-store. Capacity rejection passes, but
valid twin availability is blocked. Source ПолучитьНастройки checks current
application user against the row Ответственный and raises FORBIDDEN on mismatch;
owner should select the same ws reference as the working single-balcony row.
No reservation POST or new fixture was created. Multi-room acceptance pending.

## Double-twin retry returns generic503 (2026-11-21)

Owner requests retry after responsible-user correction. Jan10–12 2027 twin/2
now returns503 BOOKING_UNAVAILABLE with exact generic envelope/no-store, rather
than403. This does not establish which setup check failed; disabled/incomplete
settings or room JSON/type validation can cause this response. Same-date
single-balcony/1 remains200/available=true; twin/3 remains400 INVALID_REQUEST.
No reservation created. Ask owner for the complete saved double-twin settings
record to inspect enabled flag, references, four UUIDs and actual room types.

## Double-twin JSON correction: advisory checks pass (2026-11-21)

Owner reports updating НомераJSON and requests retry. Double-twin Jan10–12 2027,
guests2 returns HTTP200, exact echoed stay and available=true. Guests3 returns
HTTP400 with exact INVALID_REQUEST envelope. Both responses have no-store.
This verifies the configured twin advisory path and capacity rejection after
the owner's JSON correction; it does not prove native guest count, multi-room
allocation or whole-stay fragmentation. No reservation POST or fixture created.
The prior503 root cause was not instrumented; do not attribute it to room types.

## Two-guest twin creation fixture planned (2026-11-21)

Owner authorizes continuing task02 testing. Retained key 94128d74-eb0a-48a7-9f1f-4e576a2e4655
for double-twin Jan10–12 2027, guests2, synthetic Тест двух гостей — twin 1,
booking-twin-1@example.invalid / +70000000008, RUB2×400000=800000 kopecks.
Exact request is retained locally in /tmp/citypark-twin-capacity-fixture.json
without credentials. Dispatch only after availability200/true; recover this same
key before retry after any uncertain result. One fixture only at this stage;
owner native room, guest-count, pricing and Reserved-row inspection pending.
No successful creation claimed by this planning entry.

## Two-guest twin creation HTTP checks pass (2026-11-21)

Retained key94128d74-eb0a-48a7-9f1f-4e576a2e4655 initially404; advisorytrue;
POST returns201 confirmed UUID84a851e2-e580-11f1-82c4-e46f13aa4449, Jan10–12 2027,
double-twin/guests2, exact RUB price2×400000=800000 kopecks. GET and exact-payload
POST replay return200 with identical saved result; all no-store and JSON.
Synthetic counterparty name Тест двух гостей — twin 1. One active twin fixture
remains for owner native document number/assigned room/КоличествоГостей=2,
configured times, cost/Reserved movement and count inspection. No native
allocation or guest-count acceptance inferred from HTTP response. Keep fixture
for subsequent four-room allocation test, then release via staff cancellation.

## Native twin booking form evidence (2026-11-21)

Owner screenshots show booking ОЛ-00015935 dated21.11.2026 10:52:37,
counterparty Тест двух гостей — twin 1, room203, responsiblews, blank comment.
Arrival10.01.2027 13:00, departure12.01.2027 12:00; Заселение row Сутки2;
Стоимость row2×4000=8000 RUB with matching period. This establishes native
assigned room and price/time display for retained key94128d74-eb0a-48a7-9f1f-4e576a2e4655.
The displayed Кол-во2 is tariff/night quantity, not proof of КоличествоГостей.
Reserved movements, native guest count and one-counterparty/document counts
remain unverified. Fixture stays active for subsequent allocation checks.

## Native twin Reserved movement and allocation fixture plan (2026-11-21)

Owner screenshot confirms ОЛ-00015935 room203 Reserved,10.01.2027 13:00:00
through12.01.2027 11:59:59, one visible row. Native guest-count remains pending.
Next retain twin2/3/4 creation keys and twin5 expected-refusal key before dispatch.
All same Jan10–12 2027/two guests/RUB2×400000=800000, synthetic contacts only.
One-room requests, not a multi-room cart. No further successful writes claimed yet.

- Тест двух гостей — twin 2: key 8312500d-3b61-4604-9792-85df98a1a5b3, booking-twin-2@example.invalid, +70000000009.
- Тест двух гостей — twin 3: key a0231206-f91c-4d34-b1f7-4a830fd4e9b7, booking-twin-3@example.invalid, +70000000010.
- Тест двух гостей — twin 4: key 4e2758f2-b2a6-43e5-9129-44165e8fa11e, booking-twin-4@example.invalid, +70000000011.
- Тест двух гостей — twin 5: key c20ceab2-a04c-42b3-b35e-2a4411b5408c, booking-twin-5@example.invalid, +70000000012.

Exact payloads retained in /tmp/citypark-twin-allocation-fixtures.json without credentials.
Native assigned rooms/counts and staff cleanup required after HTTP checks.

## Four-room allocation HTTP result (2026-11-21)

After native room203 Reserved evidence for twin1, twin2/3/4 each return201 with
same requested stay/guests2/8000 RUB, recovery200 exact. Between allocations,
availability remains true. After fourth booking, availability200/false;
twin5 POST409 ROOM_UNAVAILABLE exact envelope and subsequent GET404. All
responses JSON/no-store. Client requests sequential; no concurrency claim.

- Тест двух гостей — twin 2: key8312500d-3b61-4604-9792-85df98a1a5b3, reservationUUID84a851e4-e580-11f1-82c4-e46f13aa4449.
- Тест двух гостей — twin 3: keya0231206-f91c-4d34-b1f7-4a830fd4e9b7, reservationUUID84a851e6-e580-11f1-82c4-e46f13aa4449.
- Тест двух гостей — twin 4: key4e2758f2-b2a6-43e5-9129-44165e8fa11e, reservationUUID84a851e8-e580-11f1-82c4-e46f13aa4449.

Four active bookings remain Jan10–12 2027. Owner must inspect distinct native
rooms203/206/302/303, matching Reserved rows, one document/counterparty each
for twin1–4 and zero new document/counterparty for refused twin5. Native guest
count, full-stay fragmentation and fixture cleanup remain pending.

## Native four-room allocation movements confirmed (2026-11-21)

Owner screenshot shows four Reserved rows, each Jan10 2027 13:00:00 through
Jan12 2027 11:59:59: ОЛ-00015935→203, ОЛ-00015936→206,
ОЛ-00015937→302, ОЛ-00015938→303. Distinct eligible rooms and occupancy
boundaries are verified for the four HTTP-success fixtures. Screenshot does not
establish counterparty uniqueness or absence of orphan documents/counterparties
from refused twin5; owner count confirmation pending. Native guest-count and
fragmented whole-stay coverage pending. All four bookings remain active.

## Owner confirms allocation counts; fragmented-stay setup next (2026-11-21)

Owner confirms twin1–4 each have one booking and one counterparty; refused twin5
has zero bookings/counterparties. This supplements four distinct Reserved rows.
Next use development administrator to shorten existing four fixtures into
alternating one-night blocks:15935/203 and15936/206 Jan10 13→Jan11 12;
15937/302 and15938/303 Jan11 13→Jan12 12. Update Проживание and explicit staff
Стоимость rows to matching one-night periods, quantity1×4000=4000, then repost.
Online automatic pricing preserves prior snapshot, so cost changes must be
explicit staff edits. Saved key/result records remain immutable historical
creation snapshots; never edit them. Setup not yet confirmed. Expected advisory
Jan10–11=true, Jan11–12=true, Jan10–12=false, then full-stay creation409 with no
new records. No fragmentation claim until native periods/HTTP outcomes checked.

## Correction: booking cost table is read-only; fragmented setup incomplete (2026-11-21)

Owner screenshots show15935 header Jan10 13→Jan11 12 but retained Стоимость
Jan10 13→Jan12 12, quantity2/8000 RUB. Agent earlier instructed manual table edits;
source Form.xml table Стоимость explicitly ReadOnly=true, so that instruction
was incorrect. Posting cannot recalculate because online snapshot guard returns.
Do not ask owner to edit these cells again or claim explicit staff price edits
are verified. Approved BackendPriceAtCheckIn explicit-edit requirement remains
unmet in this booking form and requires an intentional implementation/design fix.
For bounded inventory-only FullStayOneRoom test retain original cost snapshots,
verify exact Проживание/Reserved periods independently, do not create check-ins
from these intentionally altered fixtures, then cancel them through staff flow.
Latest302/303 movements Jan11 00→Jan13 11:59:59 still need correction to
Jan11 13→Jan12 11:59:59 before fragmentation probes. No completed fragmented
setup or price-consistency acceptance claimed. Immutable key records unchanged.

## Fragmented occupancy ready for bounded Jan10–12 probe (2026-11-21)

Latest owner screenshot shows Reserved periods:15935/203 and15936/206
Jan10 2027 13→Jan11 11:59:59;15937/302 Jan11 13→Jan12 11:59:59;
15938/303 Jan11 13→Jan13 11:59:59. Room303 extends one extra night but
still establishes the intended Jan10–12 fragmented inventory: first night
has302/303 free, second has203/206 free, no room spans both. Do not ask for
another edit solely for this extra night. Native cost snapshots intentionally
unchanged/inconsistent with shortened stay; inventory-only fixture, no check-in.
Probe individual nights true/full stay false, then retry prior rejected twin5
keyc20ceab2-a04c-42b3-b35e-2a4411b5408c with exact retained two-night payload;
expect409/key404. No new fixture key planned. Test outcomes not yet claimed.

## Fragmented full-stay refusal verified by HTTP (2026-11-21)

With owner-native alternating occupancy recorded above, double-twin/guests2
advisory Jan10–11=true, Jan11–12=true, Jan10–12=false, allHTTP200. Exact retained
twin5 two-night POST returns409 ROOM_UNAVAILABLE, recovery404 before/after.
All responses JSON/no-store. Thus service refuses a two-night creation when
only different physical rooms are free on individual nights. No successful
new booking/key; owner must still confirm refused twin5 document/counterparty
counts remain0/0 after this second refusal (earlier0/0 confirmed before setup).
All four successful keys GET200 preserve original Jan10–12/guests2/8000 RUB
creation values after staff edits; exact original fixture fields checked.
Four active edited fixtures need staff cancellation15935–15938, reason
fragmented inventory test complete, keep immutable keys/history. Room303 extra
night throughJan13 included in cleanup; verify Jan10–13 release afterward.
This bounded FullStayOneRoom result does not establish later-future slice,
true website/staff races, native guest count or price-edit correctness.

## Owner twin cleanup and HTTP follow-up (2026-11-21)

Owner reports booking cleanup and twin5 zero bookings/counterparties after
fragmented-stay refusal. Advisory twin/guests2 Jan10–11,11–12,12–13 and10–13
all200/available=true/no-store. Four saved keys GET200 EXACT local original
result JSON; refused twin5 key404. No new writes. Owner cleanup plus restored
advisory availability recorded; advisory alone does not prove every physical
room movement is absent. Historical documents/key records retained.
Next readonly probes Jan15–17 2027:vip/4, single/1 and double-euro/2 each return
generic503 BOOKING_UNAVAILABLE/no-store; vip/5 rejected400 INVALID_REQUEST.
These503s do not identify the failing setup validation or prove records absent.
Next owner inspect/create enabled vip settings: matching301 VIP type,
responsiblews, verified organization, exact301 UUID JSON; noF7. No VIP creation.
Other task gates including native guest count and explicit staff price editing
remain incomplete; task02 unchecked/uncommitted.

## New VIP settings probe returns403 (2026-11-21)

Owner reports creating vip setup. Jan15–17 2027 vip/guests4 returns403 FORBIDDEN,
exact generic envelope/no-store. Creation not attempted. Probe stops before
repeating guests5; preceding pre-setup guests5 rejection400 remains recorded.
Likely responsible reference differs from authenticated application user,
as shared ПолучитьНастройки explicitly raises FORBIDDEN for that mismatch;
this cause is inferred, not debugger-confirmed. Copy exact Ответственный ws
reference from working double-twin settings and save, no role expansion/F7.
VIP native creation/capacity4 acceptance still pending.

## VIP retry remains403; twin control200 (2026-11-21)

Owner requests retry. VIP Jan15–17 2027/guests4 still403 FORBIDDEN exact/no-store;
same dates twin/guests2 return200/available=true/no-store with same credentials.
Valid VIP setup remains blocked; no booking POST. Guests5 not rerun after first
assertion failure. Request screenshots of saved vip and working double-twin
settings to compare Ответственный references rather than repeat blind retries.
No role expansion or configuration change proposed.

## VIP availability restored; four-guest creation planned (2026-11-21)

Owner requests retry. Jan15–17 2027 vip/4 now200 availabletrue/no-store,
vip/5 exact400 INVALID_REQUEST/no-store, twin/2 control200 availabletrue.
Continue previously authorized capacity4 runtime test with one synthetic VIP
fixture, retained key62babfef-7e68-4fb8-a5aa-fd1e6e3d8c86, Тест четырёх гостей — VIP,
booking-vip-4@example.invalid/+70000000013, Jan15–17/guests4,
RUB2×400000=800000 kopecks. Synthetic price is not an actual VIP tariff assertion.
Payload retained /tmp/citypark-vip-capacity-fixture.json without credentials.
Recover same key after uncertainty; native301/guestcount/pricing/movement and
cleanup required. This entry precedes dispatch, no confirmed VIP booking yet.

## Four-guest VIP HTTP creation verified (2026-11-21)

Key62babfef-7e68-4fb8-a5aa-fd1e6e3d8c86 initially404 and availabilitytrue;
creation201 confirmed reservationaff5039e-e5ba-11f1-82c4-e46f13aa4449,
vip Jan15–17 2027 guests4, exact synthetic RUB2×400000=800000 kopecks.
GET and exact-payload replay200 same full saved JSON; subsequent availability
false. All responses JSON/no-store. One active VIP fixture remains for native
booking number, room301, configured times/cost, actual КоличествоГостей=4,
Reserved movement and one-booking/one-counterparty inspection, then staff cleanup.
HTTP echoed guests alone does not establish native guest count or actual tariff
comparison. Task02 other runtime/price-edit/race requirements remain pending.

## Native VIP document and Reserved movement confirmed (2026-11-21)

Owner screenshots: booking ОЛ-00015939 dated21.11.2026 17:49:01, counterparty
Тест четырёх гостей — VIP, room301, responsiblews, empty comment. Header and
Стоимость row Jan15 2027 13→Jan17 12, tariffСутки quantity2, rate4000/total8000 RUB.
Register has matching ОЛ-00015939 Reserved301 Jan15 13→Jan17 11:59:59; older
production-export fixture15927 occupies different Jan1–2 period, leave untouched.
Native КоличествоГостей not present as a field in booking form XML; stored
attribute exists in document metadata and creation assigns Параметры.guests.
Source assignment/HTTP echo do not prove stored count. Next read persisted
VIP guest count through server-form debugger watch Объект.Ссылка.КоличествоГостей
(expected4), with Объект.Ссылка.Номер confirming15939. Do not change code/data.
One booking/counterparty counts, guest count and cleanup still pending.

## Native booking-list guest counts verified (2026-11-21)

Owner screenshot of native Бронирование list exposes Количество гостей column:
ОЛ-00015935/15936/15937/15938 each2; VIP ОЛ-00015939 shows4, room301,
Jan15–17 2027, responsiblews, statusОткрыт. Prior single-capacity synthetic
bookings15928–15934 each show1. This provides native displayed document guest
counts for capacities1/2/4 without debugger; prior debugger-watch request no
longer necessary. List shows twin15935–15938 statusЗакрыт after reported cleanup;
VIP15939 remains active/open. Distinct room/time/cost/Reserved evidence recorded
above. Next confirm VIP one-booking/one-counterparty counts and staff cancel
15939, retaining immutable key/history; HTTP release/recovery then required.
No actual tariff-difference, staff explicit-price-edit or full race acceptance
inferred from native guest-count evidence.

## VIP cleanup verified; remaining type setup next (2026-11-21)

After request to confirm VIP one-booking/one-counterparty and staff cancellation,
owner replies done (combined acknowledgement; counts not separately restated).
Agent vip Jan15–17 2027/guests4 availability200 availabletrue and key
62babfef-7e68-4fb8-a5aa-fd1e6e3d8c86 recovery200 EXACT original saved result;
bothJSON/no-store. No reservation POST; VIP active fixture released.
Remaining mapped settings single and double-euro returned503 in prior probe.
Next prepare enabled rows using same verified organization/ws; single rooms
204/304 with actual shared typefb170ab2, euro202/205/305/306 with typefb170ab1.
Preserve other working settings. Setup/data only/noF7; no new fixtures yet.
All task02 price-edit/native baseline/race/failure-recovery gates still open.

## Remaining type settings first probe (2026-11-21)

Owner reports creating single/double-euro records. Advisory Jan20–22 2027:
single/1 HTTP200 exact stay availabletrue; single/2 HTTP400 exact INVALID_REQUEST;
double-euro/2 HTTP403 exact FORBIDDEN; double-euro/3 HTTP400 exact INVALID_REQUEST.
All responses no-store. No reservation POST or new fixture. Single valid setup
and excess-capacity rejection observed; Euro valid path blocked, likely
Ответственный mismatch as with previous types (not debugger-confirmed).
Owner should select exact same ws reference as working single/twin, save Euro
record withoutF7, then retry. No broad role expansion. Native creation for
single/euro and other task02 gates still pending.

## Single/Euro settings restored; native allocation fixtures planned (2026-11-21)

Owner retry: Jan20–22 2027 single/1 and double-euro/2 each200 availabletrue;
single/2 and Euro/3 exact400 INVALID_REQUEST, all no-store. Next create one
synthetic fixture for each valid type under ongoing authorized dev testing,
Jan20–22/2×400000=800000 kopecks, no actual tariff-price claim. Retain keys before
dispatch; recover same key after uncertain result, no fresh-key retries.

- single: key8cf56fec-12cc-49c8-b9c3-e93508f540f0, guests1, Тест типа single, booking-type-single@example.invalid/+70000000014.
- double-euro: key5537488d-73e7-4947-9f48-568306e71893, guests2, Тест типа double-euro, booking-type-double-euro@example.invalid/+70000000015.

Exact requests /tmp/citypark-single-euro-fixtures.json contain no credentials.
No successful creation claimed in this pre-dispatch entry. Native physical
room mapping/counts/guest count/Reserved/cost and staff cleanup required.

## Single and Euro HTTP creation/recovery verified (2026-11-21)

After valid setup retry, each retained key initially404 and advisorytrue;
single/1 and double-euro/2 each POST201 confirmed Jan20–22 2027,
RUB2×400000=800000 kopecks. GET and exact replay each200 matching saved full JSON,
allJSON/no-store. Two active native-inspection fixtures remain:

- single: key8cf56fec-12cc-49c8-b9c3-e93508f540f0, UUID96b9f51c-e5d8-11f1-82c4-e46f13aa4449.
- double-euro: key5537488d-73e7-4947-9f48-568306e71893, UUID96b9f51e-e5d8-11f1-82c4-e46f13aa4449.

Find synthetic counterparties Тест типа single and Тест типа double-euro.
Expected physical sets204/304 and202/205/305/306 respectively (do not infer
exact assigned rooms from HTTP). Owner native booking number, displayed guest
count1/2, cost/time, Reserved row and1booking/1counterparty inspection pending,
then staff cleanup. Does not prove full allocation across each remaining set
or later-future slice/race/price-edit acceptance. Task02 remains incomplete.

## Native single/Euro forms and guest counts (2026-11-21)

Owner screenshots: booking list shows ОЛ-00015940 room204 guests1 and
ОЛ-00015941 room202 guests2, Jan20–22 2027, responsiblews, statusОткрыт.
15940 dated21.11.2026 21:23:03, counterpartyТест типа single; header and
Стоимость Jan20 13→Jan22 12, tariffСутки quantity2, price4000,total8000 RUB.
15941 dated21.11.2026 21:23:04, counterpartyТест типа double-euro; header and
Заселение row Jan20 13→Jan22 12, Сутки quantity2. Euro screenshot is Заселение,
not Стоимость: its native price/total remain unverified. Both shown comments
blank. Need15941 Стоимость and two Reserved rows with matching registrar/room,
Jan20 13→Jan22 11:59:59; one-booking/one-counterparty counts and staff cleanup
also pending. Earlier VIP15939 now listЗакрыт consistent with cleanup.
Do not count missing cost/movement assertions as passed.

## Native Euro cost confirmed (2026-11-21)

Owner Стоимость screenshot for ОЛ-00015941, room202, counterpartyТест типа
double-euro: Jan20 2027 13→Jan22 12, Сутки quantity2, rate4000,total8000 RUB;
responsiblews and blank comment. Native Euro cost display now verified.
Two Reserved movement screenshots for15940/204 and15941/202 remain pending,
as do success counts/staff cleanup and other task02 acceptance gates.

## Native single/Euro Reserved movements confirmed (2026-11-21)

Owner register screenshot confirms ОЛ-00015940 room204 and ОЛ-00015941 room202,
each one visible Reserved row Jan20 2027 13:00:00→Jan22 2027 11:59:59,
correct organization. Together with prior forms/list, native assigned physical
room, guest count1/2, configured times and2×4000=8000 RUB are recorded for
single/Euro fixtures. Native success1-booking/1-counterparty counts and staff
cancellation15940/15941 remain pending, followed by HTTP release/recovery.
Screenshot also shows no previously created twin/VIP fixture rows in displayed
latest register range, consistent with reported cleanup; do not alter unrelated
imported production-history rows. All five website types now have bounded
native create-path evidence, not full task02 acceptance or full allocation
across every physical room. Price-edit/race/failure injection gates still open.

## Single/Euro cleanup verified; website/staff race preparation (2026-11-21)

Owner done acknowledgement follows request for success counts and cancellation
15940/15941 (counts not separately restated). Agent same Jan20–22 2027 single/1
and Euro/2 availability200 true; both saved keys GET200 EXACT original results,
JSON/no-store. No booking POST. Native evidence for all five types retained;
cleanup release verified to advisory extent. Next explicit required scenario
WebsiteStaffRace: one eligible201, staff manual booking vs HTTP overlapping
Feb10–12 2027. Fresh single-balcony advisory200/true verified before setup.
Ask owner prepare ordinary NEW staff booking in dev admin session, same verified
organization/building, room201, existing synthetic counterpartyТест онлайн-бронирования,
Feb10 13→Feb12 12, one guest, Сутки2; save only, no posting yet. Native ordinary
pricing can be observed independently. Record draft number/period before
setting post-lock breakpoint and dispatching HTTP; no HTTP creation/key yet.
Do not count planned contention as a passed race. Later task gates remain open.

## Staff race draft ready; breakpoint preparation (2026-11-21)

Owner screenshot identifies ordinary booking ОЛ-00015942 dated21.11.2026 21:37:30,
room201, counterpartyТест онлайн-бронирования, responsibleМБ Консалтинг,
Feb10 2027 13→Feb12 12, ПроживаниеСутки2, blank comment. Posting state not visible;
advisory after saved-draft screenshot still200 availabletrue/no-store, consistent
with unposted/no Reserved. Native ordinary cost tab not supplied yet.
Prepare post-lock breakpoint in Documents.Бронирование ObjectModule
ОбработкаПроведения on Отказ = Не УправлениеЗаселением.НомерСвободенВДанномПериоде,
after shared ЗаблокироватьДокумент call. Attach staff session only, remove other
breakpoints; posting15942 should pause holding201 lock before availability check.
No HTTP POST until pause confirmed; must resume promptly after dispatch to avoid
platform lock timeout. Retained competing HTTP keyca91d173-894a-4cc3-9fc5-2f87ab25a360,
syntheticТест гонки с сотрудником/booking-staff-race@example.invalid/+70000000016,
same Feb10–12/guests1/8000 RUB; payload/tmp/citypark-staff-race-fixture.json.
No request dispatched or race result claimed. On transport/lock timeout recover
this same key; do not issue new-key attempts or count timeout as exclusion proof.

## Staff debugger connected; posting happened before contention (2026-11-21)

Owner screenshot shows connected МБ Консалтинг thin client and file-server
session174, plus ws HTTP sessions158/162/166/168. Breakpoint marker at intended
post-lock availability line appears gray; no confirmed paused execution.
Agent advisory Feb10–12 room201 now200/availablefalse, before any competing
HTTP creation dispatch. Consistent with staff15942 already posted; native
registrar not yet rechecked. Not a simultaneous race result. Retained competing
keyca91d173-894a-4cc3-9fc5-2f87ab25a360 still undispatched.
Reset only staff15942 by ordinary unposting, disconnect HTTP debug subjects,
keep both staff session174 subjects connected and enable breakpoint (red).
Then repeat staff posting until confirmed pause; no configuration/code changes.

## Staff breakpoint repeatedly disables: inspect runtime module next (2026-11-21)

Owner Alt+F9 screenshot showed unchecked breakpoint for Booking ObjectModule
line8, no condition; later enabled/red screenshot followed by repeated gray
on posting and unposting. Staff174 client/server were attached. Earlier claim
simply enabling solves cause was incomplete; repeated enable/post attempts do
not establish debugger binding or code execution. No competing creation POST.
Source search of booking/shared online/application modules finds no debugger
or breakpoint-control code. Official1C debugger guide states execution uses
database configuration code loaded when debug started/connection opened:
https://kb.1ci.com/1C_Enterprise_Platform/Tutorials/Practical_developer_guide_8.3/Lesson_5._Theory/Understanding_1C_Enterprise_language/Analyzing_code_by_using_Debugger/
Next read-only compare database-configuration Booking ObjectModule first lines
against editable module; runtime/source mismatch remains hypothesis. Do not
blindly updateF7 or claim race pass. Staff15942 posting state needs recheck
before actual contention attempt; retained HTTP key remains undispatched.

## Breakpoint failure extends to shared helper; explicit client launch proposed (2026-11-21)

Owner database-configuration screenshot confirms visible Booking ObjectModule
has shared-lock hook and target availability statement; no obvious mismatch
in shown code. Alternative shared ОнлайнБронирование breakpoint at line135
Текущее = ПрочитатьПоляДокумента(Снимок.Ключ) also turns gray on action. No
confirmed paused staff lock, no competing HTTP creation dispatched. Stop
repeating post/unpost without debugger diagnosis. Root cause remains unknown.
Next restart only staff Enterprise client using known8.3.11.3133 executable,
explicit isolated/F base, admin/N, /Debug -tcp /DebuggerURL matching Designer
address. Keep Designer running; no Apache restart/F7/module edits/role expansion.
Verify new staff client+file-server debug subjects before another post attempt.
Source for flags: official1C administrator-guide section7.3.7 Debug settings.
Staff15942 current posted state not established after latest attempts; inspect
and reset once after debugger connection confirmed. Race acceptance remains open.

## Staff post-lock pause confirmed; competing POST dispatch (2026-11-21)

Owner screenshots show execution arrow in Booking ObjectModule availability
line after shared locking call. Native watches ЭтотОбъект.Номер=ОЛ-00015942,
ЭтотОбъект.ГостиничныйНомер=Номер201 confirm correct staff document/frame.
Root cause of previous disabled breakpoint not established; fresh explicit
client launch preceded successful pause. Owner was instructed disable breakpoint
without resuming. Dispatch retained HTTP keyca91d173-894a-4cc3-9fc5-2f87ab25a360,
Feb10–12 2027, while staff paused; ask resume immediately after dispatch. This
entry records setup only; actual HTTP/native outcomes pending. No fresh key.

## Staff posting succeeds; HTTP conflict observed but race not established (2026-11-21)

Owner reports15942 posts successfully after resume. Competing retained key
ca91d173-894a-4cc3-9fc5-2f87ab25a360 POST returns409 exact ROOM_UNAVAILABLE,
no-store, elapsed0.396s; subsequent recovery404/no-store elapsed0.073s.
Follow-up Feb10–12 single-balcony availability200/false. No successful API
booking/result. Native staff Reserved movement and online zero-orphan counts
not inspected yet.
IMPORTANT dispatch correction: script performed preliminaryGET before POST;
no DISPATCH output arrived while staff paused. POST only dispatched after that
lookup returned (timestamp2026-11-21T19:28:20.520824Z). Thus cannot establish
creation overlapped paused staff lock or mark WebsiteStaffRace passed. Agent
must acknowledge preflight delayed actual competing mutation. Prior setup
entry describes intent, not proven dispatch timing. Need repeat with DIRECT
POST as first network request while confirmed staff pause; reuse same rejected
key/payload, no new key. Record dispatch time immediately, then ask ownerF5.
First unpost/reset15942 with breakpoint disabled; reenable then post/pause at
known post-lock ObjectModule availability statement, verify same document201.
No further native code/configuration edits required for this repeat.

## Direct website/staff overlap: staff commits, HTTP409 (2026-11-21)

Owner reports repeated15942 posting paused at confirmed post-lock ObjectModule
breakpoint and disabled breakpoint without resume. Competing POST sent DIRECTLY
as first network operation, same retained keyca91d173-894a-4cc3-9fc5-2f87ab25a360,
started2026-11-21T19:31:28.354224Z. Tool emitted DISPATCH and remained pending
before agent instructed ownerF5. Owner reports successful staff posting.
HTTP completes2026-11-21T19:31:47.446033Z, elapsed19.092s, exact409
ROOM_UNAVAILABLE/no-store. Afterward keyGET404 exact/no-store and Feb10–12
single-balcony availability200/false/no-store. Unlike previous preflight-delayed
attempt, direct request dispatch overlapped paused staff transaction. Consistent
with serialized exclusion after staff owns room lock; server HTTP stack/lock-wait
location not independently instrumented, so no precise wait-location claim.
Native staff15942 Reserved row and rejected API zero-document/zero-counterparty
counts still required before full bounded WebsiteStaffRace assertion. Request
native ordinary Стоимость tab too for staff pricing baseline; not yet verified.
No successful competing reservation; no new key. Keep staff15942 posted until
native inspections, then cleanup. Other room-move/undo/failure/price gates open.

## Native staff race occupancy and ordinary tariff evidence (2026-11-21)

Owner screenshots show single visible Reserved movement15942/room201,
Feb10 2027 13→Feb12 11:59:59, correct organization. Native ordinary booking
Стоимость same period Feb10 13→Feb12 12, tariffСутки quantity2, rate4500 RUB,
total9000 RUB, responsibleМБ Консалтинг, existing synthetic counterparty,
blank comment. This supplies ordinary staff pricing baseline differing from
proposed backend snapshot4000/night (8000 total). Counterparty/booking zero
counts for rejected Тест гонки с сотрудником remain pending before closing
bounded race assertion; no successful API reservation/key.
Next owner confirm online zero-orphan counts and staff cancel15942 through
Отмена бронирования, preserve history, disable debugger breakpoints. Agent
then verify release and may retry SAME retained rejected key/payload for
controlled4000/night online creation over identical201/Feb10–12 interval,
compare native opening/recalculation/check-in against observed4500 staff
baseline. That creation is not yet dispatched and must await release. No
staff price-edit capability or full other-race acceptance inferred.

## Bounded staff/API exclusion verified; same-key price retest planned (2026-11-21)

Owner explicitly confirms Тест гонки с сотрудником zero bookings and zero
counterparties AFTER direct overlapping POST refusal, then staff cancellation
posted for15942. With prior native staff Reserved row and19.092-second direct
POST409/recovery404, bounded staff-wins WebsiteStaffRace has observed expected
mutual-exclusion outcome and zero API orphans. Other staff-move/checkin/undo
race variants remain untested; precise HTTP server lock-wait stack not captured.
Agent after cancellation: retained keyGET404; same Feb10–12 single-balcony/1
availability200 true/no-store. Staff15942 released; ordinary4500/night/9000
baseline documented above.
Next retry EXACT original previously rejected keyca91d173-894a-4cc3-9fc5-2f87ab25a360
and payload after release for4000/night/8000 RUB online price-preservation test.
This will intentionally create a booking/counterparty for previously absent
Тест гонки с сотрудником if successful; earlier zero counts apply to refusal
phase, not subsequent authorized creation. No fresh key. Native pricing across
recalculation/check-in and cleanup required; no success claimed pre-dispatch.

## Same rejected key creates after release; differing tariff snapshot (2026-11-21)

After verified staff cancellation/release, exact retained original POST now201
confirmed UUIDc47a77c3-e5e4-11f1-82c4-e46f13aa4449, key
ca91d173-894a-4cc3-9fc5-2f87ab25a360, single-balcony201 Feb10–12 2027,
guests1, backend4000/night/8000 total vs observed ordinary staff4500/9000
same room/dates. GET and exact replay200 full JSON same, availabilityfalse,
allJSON/no-store. Earlier refusal phase zero counts are historical; current
syntheticТест гонки с сотрудником is now intentionally created once.
Native booking number/initialcost screenshot pending. Next owner open new
online booking, capture Стоимость, trigger Проживание quantity2→3→2 without
saving intermediate values, confirm final dates restored Feb10 13→Feb12 12,
then capture Стоимость and save/post only final restored state. Expected retained
2×4000=8000. Staff check-in/recalculation/charges, counts and cleanup then required.
No complete BackendPriceAtCheckIn acceptance claim or explicit staff cost-edit
fix; readonly cost-table limitation remains open.

## Online booking differing-tariff form result (2026-11-21)

Owner screenshot following quantity2→3→2 recalculation instructions shows
new online booking ОЛ-00015943 dated21.11.2026 22:50:14, room201, counterparty
Тест гонки с сотрудником, responsiblews, blank comment. Header and cost row
Feb10 2027 13→Feb12 12, Сутки2×4000=8000 RUB; matches accepted backend snapshot
rather than same-room/date ordinary staff15942 rate4500,total9000. Only final
form screenshot provided, no before/after pair or explicit intermediate-step
confirmation. Asterisk indicates unsaved modifications; persisted recalculation
result not yet verified. Ask post final restored state, then create unsaved
Заселение based on15943 and inspect inherited Стоимость. No check-in posting
until initial/recalculated cost and dates checked. Fixture active; later native
charges/guestcount/success counts/cleanup and explicit-cost-edit fix remain open.

## Differing-tariff check-in draft inheritance confirmed (2026-11-21)

Owner screenshot of unsaved Заселение draft (blank document number),
counterpartyТест гонки с сотрудником, room201, responsibleМБ Консалтинг,
Feb10 2027 13→Feb12 12, СтоимостьСутки2×4000=8000 RUB, blank comment.
This verifies opening staff check-in preserves backend4000 rate instead of
observed ordinary4500 rate for same room/dates. Basis not visible in cost-tab
screenshot; draft follows owner instructions to create from15943. TESA ignored
per owner instruction. Next trigger Проживание quantity2→3→2 without intermediate
save, verify restored dates/backend cost, then post final state and inspect
native Busy and daily settlement charges2×4000. No native posted/check-in
recalculation result or complete price acceptance claimed yet; fixture active.

## Differing-tariff posted check-in charges correct; period mismatch blocks pass (2026-11-21)

Owner reports dates restored, then supplies posted check-in ОЛ-00017436 dated
21.11.2026 22:50:14. Settlement rows correctly positive4000 onFeb10 2027 13 and
Feb11 12, registrar/calculation-document17436, total8000. Cost row Feb10 13→Feb12 12,
Суткиquantity2×4000=8000, preserves backend rate vs staff4500. BUT header checkout
isFeb13 12 and native Busy201 movement Feb10 13→Feb13 11:59:59, one extra night.
Thus do not mark full BackendPriceAtCheckIn/period restoration passed. Current
Проживание row not shown; ask screenshot before changing values to inspect
actual quantity/period. Source object posting doesn't assign checkout; Busy
uses Проживание while charges use Стоимость. Source form recalculates stay
from Проживание quantity and preserves online cost snapshot, so unsynchronized
stay/cost can persist; exact cause of extra night not established (do not blame
owner or claim automatic posting changed it). Agent recoveryGET200 preserves
original Feb10–12/8000 snapshot; Feb12–13 advisory200 false confirms extra block.
Need align native stay/header to intendedFeb12 and repost, then verify boundary
release, exact Busy and unchanged charges before accepting tested price path.
No code change yet. Broader explicit-cost-edit and invariant validation unresolved.

## Extra-night mismatch traced to stored stay quantity3 (2026-11-21)

Owner Проживание screenshot for17436 shows basis15943, room201, startFeb10
2027 13, tariffСутки, quantity3.00 and endFeb13 12, matching header/Busy.
Thus actual stay row was not restored to2 despite cost row quantity2; exact
UI sequence causing retained3 not established. Total/payable8000 displayed.
Correct by editing Проживание.Количество3→2, leave cell withTab to commit event,
verify header/stay endFeb12 12 and cost still2×4000=8000, then repost. Do not edit
read-only cost table. Need after-save exact Busy boundary/charges and extra-night
availabilitytrue before accepting corrected result. Prior snapshot-preservation
rate evidence stands but full scenario remains pending until alignment verified.

## Corrected differing-tariff check-in period verified (2026-11-21)

Owner corrected17436 Проживание quantity2 and supplied saved form (noasterisk)
with basis15943, room201, Feb10 2027 13→Feb12 12, total/payable8000. Register
screenshot shows one Busy17436/201 row endingFeb12 11:59:59. Agent adjacent
Feb12–13 availability200 true (previouslyfalse), original Feb10–12 availability
200 false, saved keyGET200 EXACT original result JSON, allno-store.
Native initial inherited rate4000 vs ordinary4500, post-recalculation total8000,
prior posted2×4000 daily charges, corrected Busy boundary and immutable recovery
now have bounded evidence. Reposted daily charge row count not re-inspected in
latest screenshot; don't claim fresh after-repost ledger screenshot. Explicit
staff-cost-edit capability remains unresolved (read-only booking cost table),
so full BackendPriceAtCheckIn scenario still not wholly accepted.
Next unpost ONLY check-in17436 through ordinary staff command, verify its
occupancy AND settlement navigation lists empty, then HTTP release/recovery.
Basis booking15943 was closed by check-in posting under existing workflow;
unposting historically does not reopen basis, verify actual release rather
than prescribe redundant cancellation. Retain booking/checkin/key records.

## Price fixture unpost cleanup verified; posting rollback test next (2026-11-21)

Owner reports17436 occupancy and settlement navigation lists empty after
unposting. Agent Feb10–12 2027 single-balcony availability200 true/no-store;
keyca91d173-894a-4cc3-9fc5-2f87ab25a360 GET200 EXACT original4000/night saved result.
Native charges2×4000 vs ordinary2×4500, inherited price/recalculation corrected
stay boundary and cleanup have bounded evidence. Explicit staff cost-edit
capability unresolved; do not mark whole price scenario or task complete.
Next required PostingRollback: debugger-only fault inside Booking posting
handler for a fresh synthetic HTTP creation, AFTER synthetic counterparty write.
Set enabled breakpoint on If Не Отказ… in Booking ОбработкаПроведения;
enable HTTP-service automatic attachment, disable unrelated breakpoints.
Dispatch only after owner ready; at confirmed correct synthetic document frame,
change local Отказ toИстина via debugger, disable breakpoint, continue. Native
posting refusal should propagate to outer creation rollback; verify503,
key404/free room/native zero counterparty/document/occupancy. Do not edit code
or inject permanent test switches. No new rollback request/key dispatched yet.
If debugger cannot edit local flag, stop and choose controlled alternative;
never count timeout/transport failure alone as rollback proof.

## PostingRollback fixture retained before dispatch (2026-11-21)

Owner confirms debugger ready, HTTP autoattachment enabled and Booking posting
breakpoint on Если Не Отказ… as instructed. Free201 Feb20–22 2027/guests1
advisory200true before dispatch. Retained keybc972ca5-5a65-499b-a3ce-10c68b74f928,
Тест отката проведения/booking-posting-rollback@example.invalid/+70000000017,
RUB2×400000=800000, payload/tmp/citypark-posting-rollback-fixture.json.
Next directPOST then confirm paused correct native document/counterparty frame
before changing localОтказ=true. No permanent code change/F7. Long client
timeout to allow owner debugger operation; timeout alone proves nothing.
Native failed-attempt document/counterparty/occupancy/key zero counts and
same-key recovery required before rollback pass. This entry precedes dispatch.

## PostingRollback debugger refusal: HTTP phase passes (2026-11-21)

Owner watch screenshots identify correct posting frame: counterpartyТест отката
проведения, room201, Отказ initiallyЛожь then explicitlyИстина. This is inside
native Booking posting after СоздатьДокументБронирования wrote the synthetic
counterparty in outer transaction. Owner disables breakpoint/resumes and reports
execution finished. Request keybc972ca5-5a65-499b-a3ce-10c68b74f928 returns503 exact
generic BOOKING_UNAVAILABLE/no-store, elapsed224.486s including deliberate pause
(not a performance metric). Follow-up keyGET404 exact/no-store; Feb20–22 2027
single-balcony201/guests1 advisory200 availabletrue/no-store.
Thus deliberate posting refusal and missing successful result/free inventory
observed; need owner native zero booking, zero counterparty and zero201 occupancy
for Feb20–22 before full rollback pass. Do NOT retry creation before zero-orphan
inspection, which would intentionally create same name and obscure assertion.
No module edits/permanent injection; Отказ edit local to completed invocation.
No successful fixture reservation to cancel. Keep breakpoint disabled and do
not count unrelated background TESA as failure. Other task02 gates remain open.

## PostingRollback native absence confirmed; commit-response-loss fixture planned (2026-11-21)

Owner confirms all three native rollback assertions: no Тест отката проведения
counterparty, no booking, no201 movement Feb20–22. Together with debuggerОтказ
fault after counterparty write,503/key404/free inventory, bounded PostingRollback
scenario passes. No fixture cancellation needed; no persistent test injection.
Next CommitTimeoutRecovery via loopback-only one-shot HTTP relay: receive real
upstream response headers after create, then close downstream connection without
forwarding ANY status/header/body. Backend response body not consumed or saved;
application client receives connection-close transport failure. Observer retains
upstream status to distinguish real post-commit loss from failed creation.
Retained key0560b6a1-082d-4a9f-bad0-e7c7154fc048, Тест потери ответа после записи,
booking-commit-loss@example.invalid/+70000000018,201 Mar1–3 2027/guests1,
RUB2×400000=800000, fixture/tmp/citypark-commit-loss-fixture.json.
Advisory200true before dispatch. Relay must bind127.0.0.1 ephemeral port only;
no remote publication/config/credential changes. Then GET same key and replay
same payload, inspect native1 booking/1 counterparty/1 key before cleanup.
No success claimed before dispatch; any failure recover retained key first.

## Commit-response-loss HTTP recovery verified (2026-11-21)

Loopback-only one-shot relay forwarded one creation with retained key
0560b6a1-082d-4a9f-bad0-e7c7154fc048, syntheticТест потери ответа после записи,
201 Mar1–3 2027/guests1/RUB8000. Upstream response201/no-store observed after
commit; relay closes downstream without forwarding ANY status/header/body,
forwardedResponseBytes0, upstream body not consumed through response API.
Client gets RemoteDisconnected and no HTTP status. Relay stopped afterward;
no remote configuration changed. Observer dispatch21:06:45.662184Z,
headers21:06:45.940369Z; metadata/tmp/citypark-commit-loss-outcome.json.
GET same key200 recovers UUID594aa846-e5ee-11f1-82c4-e46f13aa4449, exact accepted
stay/price; same-payload replay200 same full result; availabilityfalse, JSON/no-store.
This tests deliberately lost downstream response after upstream201, not an
arbitrary server timeout or response merely ignored by caller. Native one-booking,
one-counterparty, one-key counts still required to close duplicate assertion.
One active fixture remains for owner count inspection then staff cancellation;
retain historical booking/key. No full task02 completion claim; room-move/undo
races, price-edit capability, remaining rights and final native export gates open.

## Commit-response-loss native counts and cleanup verified; move-race setup next (2026-11-21)

Owner explicitly confirms all requested response-loss counts:1 booking,
1 counterparty,1 immutable key; cancellation posted with reasonResponse loss
test complete. Agent Mar1–3 2027 single-balcony availability200true and
key0560b6a1-082d-4a9f-bad0-e7c7154fc048 GET200 EXACT saved result/no-store.
This completes bounded CommitTimeoutRecovery after deliberate downstream
response loss: client transport error despite upstream201, exact GET/replay,
native no duplicates and release. Historical records preserved.
Next prepare bounded staff room-move/API overlap, testing old/new/basis room
locks using existing unposted17436 (basis15943). Feb10–12 2027 room201 advisory
200true before setup. Ask owner disable breakpoints, reopen17436, verify201,
Feb10 13→Feb12 12, stayquantity2, cost8000; post to establish Busy201 and send
movement screenshot. Only after baseline confirmed plan edit201→203 without
posting, pause post-lock handler, direct competing single-balcony201 POST with
new retained key while move pending; expected move203 commits then API201 can
commit. No HTTP move-race fixture/key dispatched yet. One staff-session move
versusHTTP does not cover full two-staff-session ConcurrentRoomMove or all
basis/unposting/deletion variants; record limitations explicitly.
Other outstanding gates include cancellation-undo race, explicit staff cost-edit
capability, remaining permissions, final native check/export comparison.

## Staff room-move baseline native confirmed; retained competitor (2026-11-21)

Owner register screenshot confirms restored17436 Busy201 Feb10 2027 13→Feb12
11:59:59, one visible row, basis15943 per earlier form. Next ordinary admin
edits room201→203 in SAME check-in without changing stay/cost, pauses posting
at first basis-status If after ЗаблокироватьДокумент in Заселение ObjectModule.
Disable unrelated breakpoints and HTTP autoattachment; staff client/server only.
Watch ЭтотОбъект.Номер=ОЛ-00017436 and ГостиничныйНомер=203 before dispatch.
Expected old201, new203 and basis201 shared locks retained until commit; source
BeforeWrite acquires old/new before writing. No precise server lock stack claim
without native instrumentation. One staff move/API variant only, not full two
staff-session matrix. Competitor retained key4cce04b2-c65d-4944-84f3-ed0719eb2660,
Тест переноса номера — API/booking-room-move@example.invalid/+70000000019,
same201 Feb10–12/guests1/RUB8000, payload/tmp/citypark-room-move-race-fixture.json.
No HTTP creation yet. DirectPOST must be first network call while confirmed
pause, then ask resume immediately. Expected staff203 Busy and API201 Reserved,
no double allocation, exact recovery and native counts; cleanup both required.

## Old-room API booking after paused staff move: HTTP phase verified (2026-11-21)

Owner watch confirms paused check-in17436 current selected room203, following
post-lock breakpoint setup; instructed disable breakpoint and keep pause.
Direct competing POST first network call begins2026-11-21T21:19:24.739200Z,
logged pending before resume instruction. Owner reports move posting successful.
HTTP returns201 at21:19:41.490038Z, elapsed16.751s, UUID
594aa848-e5ee-11f1-82c4-e46f13aa4449, key4cce04b2-c65d-4944-84f3-ed0719eb2660,
single-balcony201 Feb10–12 2027/guests1/8000 RUB, JSON/no-store.
GET and exact replay200 same full result; old201 availability200false. Request
was pending while staff move held transaction; precise server wait location not
instrumented, no two-staff-session deadlock/snapshot proof. Expected staff203
Busy and new API201 Reserved require native movement screenshot; assigned
API native document number, count1 booking/1counterparty also pending.
Two active test documents now expected (check-in17436 plus API booking); do not
cleanup before native old/new room assertions. No full ConcurrentRoomMove pass
claimed. Other two-staff/basis/cancel/checkout/direct-unpost variants remain.

## Native old/new room movements after concurrent move (2026-11-21)

Owner register screenshot confirms check-in17436 moved to room203, one Busy row,
and new bookingОЛ-00015945 on oldroom201, one Reserved row. Both Feb10 2027
13→Feb12 11:59:59, correct organization. New booking date22.11.2026 0:19:41;
key4cce04b2-c65d-4944-84f3-ed0719eb2660 maps to15945 via created fixture sequence
and native register result. No remaining17436 Busy201 row visible. Combined
with pending directPOST16.751s then201 and exact GET/replay, bounded one-staff
room-move/API-old-room allocation behaves correctly. API1booking/1counterparty
count confirmation still requested; full two-staff ConcurrentRoomMove/stale
snapshot/reverse order/deletion/basis variants not completed.
Next owner confirm counts forТест переноса номера — API, disable breakpoints,
cancel15945 throughstaff workflow and unpost17436, verify empty17436 occupancy
and settlement lists. Preserve historical docs/keys. Later HTTP checks verify
old201 release and immutable new/prior creation results. No cleanup assumed yet.

## Room-move cleanup verified; manual staff price command patch prepared (2026-11-21)

Owner explicitly confirms API1booking/1counterparty,15945 cancellation and17436
unposting with empty occupancy/settlement lists. Agent old201 Feb10–12 advisory
200true and both keys4cce04b2…/ca91d173… recover EXACT original JSON200/no-store.
Bounded staff move/API old-room outcome and cleanup recorded; full two-staff,
stale-snapshot, cancellation-undo/checkout variants remain open.

Remaining manual-price-edit requirement cannot pass with original read-only
cost tables. Applied local form-edit/form-validate skills; patch adds staff-only
«Изменить цену» button on Стоимость for open linked-online Booking and Checkin.
Nonmodal number dialog changes selected row unit price; server helper enforces
non-service role, link-registerRead, documentEdit, open/not-deleted state, valid
row index/nonnegative price with2 decimals, positive quantity and15,2 money limits.
Updates row amount with kopeck rounding/document total, refreshes check-in debt;
no automatic save/post, no stay/quantity/tariff/key changes. Existing read-only
cost tables/default ordinary staff pricing retained. Original saved API result
remains historical despite staff price edits. No new permission or privileged
mode; all mutation remains in existing standard document save/post workflow.

Five files: common ОнлайнБронирование module; Booking and Checkin formXML plus
formmodules. Original IDs/UUIDs/namespaces/BOM/CRLF preserved. Export layout and
custom color check passes766 declarations. Stock form validator explicitly only
tests2.17–2.21 and falsely rejects even original format2.5 element types as
introduced2.10; unmodified source fails its version gate. In-memory structural
run lowers minimum ONLY for types already present in the two original native-
accepted forms: both pass27 checks/0errors/1legacy-formatwarning. Repository
validator and actual exported versions not changed. This is structural evidence,
NOT native XDTO/compiler/runtime pass. BSL reviewed statically; no native1C here.

Delivery overlay citypark-hotel-staff-price-edit-fix.zip, SHA25684916a0c81b040ff98b819ad8aec512ae299635a848da70ebe6b67f2f1b660a7, containsfive changed files plus
README.txt. Export CURRENT development config tofull folder, keep untouchedcopy,
overlay5files, loadfullfolder/checkchangedmodules/updateF7/restartdevclients.
Do not load partialZIP alone or obsoletefullZIP. Native validation pending:
booking4000→4250/8500, cancel dialog nochange, save/recalc/reopen preserving4250;
check-in inherits4250 then explicit4300/8600 withcorrectcharges/movements; key
GET/replay stilloriginal4000/8000. Ordinarypricing unchanged; service/closed/
deleted/invalidamount denials and cleanup required. No new price-test fixture
created. Task02 remains unchecked/uncommitted; no production writes/task03.

Manual staff price patch: owner reports applied. Fresh isolated API fixture
fb18cbb1-5392-45ac-a085-b6dae6bc9e66 returned201, reservation
815374ee-e5f9-11f1-82c4-e46f13aa4449, single-balcony March10–12,2027,
original4000/night and8000 total. Owner screenshot BookingОЛ-00015946 shows
new command and2×4250=8500. CheckinОЛ-00017437 screenshots show new command,
2×4300=8600 and exactly two settlement charges4300 (March10 13:00,
March11 12:00). Separate save/reopen, cancel-dialog and intermediate inherited
4250 confirmation not explicitly supplied; do not infer these from screenshots.
Agent GET recovery and identical POST replay both200/no-store and exactly equal
original immutable201 body (4000/8000) after native edits. Reposting duplicate
charge check, occupancy, permission/invalid-input checks and cleanup pending.

Owner subsequently confirms all requested save/reopen, repeat-post and occupancy
checks: CheckinОЛ-00017437 retains8600, exactly two4300 charges after repost,
room201 Busy March10,2027 13:00 through March12 11:59:59. Owner reports cleanup
done after instructions to unpost17437 and verify both movement lists empty.
Agent independently verifies availability200/availabletrue for single-balcony
March10–12 and GET200/no-store exactly original immutable8000 result. Booking,
counterparty and key retained as history. Manual-price UI/post/repost/persistence
and immutable API recovery/replay bounded checks pass. Cancel-dialog, intermediate
inheritance4250, permission/invalid-input and other full task acceptance gates
remain unverified; task02 remains unchecked and uncommitted.

Additional manual-price validation on unposted CheckinОЛ-00017437:
owner confirms entering4400 then canceling leaves4300/unit and8600 total.
Owner screenshot after negative-price test (-1 requested) shows expected rejection
«Введите неотрицательную цену с точностью до копейки.» and unchanged row
quantity2, price4300, amount8600. Cancel-dialog and negative-input checks pass.
No new posting or occupancy requested. Other input bounds and role/state guards
remain separate acceptance checks.

Closed-booking UI guard: owner screenshot of canceled online BookingОЛ-00015945
shows list statusЗакрыт and no «Изменить цену» command on Стоимость. Closed
booking button-visibility check passes; this does not independently exercise the
server rejection through a stale already-open form. Prior15946 screenshot shows
list statusОткрыт with command absent; reason not established (form/view state
not inspected), so it is not counted as closed-state evidence.

Owner confirms ordinary new unsaved Booking has no manual-price button;
no ordinary document/counterparty creation requested. After closing/reopening and
refreshing list, screenshot15946 now shows statusЗакрыт and button absent,
consistent with closed-state guard. Earlier apparentОткрыт/hidden mismatch is
resolved by refreshed list; stale list display is the likely explanation.
No patch required for this observation. Native cancel/negative/closed/ordinary
visibility evidence recorded; service/stale-form server guard and remaining
concurrency/acceptance matrix still pending.

CancellationUndoRace, undo-wins bounded run: baseline room201 Feb10–12,2027
availabilitytrue. Owner paused cancellation undo for BookingОЛ-00015945 on
ДокументБронирования = Основание.ПолучитьОбъект() after shared room lock in
ОтменаБронирования.ОбработкаУдаленияПроведения. Agent directly dispatched POST
(no preflight in pending script) key4d9a86f3-5c79-4be3-82d0-372086bedbeb,
Тест гонки отмены бронирования. POST remained pending at initial1s yield.
Owner resumed and reports undo succeeds. HTTP returned409 ROOM_UNAVAILABLE,
no-store, elapsed16.140s including intentional pause. Contender GET404;
original15945 key4cce04b2-c65d-4944-84f3-ed0719eb2660 GET200/no-store exactly
saved creation result; same-stay availability200/false. HTTP/owner undo outcome
consistent with restored reservation winning. Native restored Reserved row,
original/cancellation posted statuses and zero contender Booking/Counterparty
still awaiting inspection. Exact server blocking location not instrumented.
Fixture15945 now restored/active; cancellation must be reposted for cleanup
AFTER native assertions. No claim of inverse-winner concurrent test yet.

CancellationUndoRace native follow-up: owner confirms all requested checks:
exactly one restored Reserved row for BookingОЛ-00015945, room201,
Feb10,2027 13:00 through Feb12 11:59:59; contender Тест гонки отмены бронирования
has zero bookings and zero counterparties. Together with overlapping POST409,
contender GET404 and unchanged original GET200, bounded undo-wins race passes.
Cleanup requested: repost existing cancellation with existing reason (no new
cancellation), then verify removal of15945 occupancy row. Until confirmed,
15945 remains tracked as active. Inverse-winner race not inferred from this run.

CancellationUndoRace cleanup: owner reports successful cleanup after instructions
to repost existing cancellation15945 and remove its Reserved movement. Agent
confirms same-stay availability200/true, original GET200 exact immutable result,
contender GET404; all no-store. Fixture occupancy released; historical docs/keys
retained. Undo-wins bounded race and cleanup complete.

Two-staff room-move setup: owner confirms17436 posting succeeds with Busy203
baseline, February10–12,2027. Attempt to edit same17436 in both sessions hit
native object edit lock (holder session314, МБ Консалтинг) before race dispatch.
This is a setup limitation, not evidence of shared-room-lock failure or completed
ConcurrentRoomMove. Owner closes document windows in both sessions, discarding
unsaved edits; posted baseline retained. Revised setup uses distinct documents.
Agent creates second-session double-twin/2guest fixture HTTP201, key
97806403-2151-4513-96ba-7f85be63065f, reservation815374f0-e5f9-11f1-82c4-e46f13aa4449,
Тест переноса — сотрудник Б, February10–12,2027,4000/night8000total.
Native allocated room/document number and check-in setup pending. Expected206
if203 baseline remains posted; API response does not expose physical room.
New fixture active and requires tracked cleanup after race.

Distinct-document three-way move attempt: A17436 baseline Busy203, B17438
basisBooking15947 baseline Busy206. Requested B edit206→203 withoutsave then
A edit203→204 and pause post-lock in Checkin posting. Owner reports execution
paused; agent directly dispatches API single-balcony201 Feb10–12,2027,
keya73ae6a7-70ca-4004-9e30-1cdf919a5033 (first network call, pending at1s yield).
Owner attempts Bpost then resumesA; B screenshot has unsaved203 and
«Конфликт блокировок при выполнении транзакции», A reports noerrors.
API returns201/no-store, reservation815374f2-e5f9-11f1-82c4-e46f13aa4449,
8000total, elapsed58.931s including debugger pause/user interaction.
Do not label conflict timeout or deadlock without native detail/timing evidence;
long intentional pause may exceed B lock wait. Full expected two-move success
not observed; run remains incomplete pending stored rooms/movements/rollback
inspection, diagnostic evidence and controlled short-pause repeat if appropriate.
Do not retryB or count full ConcurrentRoomMove pass from API201 alone.
Active fixtures: A17436 expected204, B17438 expected206, newAPI201 expected
Reserved (native booking number pending). All require cleanup after inspection.

Owner occupancy screenshot after three-way attempt shows exactly the expected
three synthetic rows: Checkin17436 Busy204; Checkin17438 Busy206; Booking15948
Reserved201; each February10,2027 13:00 through February12 11:59:59. No residual
Busy203 row visible. This establishes consistent occupancy after B lock-conflict
rollback; source of lock error remains undetermined. Settlement counts and
stored B form reopen still not independently supplied. Before controlled shorter-
pause repeat, requested cancelAPI15948 via existing workflow, restoreA17436
203 with all breakpointsdisabled, retainB17438 posted206 and verify charges.
A repeat must use a NEW APIkey, as15948 key replay bypasses allocation by design.

Three-way repeat preflight: owner reports resetready; agent availabilitytrue201.
Owner reports Apaused; fresh direct API POST key4512c192-529c-4747-a9dc-de91e55dd87c,
Тест переноса — короткая пауза returned201/no-store in0.516s, UUID
815374f4-e5f9-11f1-82c4-e46f13aa4449. No observed waiting; intended lock contention
not established. A exact execution arrow/frame/values not yet inspected. Cannot
infer missinglock, wrongbreakpoint, bypass or runtime defect from this alone.
Owner requests reset before supplying debugger evidence. Latest GET200 exact
verified; newAPIfixture active pending staff cancellation. Bpost was explicitly
held; A resume outcome and baseline rooms must be rechecked during reset.
No full ConcurrentRoomMove pass. Next attempt requires pause-location evidence
before dispatch; no new reservation sent during reset.

Reset correction: owner screenshot attemptingA17436→203 is rejected normally:
room203 Busy Feb10,2027 13:00 through Feb12 11:59:59 by Checkin17438.
Thus previous assumption B remained206 is not valid for latest state; exact B
commit timing not supplied. Reset must first reopen/move/postB17438→206 and
verify Busy206, then restoreA17436→203. FailedA form remains unsaved; discard
before reopening. No forced status/movement edits or conflict bypass requested.
New API cancellation/reset completion still requires independent confirmation.

Verified-pause three-way move: owner debugger screenshot shows execution arrow
on Текущее = ПрочитатьПоляДокумента(Снимок.Ключ) AFTER ЗаблокироватьНомера;
Документ.Номер17436, newroom204, basisroom201. Baseline intendedA203/B206.
Agent directly dispatches fresh API keyb0833311-0b45-463d-a562-2f28d7fb873f,
Тест переноса — проверенная остановка. Pending at initial1s yield. Owner posts
B17438→203, disables breakpoints/resumesA, reports A and B each succeeded.
API201/no-store, UUID815374f6-e5f9-11f1-82c4-e46f13aa4449, elapsed57.444s
including intentional pause/interaction (not a performance measure). GET and
identical POST both200 exact/no-store, no duplicate creation. A/B origin keys
recover original stay/price200. Both moves and API report success; final native
Busy204(A)/Busy203(B)/Reserved201(API), zero residualBusy206, charge counts,
and newAPI one-booking/one-counterparty assertions pending. Do not infer stale-
snapshot/reverse-order/allbasis paths or instrumented lock wait location.
Active syntheticfixtures require cleanup after native inspection.

CORRECTION to verified-pause three-way run: owner retracts earlier «A and B each
succeeded» report and supplies screenshot17438 with unsavedroom203 and explicit
«Конфликт блокировок при выполнении транзакции», now with Подробнее button.
A success and API201 remain reported/observed; Bmove FAILED. Any preceding
wording that bothmoves succeeded is superseded. Full ConcurrentRoomMove remains
failed/incomplete pending diagnostic cause, rollback state and retest. No further
blind repeats: request full native error detail/callstack before changes. Do not
infer timeout versus deadlock without details. ActiveAPIfixture retained;
no new request, retry, cleanup or code mutation initiated at correction.

Native full error detail for verified-pause B failure: handlerПередЗаписью;
ОбщийМодуль.ОнлайнБронирование.Модуль(120), methodЗаблокировать,
Блокировка.Заблокировать(); nested«Конфликт блокировок при выполнении транзакции».
Failure is during managed shared-room-lock acquisition; screenshot contains no
explicit timeout/deadlock event or participant chain. Timing of Berror relative
to Aresume unknown. Source uses sorted UUID room locking, rechecks snapshots;
no code change justified from generic detail alone. Next obtain timing and, if
needed, technological logTLOCK/TTIMEOUT/TDEADLOCK to distinguish lock graph/cause.
No retries/cleanup initiated in this diagnostic step.

Owner requests skipping further ConcurrentRoomMove investigation/retests.
Further retries and technological-log installation are deferred. Last verified-
pause run remains incomplete/failed: A reports success, B managed-room-lock
acquisition conflict at commonmoduleline120, API201; approximate5s between Bpost
and Aresume owner-estimated, ordering of error versus resume not established.
No timeout/deadlock rootcause proven; no complete two-staff race pass claimed.
Prepared local citypark-lock-diagnostics-logcfg.xml template is NOT installed.
Continue other acceptance work, retaining unresolved concurrency item as an
explicit completion limitation. Last activefixtures need cleanup:17436,17438,
APIguestТест переноса — проверенная остановка; stored post-error rooms require
refresh, not inference from unsavedforms. No diagnostic config or production edit.

Acceptance scope supersession (owner decision, 2026-11-22): owner explicitly
excludes ConcurrentRoomMove altogether, rather than merely deferring it, citing
the difficulty of manual orchestration. Remove its two-staff/three-way and
reverse/stale-snapshot variants from mandatory task02 acceptance. Earlier entries
calling this an unresolved acceptance blocker are historical and superseded for
scope, not for observed results. Lock conflict remains recorded/unexplained;
scenario is excluded, never marked passed. No further diagnostic installation,
repeat or automatic-test implementation requested. Other named scenarios and
shared-lock implementation stay in scope. Active test fixtures still need cleanup.

Wrong-password publication checks completed with deliberately invalid Basic
password (never stored/logged): POSTavailability, POSTreservations(emptyJSON),
GETreservations/{latestexistingkey} each401. No write dispatched past auth gate.
Valid latest-key GET200/no-store exactly original815374f6 creation result.
Post-check single-balcony201 February10–12 advisory available=False.
No cleanup completion inferred solely from advisory; A/B movement cleanup still
requires owner confirmation. ConcurrentRoomMove excluded by owner scope decision.

Checkout regression: owner opens checkout from17438 (basis15947, room203).
Defaultcurrentdate precedes futureFebruary2027stay; owner sets checkoutdocument
February12,2027 12:00 and quantity2. Screenshot shows correct period but online
price silently replaced by ordinary tariff4500/night9000 rather than saved
check-in4000/8000. Checkout remains unposted. Rootcause local form recalculation
unconditionally callsЦенообразование.ЗаполнитьТаблицуСтоимостиПроживания.
Prepared one-file checkout form BSL patch: online-linked basis uses saved
check-in cost rows (including staff overrides), intersects checkout intervals,
keeps full-source quantities, uses existing tariff helper for partialsegments,
rounds amounts and includesorders; no rate coverage rejects, ordinarypricing
unchanged. No objectmodule/XML/role/API edits. BOM/CRLF and unaffected-procedure
preservation assertions pass; nativecompiler/runtime pending.
Deliverycitypark-hotel-checkout-price-fix.zip includes oneBSLfile+README; overlay
CURRENTfullnativeexport, import/checkchangedmodule/F7/restart, discardoldcheckout
draft and createfresh for native retest. No fullacceptance or deploymentclaim.

Checkout fixv1 owner-applied: DBconfiguration screenshot confirms online branch
and saved-price helper. Native debugger screenshots show beforecondition basis
Checkin17438 and nestedBooking15947; after F10 execution selects ordinary-price
branch and basis/nestedbasis watch values become blank (checkin-reference type
retained). Source ЭтоОнлайнБронирование(Ссылка) accepts argument byreference,
then assigns Ссылка = Ссылка.Основание for check-in. When passed formОснование,
this can mutate/coerce the caller's checkin-only reference instead of resolving
a local variable, explaining falseclassification and lostbasis. Changed ONLY
signature to ЭтоОнлайнБронирование(Знач Ссылка) Экспорт; byte-for-byte assertion
confirms no other commonmodulechanges. Prevents inputmutation on all callsites.
Cumulative deliverycitypark-hotel-checkout-price-fix-v2.zip contains commonmodule
and previouscheckoutformfix, README. SHA2568741f3c061d27aa38184dfed35bd8d1287618a1685f1f1f57673cd810ff46dad.
Bothmodules UTF8BOM/CRLF retained. Native compiler/update and freshdraft basis-
retention/4000×2=8000 retest pending; do not save current mutateddraft. No API
writes or idempotency changes. No passing checkout runtime result claimed.

Checkoutfixv2 native outcome: owner screenshot fresh checkout draft from17438,
room203, February10–12,2027 costrow2×4000=8000 after pass-by-value fix. Topdocdate
stillNovember22 in screenshot; requested change to February12,2027 12:00,
verifybasis17438 and stay2, thenpost. Owner reports posting succeeds. Native
number, finaldocumentdate, Finished occupancy, charge rows and undo not yet
shown. Agent originalbooking15947 key97806403-2151-4513-96ba-7f85be63065f GET200
exact immutablecreationresult/no-store after posting. Draftprice correction and
owner-reported posting pass; no complete checkout/undo acceptance yet.

Checkout17595 native screenshots: document dateFebruary12,2027 12:00,
room203, guestТест переноса — сотрудник Б, costrowFebruary10 13:00→February12
12:00, quantity2×4000=8000; savednumber/no unsavedmarker. Own occupancy tab has
one Finished row, periodFebruary12 12:00, stayFebruary10 13:00→February12
11:59:59. Settlement tab has exactlytwo rows February10 13:00/February11 12:00,
calculationdocumentCheckin; amounts clipped at rightedge (leading4 visible), so
exactper-row amounts not independently legible there. Originalcheckin price and
checkoutcostrow previouslyconfirmed4000/8000. Checkoutundo now requested on
17595, then verify emptycheckoutmovements and restored17438 Busy203/two4000
charges without duplicates. No checkoutundo outcome yet.

Checkoutundo17595 owner confirms all requested assertions pass: undo succeeds,
checkoutoccupancy/settlement lists empty; reopened17438 restoresBusy203,
February10,2027 13:00→February12 11:59:59 and exactlytwo4000charges/8000total.
Agent15947 originalkeyGET200/no-store exactly savedcreationresult afterundo.
Normal checkout/post/pricepreservation and serialundo restoration boundedpass;
concurrent two-staff scenario remains explicitlyexcluded. No conflicting-undo,
partial-rate-boundary or restricted-role paths inferred from this normalundo.
17595 retainedunposted;17438 restoredposted and17436/latestAPI syntheticfixtures
still tracked for cleanup. Nextcleanuprequest unpost17436/17438 and cancelAPI
Тест переноса — проверенная остановка, retaining all historicaldocs/keys.

Owner confirms all cleanup instructions complete:17595 leftunposted,
17436/17438unposted with emptyoccupancy/settlement lists, latestAPIbookingguest
Тест переноса — проверенная остановка canceledreasonTest complete. Agent verifies
February10–12 single-balcony availability200/true, latestkeyGET200/no-store exact
historical815374f6 result. All known syntheticoccupancy released by ownerreport;
historicaldocuments/counterparties/keys retained. Serviceeffective-permission
inspection is next; no booking writes required.

RestrictedRole effective-permission runtime evidence: owner pauses authenticated
ws recoveryGET in ПолучитьРезультат afterservicegate. Табло screenshot confirms
РольДоступна(ОнлайнБронированиеСервис)=true; РольДоступна(ПолныеПрава)=false;
ПривилегированныйРежим()=false; ПравоДоступа(Изменение, settingsregister)=false;
sameforЦеныПроживания=false; ПравоДоступа(Проведение,ОтменаБронирования)=false;
sameforВыселение=false. These are native effective permission checks in actual
HTTPcontext, not merely staticroleXML. No deniedwrites attempted. Settings/price
edit and cancel/checkoutposting denials established for this context. GETstill
paused; requesteddisablebreakpoint/F5 tocomplete. Pending GET outcome retained
in /tmp/citypark-service-permissions-get-outcome.json whenfinished. Staffkey
mutation restrictions and other acceptance checks remain separate.

Servicepermission inspection completed: owner resumes GET after nativeeffective
rights screenshot; HTTP200, exactoriginalresponse, Cache-Control:no-store,
elapsed205.003s includes deliberate debuggerpause, not latencybenchmark.
No datawrites; service-role creation/replay/recovery and deniedsettings/prices/
cancellationposting/checkoutposting now have bounded HTTP/nativeevidence.
Nextcheck ordinaryhotelstaff effectivepermissions for keymutation/settings/API;
fullrights administrator account cannot establish ordinarystaff restrictions.
Do not alterws or remove existingadministrator rights to set up this check.

Ordinarystaff testaccount setup: owner creates development-onlydev_booking_staff
with requested АдминистраторГостиницы + ЗапускТонкогоКлиента roles. Initial
loginfails applicationauthorization because no linkedСправочник.Пользователи
entry; this is separate from platformlogin. Source authorization uses infobase
useridentifier. Instructed administrator UI Пользователиинформационнойбазы→
Сопоставитьсновым, preservingtwo roles. Owner now reports successfullogin.
Effective-role/keywrite checks still pending; no administrator/wsrole edits.
No password collected or stored. Account retained for acceptance/cleanuptracking.

Ordinarystaff runtimepermissions: initialdebuglaunch using1cv8.exe failed
client-kind authorization because account hasthin-client-only role. Corrected
launch to1cv8c.exe, sameisolatedbase/dev_booking_staff/Debugtcp; no extra role.
Owner screenshot pausedinМожноИзменитьЦенуВручную shows hotelroletrue,
FullRightsfalse, servicerolefalse, privilegedmodefalse, keyregisterReadtrue,
keyregisterUpdatefalse, settingsregisterUpdatefalse. Effective native staffrights
allow originlookup but deny keywrites (including recordset deletion) and settings
changes; no datawrites attempted. Togetherwithserverrole gate and explicit
HTTPmethodUsefalse in hotelrole source, confirms intended separation, though
no staffcredential HTTPrequest or deniedwrite attempt was made. Resume/close
withoutchanges requested; finalnativecheck/export and remainingmatrixaudit pending.

Owner reports staffdebugexecutionfinished; no documentchanges requested.
AdditionalKeyPayloadConflict HTTPchecks reuse EXISTING97806403 key, nofreshkeys:
valid changedroomType(double-euro), arrival(plusconsistentnightcount/total),
departure(plusconsistentnightcount/total), guests2→1, guestname, email, phone,
nightlyprice/plausibletotal all409 IDEMPOTENCY_CONFLICT/no-store. Whitespace name,
uppercasedemaildomain, formattedphone and reversedJSONrootfieldorder replay200
exactoriginal815374f0 result/no-store. No creation or guestwrites. Complements
prior malformed/type/priceoverflow/oversized-body400 checks.

Native staff-override/partial-checkout regression screenshot: from17438 whose
savedrate ownerinstructed4250, fresh unsavedcheckout has docdateFebruary11,2027
12:00; costrowFebruary10 13:00→February11 12:00, quantity1, price4250, amount4250,
total4250, room203/guestТест переноса — сотрудник Б. Confirms onlinecheckout
uses staffsavedrate and existingtariffhelper for shorterstay, avoiding ordinary
4500tariff. Draftunposted/nounsavednumber; no charges or occupancy writes claimed.
Requesteddiscarddraft, restore17438 savedrate4000/total8000 whileunposted,
then finalDesignerchecklog/currentfullnativeexport for reconciliation. Cleanup
restoration and finalartifacts stillpending. No automatic taskcompletion.

## Final owner source export received (2026-11-22)

Received `htl.zip` and check screenshots 18:28–18:29. All 577 BSL files match
byte for byte; all 1,630 compared XML UUID inventories match. Nine XML/index
serialization differences were reviewed, including unlimited native JSON/settings
strings versus authored bounds. No configuration source was overwritten.
The full check still shows baseline issues; no new booking diagnostic is visible,
but no fresh text log was supplied. Cleanup price restoration is not established
by a configuration export. See [detailed reconciliation](BOOKING_EXPORT_RECONCILIATION.md).
Task 02 remains unchecked pending the remaining required coverage audit.

## Remaining boundary and future-slice checks (2026-11-22)

Development HTTP requests, service-only ws, no production access. Current Moscow
date November22: yesterday/today rejected400; tomorrow accepted200/availabletrue.
Leap2028 February28–March1 accepted200. Storage upper boundary3999 December30–31
accepted200/availabletrue. Year0, five-digit year10000 and invalid2027 leap day
rejected400. Years4000 and9999 incorrectly returned503 BOOKING_UNAVAILABLE.
Primary 1C documentation distinguishes in-memory dates up to9999 from database
storage through3999: https://its.1c.ru/db/content/metod8dev/src/admins/mssqlserver/i8104055.htm .
Fixed common ПрочитатьДату to reject years outside1–3999 before inventory queries;
BOM/CRLF preserved. Patch citypark-hotel-date-storage-boundary-fix.zip SHA256
`1b2714d8c93f3f5c8ce46bb75e0dc765b7d4eff1884185ebf52b8624b983091a`. Owner import/Designer/native HTTP recheck pending; do not record a runtime pass
for this fix. Contract documents the technical storage bound, not a business horizon.
No actual Moscow-midnight clock transition was exercised.

Reused existing97806403 key for validation probes: min1/max999999999999999kopecks
passed price validation and reached409 IDEMPOTENCY_CONFLICT; zero/overflow400.
These probes created nothing. Then three new isolated synthetic bookings were
created for native representable-money and future-slice inspection:

| Guest | Stay | Nightly kopecks | Recovery key | Reservation UUID |
| --- | --- | --- | --- | --- |
| Тест среза будущих броней — 1 | 2027-04-10 → 2027-04-11 | 1 | `0f6b37bb-2543-44c1-aa4c-0254471f6945` | `0d4c2a74-e68c-11f1-82c4-e46f13aa4449` |
| Тест среза будущих броней — 2 | 2027-04-13 → 2027-04-14 | 400000 | `0e764d1b-67a9-4315-b2af-0a9a86a4c139` | `0d4c2a76-e68c-11f1-82c4-e46f13aa4449` |
| Тест среза будущих броней — 3 | 2027-04-16 → 2027-04-17 | 999999999999999 | `e856a25c-cc6d-4064-ba6e-f2c385a2a857` | `0d4c2a78-e68c-11f1-82c4-e46f13aa4449` |

All three POSTs201; each GET and same-key replay200 returned the exact creation
result and price. All responsesno-store. Single-balcony has exactly one eligible
room201. After posting all three future stays, availabilityfalse for each of the
first/middle/last stays, true for both gaps April11–13 and14–16, false for spanning
April9–18. This establishes HTTP later-future slice behavior; native rows/cost values
still require owner inspection. In particular API echo alone does not establish
that the document preserved the maximum amount without coercion.
All three bookings remain posted for inspection. Owner cancellation/cleanup is
pending; preserve documents/counterparties/key history. Local credential-free
artifacts: /tmp/citypark-future-slice-fixtures.json, future-slice-outcomes.json and
citypark-remaining-boundaries-results.json. No claim of native maximum-price or
cleanup completion yet. Task02 remains unchecked.

## Date-storage validation fix: native HTTP recheck passed (2026-11-22)

Owner reports date-storage patch applied. Development HTTP rerun: all14 bounded
cases pass. Years4000 and9999 now400 INVALID_REQUEST instead of503; year0,
five-digit10000 and invalid2027 leap day400. Yesterday/today400; tomorrow200;
valid2028 leap stay200;3999 December30–31 still200/availabletrue. Min1/max
999999999999999kopecks reach expected409 on changed-payload existing-key probes;
zero and overflow400. Every response Cache-Control:no-store. No new reservations
created by this rerun. Results /tmp/citypark-remaining-boundaries-results.json.

For each of the three retained future-slice fixtures, GET and same-key POST
return200 and exactly the original successful result/price after patch application
(six checks, allno-store). Results /tmp/citypark-post-date-fix-recovery-results.json.
Native document cost values, Reserved rows and owner cancellation of those three
fixtures remain pending. Real Moscow-midnight transition not exercised. Full
Designer baseline failures and task02 remaining acceptance limits still apply.

## Future-slice/money native inspection and cleanup confirmed (2026-11-22)

Owner confirms all requested values match and all three fixtures were canceled.
This covers native quantity1, room201 Reserved movements and cost/total0.01,
4000.00,9999999999999.99 RUB for April10–11,13–14,16–17 respectively, followed
by removal of their Reserved movements. This is owner-reported native inspection,
not additional screenshots or a direct database query.

Agent independently verifies all three stays now availability200/true; GET and
same-key replay each200 exact original result/price after cancellation (nine
checks, allno-store). Same-key replay did not recreate occupancy. No fresh keys
or bookings created. Results /tmp/citypark-future-slice-cleanup-results.json.
Representable-price native round trip and later-future slice check now have both
HTTP and owner-native evidence. All three fixtures cleaned up; history retained.
Moscow-midnight boundary remains unexercised; full Designer baseline failures
remain documented, not a clean check. Task02 remains unchecked/uncommitted.

## Native Moscow-midnight expression check passed (2026-11-22)

Owner screenshot19:11 shows actual 1C Табло evaluation: arrival2026-11-23
compared with НачалоДня(UTC+10800) yields Ложь atUTC2026-11-22 20:59:59,
Истина at21:00:00. Tomorrow becomes today exactly at Moscow midnight, matching
the rejection predicate used by ПроверитьБудущуюДату. This is native boundary
calculation evidence with explicit input dates, not a live HTTP request spanning
midnight or an OS-clock change. Separate live today/tomorrow requests passed.
Requested remove breakpoint/F5/close withoutsaving; resume remains owner action.

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

## Debugging finished and final review recorded (2026-11-22)

Owner confirms debugging finished. Final scoped source/metadata/permission,
encoding, color-only compatibility and credential/input-exclusion checks complete.
See [acceptance review](BOOKING_ACCEPTANCE_REVIEW.md) for exact results and
proposed baseline exception. Task02 remains unchecked/uncommitted; no production
write or task03 work. Owner acceptance of existing Designer failures pending.

## Task02 completion approved (2026-11-22)

Owner confirms debugging finished and explicitly accepts the Designer baseline
exception: “I accept the baseline exception.” Final scoped review completed;
structural checks pass; bounded runtime evidence/limits and ConcurrentRoomMove
exclusion retained. Task02 marked complete for the reviewed implementation.
Full check still fails on the documented production baseline; no deployment,
production write, task03 work or broader test waiver is implied.

## 1. Locate the platform and open the isolated base

Run this in Windows PowerShell; it only lists installed executable paths:

```powershell
Get-ChildItem "${env:ProgramFiles(x86)}\1cv8\*\bin\1cv8.exe", "${env:ProgramFiles}\1cv8\*\bin\1cv8.exe" -ErrorAction SilentlyContinue |
    Select-Object FullName, @{Name='Version'; Expression={$_.VersionInfo.ProductVersion}}
```

Select the matching installed build, then open Designer. Use the **isolated base
path from `1c.txt`** when prompted. Do not select the working production base.
Authentication stays in the 1C login dialog, outside command history.

```powershell
$bookingV8 = Read-Host 'Full path to the selected bin\1cv8.exe'
$bookingDevBase = Read-Host 'Isolated file base path from 1c.txt'
& $bookingV8 DESIGNER /F $bookingDevBase
```

In Designer:

1. Confirm the base identity, platform build, configuration revision and development
   publication. Ensure the copy's scheduled tasks, notifications and external
   integrations are isolated. Keep a restorable backup of this development base.
2. Copy/pull this working tree onto Windows. The source directory is
   `citypark-hotel-managment`, containing `Configuration.xml` directly (there is
   no `booking/` wrapper in the current checkout).
3. Load the configuration **from files** from that directory. A full load replaces
   the development configuration, so verify the selected base before confirming
   the Designer operation. Resolve/report import or compilation errors first.
4. Run the configuration check, including server modules and managed client forms.
   Save the full errors/warnings log; redact credentials and real guest data.
5. Update the **development** database configuration. Start Enterprise and check an
   ordinary staff booking, check-in, cancellation and checkout before activation.
6. Export the accepted configuration to a separate directory and compare new
   metadata/role UUIDs and module syntax. Let the platform regenerate version
   hashes in `ConfigDumpInfo.xml`; locally added entries register identities without
   pretending to know the platform's version hashes. Do not use incremental import
   as a substitute for this first full import/check.

The locking protocol runs in object `ПередЗаписью`/`ПередУдалением` and posting/
undo handlers, covering both interactive and programmatic document writes. The
platform's documented object event sequence places object `BeforeWrite` in the
write transaction: [1C applied-object event sequence](https://kb.1ci.com/1C_Enterprise_Platform/FAQ/Development/Standards/Applied_objects/?language=en).
Runtime races on the installed build remain mandatory.

## 2. Configure and publish in development

The service starts unavailable because `НастройкиОнлайнБронирования` has no enabled
rows. Under a full-rights **development administrator**, open the register through
All functions and create one record per supported website type:

| Field | Value/check |
| --- | --- |
| `ТипНомераСайта` | One of the five IDs in BOOKING_CONTRACT.md |
| `Включено` | Keep false until references, permissions and configured check-in/out constants are checked |
| `Организация` | Installer-verified booking organization; do not infer it from catalog row count |
| `Ответственный` | Catalog user corresponding to the dedicated authenticated HTTP user |
| `ТипНомера` | Verified 1C room type from the contract |
| `НомераJSON` | JSON array containing exactly that website type's physical-room UUID set |

Example `single-balcony` room array:

```json
["fb170ab4-5e75-11e4-80cd-b8a38697f1fc"]
```

Copy the other four sets from [the frozen mapping](BOOKING_CONTRACT.md#stable-mapping-required).
Runtime validation rejects cross-type substitutions even where 1C room types are
shared, duplicate IDs, absent/deleted rooms or deleted room types/buildings/complexes.
Verify each room's building and its pricing-type reference, the predefined `Сутки`
tariff, and the actual arrival/departure constants. Match the isolated fixture
values to production: owner reports check-in 13:00; verify production check-out
rather than assuming noon. Do not change production constants for the service.
The online amount comes from the request's checked backend snapshot, not the tariff price.

Create/map the dedicated infobase user and its `Справочник.Пользователи` record.
Assign `ОнлайнБронированиеСервис` without full rights or hotel administrator rights.
The role includes the runtime-required `Get` right for session parameter
`АвторизованныйПользователь`, needed by standard booking document filling;
restricted creation/replay/recovery and Reports denial are verified above.
If the BSP session initialization requires additional access, report the exact
failing operation and add only the required access; do not work around it by
making the service user an administrator. The service role has no price, room,
settings, cancellation or checkout editing access. The common module is server
only and is not exposed through direct client server calls. Ordinary hotel staff
may read online origin for price preservation but cannot alter the idempotency
register, activate settings or invoke the booking HTTP methods through their role.

Publish the new **Booking** HTTP service in the existing development publication,
with Basic authentication and anonymous access disabled. Verify the publication's
actual root; expected route family:

```text
http://mbc.cityparkvip.ru:8081/CityParkHotelDev/ru_RU/hs/booking/v1
```

This owner-confirmed development URL is not a production transport configuration.
After setup, enable one type and run availability first, then creation checks.

## 3. HTTP checks (Python 3, no extra packages)

Use the copied repository's hotel project directory as the current directory.
The checker prompts for the dedicated development service credentials and never
prints them. Use future dates with verified free inventory. The dates below are
examples; update them if they are no longer in the future in Moscow.

```powershell
py -3 scripts/booking-http-check.py `
    --base-url 'http://mbc.cityparkvip.ru:8081/CityParkHotelDev/ru_RU/hs/booking/v1' `
    --check-in 2026-12-01 --check-out 2026-12-03
```

Default checks are anonymous denial, advisory availability, unknown-key recovery
and unknown-field rejection. **No reservation creation is sent by default.**
Once these pass, creation is explicitly opt-in for the isolated development base:

```powershell
py -3 scripts/booking-http-check.py `
    --base-url 'http://mbc.cityparkvip.ru:8081/CityParkHotelDev/ru_RU/hs/booking/v1' `
    --check-in 2026-12-01 --check-out 2026-12-03 `
    --create --isolated-dev --nightly-price-kopecks 400000
```

This intentionally creates **one synthetic development reservation**, using
`booking-check@example.invalid` and a synthetic phone. It checks 201, normalized
same-key replay with different JSON field order, GET recovery and changed-payload
409. The recovery key is printed **before** dispatch. If a transport failure occurs,
retain that key and use GET recovery before attempting another new key. No email
or payment is sent by this 1C service.

Do not infer correctness merely from an HTTP success. Inspect the document,
counterparty, occupancy and saved key/result in Enterprise. Cancel the test booking
through the staff workflow after inspection. Idempotency records are immutable,
including after cancellation; reset the isolated fixture base from its backup when
necessary instead of deleting recovery history in an active base.

## 4. Required runtime matrix

The matrix below lists required assertions. Partial HTTP and owner-native results
are recorded above; full scenario acceptance is still incomplete. Do not treat
individual passing subchecks as completion of every assertion in a named scenario. Record
the actual build, fixture document/key IDs,
observed register rows, concurrent outcomes and cleanup for each result.

| Named scenario | Setup and required assertion |
| --- | --- |
| `BookingValid` | Free one-room stay; 201 only after committed open/posted `Бронирование`, one new counterparty, exact guests/configured check-in and check-out timestamps, `Проживание` nights, backend `Стоимость`/`Сумма`, reserved movements and one immutable key/result/contact record. No contacts in response or public comment. |
| `FullStayOneRoom` | Fill alternating parts of different eligible rooms so each night has capacity but none spans the full interval; reject creation with 409 and no new document/counterparty/key. Also create several future stays on one room and inspect slice behavior. |
| `AdjacentStays` | Use configured times: checkout 12:00 permits next check-in 13:00 on the same date, and equal boundaries are also allowed. If checkout is later than the next check-in, reject the actual overlap even on adjacent calendar stays. Partial/identical overlaps still fail; no extra occupied second. |
| `OverlapVariants` | Partial overlaps from both sides, contained/containing and identical stays fail. Test capacity limits 1/2/4, today/past dates, tomorrow across Moscow midnight, invalid/leap dates, departure <= arrival, stays >90 nights and dates >6 months ahead, representable type limits and price overflow. No artificial duration/horizon cap. |
| `WebsiteStaffRace` | One eligible free room, overlapping HTTP creation and staff booking/check-in in two sessions; exactly one commits. Inspect occupancy and all documents after both complete. |
| `ConcurrentRoomMove` — excluded | Removed from required task 02 acceptance by explicit owner decision (2026-11-22). Prior lock-conflict attempts are retained as history, not passes. No further manual repeat/logging required; shared-lock implementation and other named scenarios remain unchanged. |
| `DuplicateKeyRace` | Two simultaneous identical HTTP requests, same key; one document/counterparty/key only, responses 201/200 with same UUID. |
| `KeyPayloadConflict` | Reuse key with each changed stay/contact/price field; 409, no writes. Whitespace/domain/phone normalization and JSON order replay return original result. Reject unknown fields, wrong JSON types, oversized/invalid JSON and noncanonical keys. |
| `PostingRollback` | Designer breakpoint/test-only failure during posting inside the create transaction; rollback leaves no new counterparty, reservation, occupancy or successful key. Remove injection before other tests. |
| `CommitTimeoutRecovery` | Drop first response after transaction commit; GET/same-key POST recover exact saved result and create nothing else. GET while original request is unresolved must never imply a new key is safe. |
| `CancellationUndoRace` | Cancel the website reservation, race another allocation and cancellation undo. Release/restore movements atomically; undo must fail if the room is now occupied. GET recovery still returns the immutable creation result after cancellation. |
| `RepostSelfExclusion` | Repost/edit a reservation and check-in; exclude the current document and its booking basis, but reject other conflicts. Unpost/repost and checkout undo retain consistent movements. |
| `BackendPriceAtCheckIn` | Use a backend snapshot differing from 1C's tariff. Open booking form, trigger automatic recalculation, create/open staff check-in and trigger recalculation. Cost rows, total and guest count survive; ordinary staff-created reservations retain normal pricing. Explicit staff cost edits remain possible. |
| `UnauthorizedAccess` | Missing/wrong credentials rejected by publication; unrelated authenticated role denied every HTTP method. No contact/body/credential details in errors. Verify private responses use no-store. |
| `RestrictedRole` | Run using only the dedicated role plus strictly necessary platform/session access. Quote/create/recover succeed; cannot edit settings/room prices, invoke reports or cancel/check out through the service role. Hotel staff cannot rewrite/delete key records or invoke HTTP methods. Record exact additional rights, if any, before changing them. |

## Recording a result

For each scenario append: date/time, platform build, configuration revision,
fixture IDs, steps, observed document/register state, result and cleanup.
Any failed required Designer, lock, availability-slice or pricing assertion blocks task 02. The explicitly excluded `ConcurrentRoomMove` scenario is not a completion gate.
Do not replace a failed shared lock with a handler-only lock or skip staff paths.
No production booking writes, deployment or task 03 work is included here.
