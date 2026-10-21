# Task 02: Config

## Goal
One package that reads all settings from environment variables, with defaults.

## Depends on
01.

## Read first
`docs/tasks/CONVENTIONS.md`.

## Files to change
- `internal/config/config.go`
- `internal/config/config_test.go`

Both files already exist from the first pass of this task, which was written for YooKassa. Change them to match the spec below: the `Yookassa` settings and `ThankYouPageOnline` go away, `Moneta` comes in. Everything else stays as it is.

## Spec

```go
package config

type Config struct {
    Env             string         // "development", "production" or "test"
    Port            int
    Debug           bool
    Timezone        *time.Location
    DefaultPageSize int
    DatabaseURL     string
    JWT             JWT
    Admin           Admin
    Iiko            Iiko
    Telegram        Telegram
    Moneta          Moneta
    WorkingHours    WorkingHours
    ThankYouPageCash string
}

type JWT struct {
    Secret    string
    ExpiresIn time.Duration
}
type Admin struct{ Email, Password, Phone string }
type Iiko struct{ BaseURL, APILogin, OrganizationID, CategoryMarker, UpdateToken string }
type Telegram struct{ BaseURL, Token, ChatIDOrderCreated, ChatIDOrderConfirmed string }
type Moneta struct {
    ID       string // shop id, sent to Moneta as MNT_ID
    Secret   string // integrity code from the Moneta cabinet
    TestMode bool
}
type WorkingHours struct{ Start, End string } // "9:00"; empty string means no restriction

// LookupFunc has the signature of os.LookupEnv.
type LookupFunc func(key string) (string, bool)

// Load reads the configuration. Pass os.LookupEnv in production code.
func Load(lookup LookupFunc) (Config, error)

func (c Config) IsProduction() bool
func (c Config) IsTest() bool
```

### Variables and defaults

| Variable | Field | Default |
|---|---|---|
| `APP_ENV` | `Env` | `development` |
| `PORT` | `Port` | `3000` |
| `DEBUG` | `Debug` | `false` (true only for the value `true`) |
| `APP_TIMEZONE` | `Timezone` | `Europe/Moscow` |
| `DEFAULT_PAGE_SIZE` | `DefaultPageSize` | `100` |
| `DATABASE_URL` | `DatabaseURL` | `postgres://postgres:postgres@localhost:5432/citypark_dev?sslmode=disable` |
| `JWT_SECRET` | `JWT.Secret` | `changeme` |
| `JWT_EXPIRES_IN` | `JWT.ExpiresIn` | `24h` (parsed with `time.ParseDuration`) |
| `ADMIN_EMAIL` | `Admin.Email` | `dev@dev.io` |
| `ADMIN_PASSWORD` | `Admin.Password` | `devdevdev` |
| `ADMIN_PHONE` | `Admin.Phone` | `+79990000000` |
| `IIKO_BASE_URL` | `Iiko.BaseURL` | `https://api-ru.iiko.services/api/1` |
| `IIKO_API_LOGIN` | `Iiko.APILogin` | empty |
| `IIKO_ORGANIZATION_ID` | `Iiko.OrganizationID` | empty |
| `IIKO_CATEGORY_MARKER` | `Iiko.CategoryMarker` | `#` |
| `IIKO_UPDATE_TOKEN` | `Iiko.UpdateToken` | empty |
| `TELEGRAM_BASE_URL` | `Telegram.BaseURL` | `https://api.telegram.org` |
| `TELEGRAM_TOKEN` | `Telegram.Token` | empty |
| `TELEGRAM_CHAT_ID_FOR_ORDER_CREATED` | `Telegram.ChatIDOrderCreated` | empty |
| `TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED` | `Telegram.ChatIDOrderConfirmed` | empty |
| `MONETA_ID` | `Moneta.ID` | empty |
| `MONETA_SECRET` | `Moneta.Secret` | empty |
| `MONETA_TEST_MODE` | `Moneta.TestMode` | `false` (true only for the value `true`) |
| `WORKING_HOURS_START` | `WorkingHours.Start` | `9:00` |
| `WORKING_HOURS_END` | `WorkingHours.End` | `22:00` |
| `THANK_YOU_PAGE_PAYMENT_CASH` | `ThankYouPageCash` | `http://localhost:3001/thankyou` |

### Rules
- A variable that is not set uses the default. A variable that is set to an empty string keeps the empty string. This matters for `WORKING_HOURS_START` / `WORKING_HOURS_END`: setting them to empty disables the working-hours check.
- `Load` returns an error (naming the variable) when `PORT` or `DEFAULT_PAGE_SIZE` is not a positive integer, `JWT_EXPIRES_IN` is not a valid duration, `APP_TIMEZONE` cannot be loaded with `time.LoadLocation`, or `APP_ENV` is not one of the three allowed values.
- `IIKO_CATEGORY_MARKER` set to an empty string is an error: with an empty marker every iiko group would become a site category.
- When `Env` is `production` and `JWT.Secret` is `changeme`, `Load` returns the error `JWT_SECRET must be changed in production`.
- `Load` must not ask `lookup` for `YOOKASSA_BASE_URL`, `YOOKASSA_SHOP_ID`, `YOOKASSA_SECRET_KEY` or `THANK_YOU_PAGE_PAYMENT_ONLINE` any more.
- Do not read `os.Getenv` anywhere in this package; use only the `lookup` argument. Do not load `.env` files.

## Tests to write
Use a `map[string]string` wrapped as a `LookupFunc`.
- `TestLoadDefaults`: an empty map gives the defaults above (check at least `Port`, `Timezone.String()`, `JWT.ExpiresIn`, `Iiko.CategoryMarker`, `WorkingHours`, and `Moneta` equal to the zero value).
- `TestLoadOverrides`: every string and number variable can be overridden (check five of them, and `Moneta` as a whole: `MONETA_ID=100`, `MONETA_SECRET=s`, `MONETA_TEST_MODE=true` → `Moneta{"100", "s", true}`).
- `TestLoadMonetaTestMode`: table for `MONETA_TEST_MODE`: not set → false, `true` → true, `1` → false, empty → false.
- `TestLoadNoOldVariables`: a lookup function that records the keys it is asked for; after `Load` none of them starts with `YOOKASSA_` and `THANK_YOU_PAGE_PAYMENT_ONLINE` is not among them.
- `TestLoadEmptyWorkingHours`: `WORKING_HOURS_START=""` gives `WorkingHours.Start == ""`.
- `TestLoadErrors`: table test for a bad `PORT`, bad `JWT_EXPIRES_IN`, bad `APP_TIMEZONE`, bad `APP_ENV`, empty `IIKO_CATEGORY_MARKER`.
- `TestLoadProductionSecret`: `APP_ENV=production` with the default secret fails; with `JWT_SECRET=x` it succeeds.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Using the config in `main.go` (task 31).
