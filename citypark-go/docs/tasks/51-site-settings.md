# Task 51: Site settings

## Goal
The site's contact details (phones, address, e-mail, company details, social links) are one settings object in the
database. The site reads it from `GET /settings`, and an admin replaces it with `PUT /settings`.

## Depends on
31.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/hotel/store.go` (single-row table pattern),
`../citypark-modern/src/shared/config/site.ts` (the seed values; only the fields listed below move).

## Background
Decided by the owner (`../docs/adr/0001-admin-panel.md`). Navigation, legal links, the site name, the chef's name and
the copyright year stay in the site's code. Phones are stored as a display text plus a number for the `tel:` link.

## Files to create
- `migrations/00008_site_settings.sql`, `internal/settings/store.go`, `internal/settings/handler.go`,
  `internal/settings/settings_test.go`

## Files to change
- `internal/app/app.go`: `settings.NewHandler(settings.NewStore(db), tokens).Routes(r)`.
- `docs/API.md` (new area `### Settings`), `internal/apidocs/openapi.yaml`.

## Spec

### Data
```go
type Phone struct {
	Label      string `json:"label" validate:"required,max=60"`
	ShortLabel string `json:"shortLabel" validate:"required,max=30"`
	Display    string `json:"display" validate:"required,max=40"`
	Number     string `json:"number" validate:"required,max=20"` // "+79183119791" or "88001002441"
	AltDisplay string `json:"altDisplay" validate:"max=40"`       // second (landline) number, optional
	AltNumber  string `json:"altNumber" validate:"max=20"`
}
type Social struct {
	Name    string `json:"name" validate:"required,max=40"`    // "ВКонтакте"
	Label   string `json:"label" validate:"required,max=10"`   // "VK"
	URL     string `json:"url" validate:"required,max=200"`    // https only
	Display string `json:"display" validate:"required,max=60"` // "vk.com/cityparkvip"
}
type Settings struct {
	Phones struct {
		Restaurant Phone `json:"restaurant"`
		Hotel      Phone `json:"hotel"`
		Free       Phone `json:"free"`
		Landline   Phone `json:"landline"`
	} `json:"phones"`
	Address struct {
		Postcode string `json:"postcode" validate:"required,len=6,numeric"`
		City     string `json:"city" validate:"required,max=60"`
		Street   string `json:"street" validate:"required,max=100"`
	} `json:"address"`
	Email   string `json:"email" validate:"required,email,max=100"`
	Company struct {
		Name string `json:"name" validate:"required,max=100"`
		INN  string `json:"inn" validate:"required,numeric,min=10,max=12"`
		OGRN string `json:"ogrn" validate:"required,numeric,min=13,max=15"`
	} `json:"company"`
	Social []Social `json:"social" validate:"max=10,dive"`
}
```
Extra rules after `validate.Struct` (400 with the dotted field, like `validate`): a `Number`/`AltNumber` matches
`^\+?\d{10,15}$` (`номер: только цифры и «+»`); `AltNumber` is required when `AltDisplay` is set and the other way
round (`заполните оба поля`); `Social[i].URL` starts with `https://` (`ссылка должна начинаться с https://`).
Trim every string before validating. `Social` `null` → `[]`.

### Storage
```sql
-- +goose Up
CREATE TABLE site_settings (
    id         smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    data       jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO site_settings (data) VALUES ('{...}');

-- +goose Down
DROP TABLE site_settings;
```
Seed `data` from `site.ts`: the four phones (`display` as written there with normal spaces, `Number` = the digits of
`href` after `tel:`, `alt` → `AltDisplay`/`AltNumber`), `address` (postcode, city, street; `full` is not stored, the
site builds it), `email`, `company`, and `social` (`href` → `url`; `handle` is dropped).

`Store.Get(ctx) (Settings, time.Time, error)` unmarshals `data`; `Store.Put(ctx, Settings) (time.Time, error)` updates
it (`updated_at = now()`).

### Routes
| Method | Path | Role | Behaviour | Success |
|---|---|---|---|---|
| GET | `/settings` | public | the settings and `updatedAt` | 200 |
| PUT | `/settings` | admin | replace the whole object | 200 |

Response: the `Settings` JSON plus `"updatedAt"`. PUT body: the `Settings` JSON (an `updatedAt` in the body is ignored).

## Tests to write
- `TestSeed`: `GET /settings` → restaurant `display` `8 918 311-97-91`, `number` `+79183119791`, `altNumber`
  `+78615533003`; free `number` `88001002441`; two social links; postcode `352630`.
- `TestPutSettings`: change the e-mail and remove one social link → 200; GET shows the change.
- `TestPutSettingsTrims`: `"  a@b.ru "` → stored `a@b.ru`.
- `TestPutSettingsInvalid`: table → 400 with the field: `email` `abc`; `address.postcode` `3526`;
  `phones.hotel.number` `8 918`; `phones.hotel.altDisplay` set and `altNumber` empty; `social[0].url` `http://vk.com`;
  `company.inn` `abc`.
- `TestPutSettingsAuth`: no token → 401; role `user` → 403.

## API documentation
`docs/API.md`, `### Settings`: both routes with the full JSON example from the seed; notes "PUT replaces everything;
send the object you got from GET with your changes", "the site builds `tel:` links from `number`".

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/settings` is at least 80%, docs updated.

## Out of scope
Opening hours, navigation, legal pages, the site (citypark-modern task 19).
