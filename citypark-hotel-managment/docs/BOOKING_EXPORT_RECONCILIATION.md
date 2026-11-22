# Development export reconciliation: 2026-11-22

Owner supplied `htl.zip` with Designer check screenshots at 18:28–18:29.
The archive is configuration source, not an information-base dump. It contains
2,584 files under `htl/`; CRC validation succeeds. SHA-256:
`586639bd757ecf081990b8fb4e4bce3757b6054194220f558199d5bf796daf0c`.
The input archive remains untracked and was not copied over the working tree.

## Source comparison

- All **577 BSL files match byte for byte**, including the manual price command,
  checkout saved-price helper and `ЭтоОнлайнБронирование(Знач Ссылка)` fix.
- Every archive file exists in the working configuration. Nine files differ;
  the other 2,575 match byte for byte.
- UUID inventories match for all **1,630 XML files** other than the generated
  `ConfigDumpInfo.xml` index. `Configuration.xml` itself matches byte for byte.
- Both modified booking/check-in form XML files match byte for byte.

The nine differences were inspected without applying them:

| File/group | Difference and disposition |
| --- | --- |
| `InformationRegisters/КлючиОнлайнБронирования.xml` | Native export has string length 0 (unlimited) for the two JSON resources instead of authored 16,384. Runtime replay/lookup exercised this native schema. Preserve authored bounds in the repository; record the round-trip difference explicitly. This is not byte-identical metadata. |
| `InformationRegisters/НастройкиОнлайнБронирования.xml` | Native export has length 0 instead of authored 1,024 for the settings string. Same disposition; no settings values were imported or changed. |
| `Roles/АдминистраторГостиницы/Ext/Rights.xml` | Object blocks reordered; explicit true Read entries for keys/room-lock registers omitted under `setForNewObjects=true`; additional history rights explicitly false. Existing false Update/Edit/settings/HTTP-method rights retained. Native staff debugger separately confirmed key Read=true, Update=false and settings Update=false. No new explicit allow was found. Keep authored explicit permissions. |
| Five existing form XML files | Native export omits AutoFont properties in the event-log filter/event forms and AdditionalColumns entries in the external-user/user-settings forms. These forms are outside the booking change; do not import this serialization churn. |
| `ConfigDumpInfo.xml` | 2,033 common entries have regenerated configVersion values. Two Reports booking_references index entries added; 19 help/predefined/subsystem index entries omitted. No corresponding configuration file is missing. Do not replace the repository index merely to synchronize generated hashes. |

The five forms are `Catalogs/ВнешниеПользователи/Forms/ФормаЭлемента`,
`DataProcessors/ЖурналРегистрации/Forms/{ОтборЖурналаРегистрации,ФормаСобытия}`,
and `DataProcessors/НастройкиПользователей/Forms/{ВыборНастроек,НастройкиПользователей}`
(each `Ext/Form.xml`). No original UUID or form control ID was deliberately changed.

## Designer evidence and completion limits

The new screenshots show the known unresolved-reference diagnostics, advance-report
form path issue, SORC import-form reference issue, and missing BSP dependencies in
existing administration forms. No booking compiler diagnostic is visible.
They are consistent with the previously supplied text log and production-source
baseline; screenshots do not establish an exact machine-readable diagnostic diff
or a clean full check. The existing local `config-check.txt` remains the earlier
23,935-byte log, not a newly supplied final text log.

Full configuration checking still fails. Do not stub BSP modules or repair unrelated
forms to make this task appear clean. See [baseline evidence](BOOKING_PRODUCTION_BASELINE.md)
and [configuration check history](BOOKING_CONFIGURATION_CHECK.md).

The latest partial checkout draft confirms one night at the saved staff override
of 4,250, total 4,250. It was not posted. Restoration of check-in 17438 to saved
4,000/8,000 and dismissal of that draft were requested; this source archive cannot
verify those information-base values.

Task 02 remains unchecked and uncommitted. Required matrix assertions still need
a final coverage audit, especially several future stays on one room/slice behavior,
Moscow-midnight boundary and representable input limits. Do not mark those passed
from adjacent or malformed-input tests. `ConcurrentRoomMove` remains explicitly
excluded by the owner; no further manual repeats are required. No production
writes, task 03 work or deployment occurred.
