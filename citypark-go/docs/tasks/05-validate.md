# Task 05: Validation

## Goal
Validate request structs with `go-playground/validator` and return Russian messages in the project's error format. Normalise Russian phone numbers.

## Depends on
03.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/httpx/httpx.go`.

## Files to create
- `internal/validate/validate.go`
- `internal/validate/phone.go`
- `internal/validate/validate_test.go`

Run `go get github.com/go-playground/validator/v10`.

## Spec

```go
package validate

// Struct validates v using its `validate` struct tags.
// It returns nil, or *httpx.Error with
//   Status  400
//   Message "Ошибка валидации, пожалуйста проверьте введенные вами данные"
//   Fields  one entry per invalid field
func Struct(v any) error

// NormalizePhone converts a Russian mobile or landline number to "+7XXXXXXXXXX".
// ok is false when the input is not a valid Russian number.
func NormalizePhone(s string) (phone string, ok bool)
```

### `NormalizePhone`
1. Remove spaces, `-`, `(` and `)`.
2. Accept exactly these shapes: `+7` + 10 digits, `7` + 10 digits, `8` + 10 digits.
3. Return `+7` + the 10 digits. Anything else returns `"", false`.

| Input | Output |
|---|---|
| `+79991234567` | `+79991234567`, true |
| `8 (999) 123-45-67` | `+79991234567`, true |
| `79991234567` | `+79991234567`, true |
| `9991234567` | false |
| `+7999123456` | false |
| `+7999123456a` | false |
| empty | false |

### Validator setup
Create one `*validator.Validate` inside the package (a package-level variable initialised once is allowed here) with:
- `RegisterTagNameFunc` so that field names come from the `json` tag (the part before the comma). A field with `json:"-"` keeps its Go name.
- A custom tag `ruphone`: valid when `NormalizePhone` returns ok.

### Field keys
The key in `Fields` is the JSON path of the field without the root struct name:

| Go | Key |
|---|---|
| top-level field `Phone` with `json:"phone"` | `phone` |
| nested struct `Address.City` | `address.city` |
| slice element `Items[0].Amount` | `items[0].amount` |

Build it from `validator.FieldError.Namespace()` by removing everything up to and including the first `.`.

### Messages
Chosen by `FieldError.Tag()`; `{param}` is `FieldError.Param()`:

| Tag | Message |
|---|---|
| `required` | `поле не должно быть пустым` |
| `email` | `email должен быть электронной почтой` |
| `ruphone` | `телефон должен быть действительным номером телефона` |
| `min` | `значение должно быть не меньше {param}` |
| `max` | `значение должно быть не больше {param}` |
| `gt` | `значение должно быть больше {param}` |
| `gte` | `значение должно быть не меньше {param}` |
| `oneof` | `недопустимое значение` |
| anything else | `некорректное значение` |

If `validator` returns an error that is not `validator.ValidationErrors` (for example `InvalidValidationError`), return it unchanged so that it becomes a 500.

## Tests to write
Define test structs in the test file, for example:
```go
type item struct {
    ProductID int64 `json:"productId" validate:"required,gt=0"`
    Amount    int   `json:"amount" validate:"gt=0"`
}
type addr struct {
    City string `json:"city" validate:"required,max=80"`
}
type req struct {
    Name    string `json:"name" validate:"required,max=80"`
    Email   string `json:"email" validate:"omitempty,email"`
    Phone   string `json:"phone" validate:"required,ruphone"`
    Kind    string `json:"kind" validate:"oneof=cash online"`
    Address *addr  `json:"address" validate:"omitempty"`
    Items   []item `json:"items" validate:"required,min=1,dive"`
}
```
- `TestStructValid`: a fully valid value returns nil.
- `TestStructMessages`: an invalid value returns `*httpx.Error` with status 400, the exact top-level message, and the expected key and message for `name`, `email`, `phone`, `kind`, `address.city`, `items[0].amount`.
- `TestStructOptionalEmail`: empty `Email` is valid.
- `TestNormalizePhone`: the table above.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Request structs of real endpoints; they are defined in their own tasks.
