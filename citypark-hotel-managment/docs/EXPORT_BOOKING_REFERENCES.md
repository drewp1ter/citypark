# Export booking references

The script [export-booking-references.bsl](../scripts/export-booking-references.bsl)
is a server-side BSL code fragment, not a Windows/PowerShell script or a standalone
`.epf` file. It uses the catalog fields in this repository and the JSON writer
already used by the Reports HTTP service. Target platform: owner-reported
8.3.11.3133. It has **not** been compiled or run in 1C here.

## Run

1. Open the intended hotel information base in 1C:Enterprise using an account
   permitted to read the hotel room and room-type
   catalogs. The script does not enable privileged mode or change access rights.
2. In an existing code console, select server-side execution and paste the entire
   script. No code console is bundled with this repository. If none is installed,
   a 1C developer can put this fragment inside a temporary external processing
   form's server procedure and call it from a button. No configuration import,
   database update, HTTP service modification or document posting is required.
3. Execute once. Copy the JSON message. If the console truncates messages, copy
   the complete `РезультатJSON` variable through its variable inspector instead.
   Ensure the result contains the `rooms` array and closes
   with `}`. Save it as UTF-8 JSON outside version control if needed.
4. Share the JSON.
   If multiple rooms share a number, identify the intended room by UUID.
   Do not choose the first match automatically.

## Result and verification

The export includes all rooms (the room catalog is non-hierarchical), including deletion-marked items, so
different code/name/printed-number conventions do not silently hide the target
rooms. Each room includes its UUID, code, name, printed number, room-type UUID,
room-type name and deletion marks. Missing type references produce an empty type
ID. Organization, building and complex data are excluded; no owner dereferences
are performed.

Match these 12 room numbers against code, name and printed number:

| Website ID | Expected rooms |
| --- | --- |
| `single-balcony` | 201 |
| `single` | 204, 304 |
| `double-twin` | 203, 206, 302, 303 |
| `double-euro` | 202, 205, 305, 306 |
| `vip` | 301 |

Confirm each intended room has a unique nonempty UUID and the expected type.
Deletion-marked rooms/types must not be enabled for
booking. Unexpected room counts, duplicate numbers or empty references need review.
The booking organization must be resolved separately in the integration configuration;
this export does not infer it from building ownership.

The script only performs one catalog SELECT query and serializes the selected
fields. It reads no guest/contact records, reservations, credentials, building
connection settings or occupancy history; it writes no database objects or files.
Restricted access may raise an error; do not interpret a partial or failed export
as an authoritative inventory list. Runtime acceptance remains pending execution
in 1C; repository metadata inspection is not a platform compilation check.

## HTTP export (preferred)

The existing Reports service now defines:

`GET <publication URL>/hs/reports/v1/booking-references`

It uses the publication's existing username/password authentication and the
caller's catalog read permissions. No privileged mode is enabled. Ensure anonymous
access to this service is disabled in the publication; this is enforced by the
publication, not a new password check inside the handler.

After importing the changed Reports metadata/module and checking/updating the
configuration in Designer, call it from an authorized workstation:

```powershell
curl.exe --user "YOUR_1C_USERNAME" --output booking-references.json "https://YOUR_HOST/YOUR_PUBLICATION/hs/reports/v1/booking-references"
```

curl prompts for the password; do not put credentials in the URL. Replace the URL
with your actual HTTPS publication URL. Configuration import/update is performed
by the owner; no deployment has been performed by the agent.

Success: HTTP 200, `application/json; charset=utf-8`, `Cache-Control: no-store`.
The body is the same `schemaVersion: 1`, `rooms` export described
above. Empty catalogs produce empty arrays. Catalog/query/serialization failures
return HTTP 500 with the same JSON/no-store headers and this body:

```json
{"statusCode":500,"message":"Не удалось получить справочные данные бронирования"}
```

Authentication failures are handled by the publication; their body is not part of
this handler's JSON contract. No guest records or secrets are included.

### Runtime acceptance (pending)

On 8.3.11.3133, check the configuration and server BSL in Designer. With authorized
credentials, verify valid complete JSON and the 12 intended room mappings. With no
or incorrect credentials, verify access is rejected. Test a user lacking catalog
permissions: no partial inventory should be returned. Verify existing sales,
room-sales and month-sales routes still behave as before. Confirm no catalog or
document records are changed. These checks have not been run in this environment.
