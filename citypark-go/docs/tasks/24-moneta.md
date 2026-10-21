# Task 24: Moneta signatures

## Goal
The MONETA.RU (PayAnyWay) payment signatures: one for the payment form that the frontend opens, one to check the payment callback. No network calls.

## Depends on
09.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/money/money.go`.

## Files to create
- `internal/moneta/moneta.go`
- `internal/moneta/moneta_test.go`

This package must not import `internal/order`.

## Background
Moneta does not have a "create payment" call. The frontend sends the customer to the Moneta payment form with the shop id, the order id, the amount and a signature. Only the backend knows the secret, so only the backend can compute the signature. After the customer pays, Moneta calls the backend (task 30) with its own signature, which the backend checks with the same secret.

## Spec

```go
package moneta

const Currency = "RUB"

type Signer struct { /* id, secret, testMode string */ }

// New: id is the shop id (MNT_ID), secret is the integrity code from the Moneta cabinet.
// testMode is stored as "1" or "0".
func New(id, secret string, testMode bool) *Signer

// FormatAmount returns the amount in rubles with exactly two decimals: 79000 → "790.00".
func FormatAmount(k money.Kopecks) string

// RequestSignature signs the payment form for an order. amount comes from FormatAmount.
func (s *Signer) RequestSignature(orderID int64, amount string) string

// VerifyCallback reports whether signature is the right one for a payment callback.
func (s *Signer) VerifyCallback(transactionID, operationID, amount, signature string) bool
```

### `FormatAmount`
`fmt.Sprintf("%d.%02d", k/100, k%100)`.

### Signatures
Both are the lowercase hex MD5 of a plain concatenation, with no separators:

| Signature | Concatenation |
|---|---|
| request | `id + orderID (decimal) + amount + "RUB" + testMode + secret` |
| callback | `id + transactionID + operationID + amount + "RUB" + testMode + secret` |

Moneta's formula also has a subscriber id between the currency and the test mode; this project does not use it, so it is an empty string and is left out.

`id` and `testMode` always come from the `Signer`, never from the callback.

### `VerifyCallback`
- Compute the callback signature and compare it with `signature` using `subtle.ConstantTimeCompare`, after `strings.ToLower` on `signature`.
- Return `false` when `transactionID`, `operationID`, `amount` or `signature` is empty.

Never put the secret into an error or a log message.

## Tests to write
Use `New("100", "secret", false)` unless the test says otherwise. No HTTP server is needed.

- `TestFormatAmount`: table for 79000 → `790.00`, 599 → `5.99`, 5 → `0.05`, 100050 → `1000.50`, 0 → `0.00`.
- `TestRequestSignature`: table.

  | testMode | orderID | amount | Expected |
  |---|---|---|---|
  | false | 15 | `790.00` | `8c65fcd4afe3dc4cb915e04aaa273574` |
  | true | 15 | `790.00` | `047418b1c4a31fbc826ff955c7bdf010` |
  | false | 15 | `0.05` | `0bd20edc998ca5c8e7be3780211c47f5` |

- `TestVerifyCallback`: table with transaction id `15`, operation id `555`, amount `790.00`.

  | Case | Result |
  |---|---|
  | signature `998b6268fe3a6c7f4eeb8ccec65a9566` | true |
  | the same signature in upper case | true |
  | signer in test mode, signature `700d371ec181494bb14aacaa23399dea` | true |
  | signer in test mode, signature `998b6268fe3a6c7f4eeb8ccec65a9566` | false |
  | amount `790.01` | false |
  | operation id `556` | false |
  | transaction id `16` | false |
  | another secret in the signer | false |
  | the request signature `8c65fcd4afe3dc4cb915e04aaa273574` | false |
  | empty signature | false |
  | empty operation id (with the signature computed for an empty one) | false |

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
The callback endpoint (task 30). Building the payment form URL (the frontend does it). Refunds, the Moneta Check URL, subscriber ids.
