# Task 58: Moneta payment URL

## Goal
`POST /orders` with `paymentType: "online"` returns `redirectUrl`: the full MONETA.Assistant URL with every `MNT_*`
parameter and the signature, so the website only has to send the customer there (citypark-modern task 26). Online and
cash orders then have the same response shape: `{ orderId, redirectUrl }`.

## Depends on
30.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/moneta/moneta.go` and its test, `internal/app/payments.go`,
`internal/order/service.go` (`PaymentForm`, `CreateResult`, `Create`, `PaymentConfirmed`).

## Background
The owner wants the backend to build the payment URL. Example (demo shop):
`https://assistant.demo.moneta.ru/?MNT_ID=50951076&MNT_AMOUNT=100.00&MNT_TRANSACTION_ID=8a96f492-...&MNT_SIGNATURE=ad34...`.
Our `MNT_TRANSACTION_ID` is the order id (an integer), as `RequestSignature` already signs.

From docs.moneta.ru (`/assistant/v1/payment-request/`):
- `MNT_TEST_MODE` defaults to `0` and `MNT_CURRENCY_CODE` has no default. The signature includes both, so **both must
  be in the URL**, or Moneta rejects a test-mode payment with a signature error.
- `MNT_SUCCESS_URL` and `MNT_FAIL_URL` are not signed. Moneta uses them only when «Можно переопределять настройки в
  URL» is turned on in the shop settings; Moneta appends `MNT_TRANSACTION_ID` to them.
- Production assistant: `https://www.payanyway.ru/assistant.htm`; demo: `https://assistant.demo.moneta.ru/`
  (the owner's example) or `https://demo.moneta.ru/assistant.htm`.

## Files to change
- `internal/config/config.go`, `internal/config/config_test.go`: three new fields on `Moneta`.
- `internal/moneta/moneta.go`, `internal/moneta/moneta_test.go`: `PaymentURL`.
- `internal/app/payments.go`, `internal/app/app.go`, `internal/app/app_test.go`: the adapter fills `PaymentForm.URL`.
- `internal/order/service.go`, `internal/order/service_test.go` (and `create_handler_test.go` if it checks the online
  body): online result.
- `internal/apidocs/openapi.yaml` (`CreateOrderResult` and the `POST /orders` description), `docs/API.md`
  (`#### POST /orders`), `README.md` (env table, Moneta paragraph), `.env.example`.

## Spec

### Config
```go
type Moneta struct {
	ID         string // shop id, sent to Moneta as MNT_ID
	Secret     string // integrity code from the Moneta cabinet
	TestMode   bool
	URL        string // MONETA.Assistant address the customer is sent to
	SuccessURL string // MNT_SUCCESS_URL; empty = not sent
	FailURL    string // MNT_FAIL_URL; empty = not sent
}
```
| Variable | Default |
|---|---|
| `MONETA_URL` | `https://assistant.demo.moneta.ru/` |
| `MONETA_SUCCESS_URL` | `http://localhost:3001/checkout/done` |
| `MONETA_FAIL_URL` | `http://localhost:3001/checkout?payment=failed` |

Add them to `.env.example` under `# Moneta` (production value of `MONETA_URL` in a comment) and to the README table.

### moneta
```go
// PaymentURL returns the MONETA.Assistant address for an order: base followed by the query
// MNT_ID, MNT_AMOUNT, MNT_TRANSACTION_ID, MNT_CURRENCY_CODE, MNT_TEST_MODE, MNT_SIGNATURE and,
// when not empty, MNT_SUCCESS_URL and MNT_FAIL_URL. amount comes from FormatAmount.
func (s *Signer) PaymentURL(base string, orderID int64, amount, successURL, failURL string) string
```
- Build the query with `url.Values` and `Encode()` (values are escaped, keys sorted). Result: `base + "?" + query`.
  `base` is used as given (it has no query string of its own); do not parse it.
- `MNT_CURRENCY_CODE` = `Currency` (`RUB`), `MNT_TEST_MODE` = `s.testMode` (`"0"`/`"1"`),
  `MNT_SIGNATURE` = `s.RequestSignature(orderID, amount)` (unchanged).

### app adapter
```go
// PaymentPages are the Moneta addresses the customer is sent to and back from.
type PaymentPages struct{ Base, Success, Fail string }

func NewMonetaPayments(signer *moneta.Signer, pages PaymentPages) *MonetaPayments
```
`Form` fills `PaymentForm{Amount, Signature, URL}` with `URL` from `signer.PaymentURL`. `app.New` passes
`PaymentPages{cfg.Moneta.URL, cfg.Moneta.SuccessURL, cfg.Moneta.FailURL}`.

### order
- `PaymentForm` gets `URL string // full MONETA.Assistant address`.
- `CreateResult` loses `Amount` and `MntSignature`; comment: `// CreateResult: RedirectURL is the thank-you page for a
  cash order and the Moneta payment page for an online order.`
- `Create` for online: `return CreateResult{OrderID: o.ID, RedirectURL: form.URL}, nil` (the status change to
  `pending` stays as it is). The duplicate-order case returns the same URL for the same order.
- `PaymentConfirmed` keeps comparing with `Form(...).Amount`: no change.

### Response
```json
{ "orderId": 1, "redirectUrl": "https://assistant.demo.moneta.ru/?MNT_AMOUNT=790.00&MNT_CURRENCY_CODE=RUB&MNT_FAIL_URL=http%3A%2F%2Flocalhost%3A3001%2Fcheckout%3Fpayment%3Dfailed&MNT_ID=100&MNT_SIGNATURE=8c65fcd4afe3dc4cb915e04aaa273574&MNT_SUCCESS_URL=http%3A%2F%2Flocalhost%3A3001%2Fcheckout%2Fdone&MNT_TEST_MODE=0&MNT_TRANSACTION_ID=15" }
```
(Check the example against the real output of the test below and paste the real one.)

## Tests to write
- `TestPaymentURL` (table, `New("100", "secret", testMode)`, order 15, amount `"790.00"`, base
  `https://assistant.demo.moneta.ru/`); parse the result with `url.Parse` and compare `Query()` and the part before `?`:
  - live, both pages set → all 8 keys; `MNT_TEST_MODE=0`, `MNT_SIGNATURE=8c65fcd4afe3dc4cb915e04aaa273574`,
    `MNT_CURRENCY_CODE=RUB`, `MNT_FAIL_URL` with its own `?payment=failed` intact after decoding;
  - test mode → `MNT_TEST_MODE=1`, `MNT_SIGNATURE=047418b1c4a31fbc826ff955c7bdf010`;
  - empty success and fail → exactly 6 keys, no `MNT_SUCCESS_URL`/`MNT_FAIL_URL`.
- `TestPaymentsAdapter`: `NewMonetaPayments(moneta.New("100", "secret", false), PaymentPages{Base: "https://pay.test/"})`;
  `Form` → `Amount "790.00"`, the old signature, and `URL` starting with `https://pay.test/?` and containing
  `MNT_TRANSACTION_ID=15`. The `Verify` part stays.
- `TestCreateOnline`: `fakePayments.form` gets `URL: "https://pay.test/?x"` → `res.RedirectURL == "https://pay.test/?x"`;
  the other checks stay.
- `TestCreateResultJSON`: online `CreateResult{OrderID: 1, RedirectURL: "https://pay"}` → keys `orderId, redirectUrl`.
- The line `res.Amount != "" || res.MntSignature != ""` in the cash test is removed (the fields no longer exist).
- `config_test.go`: defaults and the full-env case include the three variables.

## API documentation
`docs/API.md` `#### POST /orders`: replace the online response with the one above; Notes: «For both payment types the
frontend sends the customer to `redirectUrl`. For an online order it is the Moneta payment page; after paying Moneta
sends the customer to `MONETA_SUCCESS_URL` (or `MONETA_FAIL_URL`) with `MNT_TRANSACTION_ID` appended». Keep the note
about repeated identical online orders. Same change in `openapi.yaml` (`CreateOrderResult`: remove `amount` and
`mntSignature`, describe `redirectUrl` for both types; `POST /orders` description at about line 1793).

README Moneta paragraph: replace «The frontend must use the same shop id…» with: «The server builds the payment URL
(`MONETA_URL`: demo or production assistant). Turn on «Можно переопределять настройки в URL» in the shop settings so
`MONETA_SUCCESS_URL` and `MONETA_FAIL_URL` are used. The demo and production shops have different `MONETA_ID`.»

## Done when
CONVENTIONS "Done means" pass (coverage of `moneta`, `app`, `order`, `config` at least 80%) and `docs/API.md` is
updated. Owner check (write it as a note under the PROGRESS line, not done by the agent): with the demo shop's real
`MONETA_ID`/`MONETA_SECRET`, open `redirectUrl` from a `POST /orders` and see the payment form, not a signature error.

## Out of scope
- The website (citypark-modern task 26) and the admin panel.
- Check URL, `MNT_SUBSCRIBER_ID`, `MNT_DESCRIPTION`, receipts (54-ФЗ), the Moneta widget.
- Changing the signature or the Pay URL callback.
