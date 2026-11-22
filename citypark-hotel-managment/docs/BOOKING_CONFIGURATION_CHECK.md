# Owner configuration check: 2026-11-17

Status: failed; task 02 remains unchecked and uncommitted. The owner ran Designer
configuration checking after importing archive v3 and supplied `config-check.txt`.
The file is valid UTF-8 with readable Cyrillic; the question marks in the earlier
chat paste are not the contents of the supplied file. The local log remains
untracked user input.

## Production reference received and compared

The owner supplied `hotel-orig/`. Comparison confirms the ten missing common
modules are also absent there and all 23 failing BSL source lines are identical.
The two metadata-diagnostic form XML files also match byte-for-byte. This confirms
existing source defects; restoring supposedly omitted modules from this export
is not possible. The latest check still has no booking compiler diagnostics, but
the full check remains failing. Continue isolated development database/runtime
validation while retaining these completion limitations. No check waiver or task
completion is implied. See [the production baseline comparison](BOOKING_PRODUCTION_BASELINE.md).

## Latest owner check: booking compiler diagnostics cleared

Reread the newest owner log: 23,935 bytes, UTF-8, SHA-256
`9e73b2f6577091dfbe2c4f59eea308e8b392931c96170e02004fd12c05d34bc9`.
The four `СтрРазделить` errors and earlier date-constructor error are absent.
There are **no compiler diagnostics for the new booking module in this run**.
This is owner-run compilation evidence for those corrections, not a full check or
runtime pass.

All **23 remaining compiler errors** are the previously identified missing BSP
common-module dependencies in the two existing administration processors. The
51 no-reference diagnostics, incorrect advance-report data path and unresolved
SORC import-form metadata reference also remain. Full configuration checking is
still failing. No source code was changed merely to silence these dependencies.

The next blocker is the genuine dependency baseline: a complete XML export or
original `.cf` from the working configuration is needed to compare the intended
BSP subsystem composition and identify the missing modules. Do not infer that
these modules are absent from every working deployment solely from their absence
in this repository. Task 02 is not complete, no commit is made, and database update,
role checks, concurrency, posting and all runtime acceptance remain unrun.

## Updated owner check: string-split compatibility failure

Reread the overwritten owner log (25,263 bytes, UTF-8, SHA-256
`8ab24eddc9f5201ffe09228f72a6271ab74b12ecda56be4f9a7dc189834d6ed7`).
It now contains **27 compiler errors**: four `СтрРазделить` failures in the booking
module (lines 252, 379, 388, 470), plus the same 23 existing BSP missing-module
errors. The earlier date-constructor diagnostic is absent, providing owner-run
evidence that the date correction cleared that diagnostic; the booking module
still has not passed checking.

The installed platform is reported as 8.3.11.3133, while the configuration's
`CompatibilityMode` remains `Version8_3_4`. `СтрРазделить` was introduced with
8.3.6, whose release notes explicitly retain the previous behavior in 8.3.5
compatibility mode: [official 1C release notes](https://dl03.1c.ru/content/Platform/8_3_6_2041/1cv8upd.htm).
The observed function-availability errors are consistent with the older
compatibility setting; they do not mean all 8.3.11 installations lack the function.

Replaced all four calls with the existing exported, server-available
`СтроковыеФункцииКлиентСервер.РазложитьСтрокуВМассивПодстрок` helper. Its third
argument is `ПропускатьПустыеСтроки`, so all calls explicitly pass **Ложь** to
preserve empty fields (equivalent to inclusion of empty fields in the original
calls). This preserves leading/trailing/consecutive delimiter handling needed by
email validation and leaves name/phone/price behavior unchanged. The existing
helper code is byte-for-byte unchanged from HEAD; no new library is introduced.
Compatibility mode, XML metadata and UUIDs remain unchanged.

`citypark-hotel-booking-compat-fix.zip` contains the corrected booking module,
including the earlier six-argument date constructor. Apply it inside the v3 source
folder, replacing the single module, then reimport/recheck in the isolated base.
This newer patch supersedes the date-only patch. Static checks confirm four
replacements, absence of the unsupported calls from the new booking modules,
preservation of BOM/CRLF and date fix, and successful layout/color scans. Native
Designer recheck and runtime acceptance are still pending; the existing 23 BSP
errors are not cleared by this module correction.

## Booking error and source correction

One compiler error is in `CommonModules/ОнлайнБронирование/Ext/Module.bsl:348`:
`Ожидается символ ','` at the end of the four-argument date constructor.

Corrected the noon constructor to supply the complete time:

```bsl
Результат = Дата(Число(Лев(Значение, 4)), Число(Сред(Значение, 6, 2)), Число(Прав(Значение, 2)), 12, 0, 0);
```

This is the only `Дата(...)` call in the new service/common/idempotency modules.
The source correction preserves all other bytes, including UTF-8 BOM and CRLF.
It has not yet been rechecked in Designer; later errors can be exposed when this
syntax error is cleared. Local layout/color scans still pass all 766 root objects.

The small `citypark-hotel-booking-date-fix.zip` patch contains only
`CommonModules/ОнлайнБронирование/Ext/Module.bsl`. Extract it **inside** the existing
v3 `citypark-hotel-managment` source directory, replacing that file, then reimport
into the isolated base and rerun checking. Alternatively, apply the exact line
above directly in Designer's `ОнлайнБронирование` module for the immediate check;
keep the source file synchronized before the next import. Earlier v3 archives
retain the bad constructor and require this patch.

## Existing compiler errors: missing BSP modules

The other **23 compiler errors** are undefined variables naming ten missing common
modules. All reported BSL files are byte-for-byte identical to HEAD. All ten
module names are absent from both HEAD's `Configuration.xml` declarations and the
current `CommonModules` directory; a hidden-file search did not find misplaced
copies. These are existing unresolved dependencies, not a passing baseline.

| Missing common module | Compiler errors |
| --- | ---: |
| `ВариантыОтчетов` | 10 |
| `ОбменДаннымиСервер` | 3 |
| `ФайловыеФункции` | 2 |
| `ПолнотекстовыйПоискСервер` | 2 |
| `РезервноеКопированиеОбластейДанных` | 1 |
| `РезервноеКопированиеИБСервер` | 1 |
| `ОбновлениеКонфигурации` | 1 |
| `РезервноеКопированиеИБКлиент` | 1 |
| `УправлениеДоступомСлужебный` | 1 |
| `АвтономнаяРаботаСлужебный` | 1 |

They occur in forms of `ПанельАдминистрированияБСП` and
`ПанельАдминистрированияБСПВМоделиСервиса`, in server/thin-client contexts. The
color compatibility edits changed XML color values in some of these forms, but
none of their BSL code or missing-module declarations. Do not replace these
modules with dummy implementations or delete their calls merely to pass checking.
A genuine working configuration export/library baseline is needed to restore the
intended dependencies or establish the correct supported subsystem composition.
No database configuration update is claimed or requested while these errors remain.

## Other existing diagnostics

- **51** `Не обнаружено ссылок` diagnostics for functions/procedures, including
  plugin/contact/print handlers. Some may be attached dynamically; do not remove
  them without checking their runtime registrations.
- `Документ.АвансовыйОтчет.Форма.ФормаДокумента`: one incorrect data path.
- `Обработка.ЗагрузкаНоменклатурыИзКонфигурацииSORCКафе.Форма.Форма`: one unresolved
  metadata reference.

The two metadata-diagnostic form XML files are byte-for-byte identical to HEAD.
The log gives counts, not the exact bad paths/reference targets, so these two
items require expanded Designer diagnostics before a concrete repair. They are
recorded as unresolved and are not being classified as harmless.

## Next evidence required

Recheck the corrected booking module and full configuration in Designer; retain a
new readable UTF-8 log to identify any further booking errors. The owner has now supplied the production XML reference; it has the same
missing BSP dependencies. Their intended subsystem composition/remediation remains
unresolved; no substitute library code has been introduced. Preserve the
isolated base and original configuration for comparison. Import/compilation,
database update and booking/runtime scenarios remain separate gates.
