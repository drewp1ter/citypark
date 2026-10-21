# Task 09: Clock, sale windows and money

## Goal
Three small packages of pure logic with no database and no HTTP.

## Depends on
01.

## Read first
`docs/tasks/CONVENTIONS.md`.

## Files to create
- `internal/clock/clock.go`
- `internal/salewindow/salewindow.go`, `internal/salewindow/salewindow_test.go`
- `internal/money/money.go`, `internal/money/money_test.go`

## Spec

### `clock`
```go
type Clock interface{ Now() time.Time }

// System returns the real time converted to loc.
func System(loc *time.Location) Clock

// Fixed always returns t. Used in tests.
func Fixed(t time.Time) Clock
```

### `salewindow`
A sale window limits when a category or product can be ordered.

```go
type Window struct {
    Start *int   // minutes from midnight; nil = no time limit
    End   *int   // minutes from midnight; nil = no time limit
    Days  string // "" = every day, else 7 chars of '0'/'1', Monday first
}

// ParseClock parses "H:MM" or "HH:MM" (surrounding spaces allowed) into minutes from midnight.
// Hours 0..24, minutes 0..59, result at most 1440. "9:00" -> 540, "21:30" -> 1290.
func ParseClock(s string) (minutes int, ok bool)

// Parse reads the iiko additionalInfo value: a JSON array [startTime, endTime, daysOfWeek].
// It never fails: anything it cannot read means "no restriction".
func Parse(additionalInfo string) Window

func (w Window) OpenAtTime(t time.Time) bool
func (w Window) OpenOnDay(t time.Time) bool
func (w Window) StartString() string // "9:00", or "" when Start is nil
func (w Window) EndString() string
func (w Window) DaysString() string  // "Пн, Вт, Ср", or "" when Days is ""
```

`Parse` rules:
- Decode into `[]json.RawMessage`. Invalid JSON, empty string, or not an array → zero `Window`.
- Elements 0 and 1 are JSON strings parsed with `ParseClock`. `Start` and `End` are set only when both are present and valid; otherwise both stay nil.
- Element 2 may be a JSON string (`"1111100"`) or a JSON number (`1111100`). Take its text without quotes. If it matches `^[01]{1,7}$`, pad it on the right with `0` to 7 chars and store it in `Days`; otherwise `Days` is `""`.

Method rules:
- `OpenAtTime`: true when `Start` or `End` is nil; else `m := t.Hour()*60 + t.Minute(); *Start <= m && m < *End`.
- `OpenOnDay`: true when `Days == ""`; else index = `(int(t.Weekday()) + 6) % 7` (Monday = 0) and the char at that index is `'1'`.
- `StartString` / `EndString`: `fmt.Sprintf("%d:%02d", m/60, m%60)`.
- `DaysString`: names `Пн, Вт, Ср, Чт, Пт, Сб, Вс` for each `'1'`, joined with `", "`.

All methods use `t` as given; the caller is responsible for the time zone.

### `money`
```go
// Kopecks is an amount of money in 1/100 of a ruble.
type Kopecks int64

func FromRubles(r float64) Kopecks // math.Round(r * 100)
func (k Kopecks) Rubles() float64
func (k Kopecks) Mul(n int) Kopecks

// String formats rubles: "150" for whole rubles, "150.50" otherwise (always two decimals).
func (k Kopecks) String() string

// MarshalJSON writes rubles as a JSON number: 15000 -> 150, 599 -> 5.99, 1050 -> 10.5.
func (k Kopecks) MarshalJSON() ([]byte, error)

// UnmarshalJSON reads a JSON number of rubles. Anything that is not a number is an error.
func (k *Kopecks) UnmarshalJSON(b []byte) error
```
Use `strconv.FormatFloat(float64(k)/100, 'f', -1, 64)` for `MarshalJSON`.

## Tests to write
`salewindow_test.go` (table tests):
- `TestParseClock`: `"9:00"`→540, `" 21:30 "`→1290, `"24:00"`→1440, `"0:00"`→0; invalid: `""`, `"9"`, `"25:00"`, `"9:60"`, `"a:b"`, `"24:01"`.
- `TestParse`:

| Input | Start | End | Days |
|---|---|---|---|
| `""` | nil | nil | `""` |
| `not json` | nil | nil | `""` |
| `["9:00", "21:00", 1111100]` | 540 | 1260 | `1111100` |
| `["9:00", "21:00", "0000011"]` | 540 | 1260 | `0000011` |
| `["9:00"]` | nil | nil | `""` |
| `["", "", "11111"]` | nil | nil | `1111100` |
| `["9:00", "21:00", "abc"]` | 540 | 1260 | `""` |

- `TestOpenAtTime` with window 9:00–21:00: 08:59 false, 09:00 true, 20:59 true, 21:00 false; zero `Window` at 03:00 true.
- `TestOpenOnDay` with `Days: "1111100"`: 2024-05-24 (Friday) true, 2024-05-25 (Saturday) false, 2024-05-26 (Sunday) false; zero `Window` true.
- `TestStrings`: window 9:00–21:00 with `"1111100"` gives `"9:00"`, `"21:00"`, `"Пн, Вт, Ср, Чт, Пт"`; zero `Window` gives three empty strings.

`money_test.go`:
- `TestFromRubles`: 5.99→599, 320→32000, 0.1+0.2→30.
- `TestString`: 15000→`"150"`, 15050→`"150.50"`, 599→`"5.99"`, 0→`"0"`.
- `TestJSON`: marshal `struct{P Kopecks}` for 15000, 599, 1050 gives `150`, `5.99`, `10.5`; unmarshal `{"P": 5.99}` gives 599; unmarshal `{"P": "5"}` is an error.
- `TestMul`: `Kopecks(599).Mul(3)` is 1797.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Database columns and HTTP. These packages import only the standard library.
