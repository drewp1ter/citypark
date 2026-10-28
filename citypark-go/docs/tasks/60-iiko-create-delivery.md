# Task 60: iiko delivery client

## Goal
`(*iiko.Client).CreateDelivery` creates a delivery order in iiko Cloud (`api/1/deliveries/create`). It waits through
`api/1/commands/status` until iiko has created the order or failed, and returns the iiko order id. Nothing calls it yet:
task 62 sends paid orders through it.

## Depends on
37.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/client.go`, `internal/iiko/client_test.go` (see `newTestClient` and how the
fake server answers `v2/access_token`).

## Background
Facts from the iiko Cloud API spec (`https://api.iiko.help/docs/en/swagger.json`, `POST /api/1/deliveries/create`):
- The method is a *command*: the answer may come before iikoFront has accepted the order. `orderInfo.creationStatus` is
  `Success`, `InProgress` or `Error`. For `InProgress`, `POST /api/1/commands/status` with
  `{organizationId, correlationId}` answers `{"state": "InProgress" | "Success" | "Error", "errorReason": "..."}`.
- `order.id` must be a unique GUID. If iiko gets the same id twice, it does not create a second order. Task 62 relies on
  this for retries.
- `phone` must start with `+` (our phones are `+7XXXXXXXXXX`).
- Items, customer, address and payments are discriminated by a field: item `type: "Product"`, customer
  `type: "one-time"`, address `type: "legacy"`, payment `paymentTypeKind` (`Cash`, `Card`, `External`, ...).
- Error answers (400, 401, 408, 500) have the body `{"correlationId": "...", "errorDescription": "...", "error": "..."}`.

## Files to create
- `internal/iiko/delivery.go`
- `internal/iiko/delivery_test.go`

## Files to change
- `internal/iiko/client.go`: `post` returns `*StatusError`. `Client` gets the fields `pollEvery` and `pollTimes`.

## Spec

### Request types (`delivery.go`)
```go
const (
	ServiceDeliveryByCourier = "DeliveryByCourier"
	ServiceDeliveryByClient  = "DeliveryByClient" // pickup
)

// DeliveryOrder is the "order" object of deliveries/create.
type DeliveryOrder struct {
	ID               string            `json:"id"`                       // GUID, made by the caller
	ExternalNumber   string            `json:"externalNumber,omitempty"` // our order id
	Phone            string            `json:"phone"`
	OrderServiceType string            `json:"orderServiceType"`
	DeliveryPoint    *DeliveryPoint    `json:"deliveryPoint,omitempty"` // nil for pickup
	Comment          string            `json:"comment,omitempty"`
	Customer         DeliveryCustomer  `json:"customer"`
	Items            []DeliveryItem    `json:"items"`
	Payments         []DeliveryPayment `json:"payments"`
}

type DeliveryCustomer struct {
	Type string `json:"type"` // "one-time"
	Name string `json:"name"`
}

type DeliveryPoint struct {
	Address DeliveryAddress `json:"address"`
	Comment string          `json:"comment,omitempty"`
}

type DeliveryAddress struct {
	Type   string         `json:"type"` // "legacy"
	Street DeliveryStreet `json:"street"`
	House  string         `json:"house"`
}

type DeliveryStreet struct {
	Name string `json:"name"`
	City string `json:"city"`
}

type DeliveryItem struct {
	Type      string  `json:"type"` // "Product"
	ProductID string  `json:"productId"`
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price"` // rubles per unit
}

type DeliveryPayment struct {
	PaymentTypeKind       string  `json:"paymentTypeKind"`
	Sum                   float64 `json:"sum"` // rubles
	PaymentTypeID         string  `json:"paymentTypeId"`
	IsProcessedExternally bool    `json:"isProcessedExternally"`
}

type DeliveryResult struct{ ID string } // iiko's order id

// DeliveryError: iiko answered creationStatus "Error" (or the command ended with "Error").
type DeliveryError struct{ Reason string }

func (e *DeliveryError) Error() string // "iiko: delivery not created: " + Reason

// ErrDeliveryPending: iiko still reported InProgress after the last status check. The order may yet appear in iikoFront.
var ErrDeliveryPending = errors.New("iiko: delivery creation not confirmed")

// CreateDelivery sends the order to deliveries/create on the given terminal group and waits until it is created.
func (c *Client) CreateDelivery(ctx context.Context, terminalGroupID string, o DeliveryOrder) (DeliveryResult, error)

// Permanent reports whether err cannot be fixed by sending the same request again:
// a *DeliveryError, or a *StatusError with status 400.
func Permanent(err error) bool
```

### CreateDelivery
1. `withToken` + `post(ctx, "1/deliveries/create", token, body, &resp)`. Note that `baseURL` already ends in `/api`, so
   the endpoint is `1/deliveries/create`. Body:
   ```json
   {
     "organizationId": "<c.organizationID>",
     "terminalGroupId": "<terminalGroupID>",
     "createOrderSettings": { "transportToFrontTimeout": 180 },
     "order": { ...DeliveryOrder... }
   }
   ```
   Response struct (only these fields):
   `{correlationId string; orderInfo {id string; creationStatus string; errorInfo *{code, message, description string}}}`.
2. `creationStatus`:
   - `Success`: return `DeliveryResult{ID: orderInfo.id}`.
   - `Error`: return `&DeliveryError{Reason: ...}`. Reason is the first non-empty of `errorInfo.message`,
     `errorInfo.description` and `errorInfo.code`, or `"unknown error"`.
   - `InProgress`: poll (step 3).
   - Anything else: return `fmt.Errorf("iiko: 1/deliveries/create: unknown creationStatus %q", s)`.
3. Polling: up to `c.pollTimes` times, first wait `c.pollEvery` (`select` on `time.After` and `ctx.Done()`; return
   `ctx.Err()` when the context ends). Then call `withToken` + `post(ctx, "1/commands/status", token,
   {organizationId, correlationId}, &st)`, where `st` is `{state, errorReason string}`. Each poll takes the token lock
   on its own, so a menu sync is not blocked while we wait.
   - `Success`: return the id from step 1.
   - `Error`: return `&DeliveryError{Reason: errorReason}` (or `"unknown error"` when it is empty).
   - `InProgress`: keep polling.
   - Any other state is the same kind of error as in step 2.
   - A poll that returns an error returns that error.
   - After the last poll, return `ErrDeliveryPending`.
4. `NewClient` sets `pollEvery: 2 * time.Second`, `pollTimes: 15`. Tests in the package set them directly.

### StatusError (`client.go`)
```go
// StatusError: iiko answered with a status other than 200 and 401.
type StatusError struct {
	Endpoint    string
	Status      int
	Description string // errorDescription from the body, "" when the body has none
}

func (e *StatusError) Error() string
```
`Error()` is `iiko: <Endpoint>: unexpected status <Status>`, plus `: <Description>` when there is one. Without a
description the text is the same as today, so `TestAccessTokenUnexpectedStatus` stays unchanged. In `post`, the
`default:` branch decodes `{"errorDescription": string}` from the body and ignores decode errors.

## Tests to write
`delivery_test.go`, using `newTestClient` with a handler that routes by `r.URL.Path`. It answers `v2/access_token`
as the existing tests do. Set `c.pollEvery = time.Millisecond`.
- `TestCreateDeliverySuccess`: answer `creationStatus: "Success"`, `id: "iiko-1"`. Expect `DeliveryResult{ID: "iiko-1"}`,
  and decode the request body into `map[string]any` to check:
  - `organizationId` and `terminalGroupId`;
  - `createOrderSettings.transportToFrontTimeout = 180`;
  - `order.id`, `order.phone`, `order.customer.type = "one-time"`;
  - `order.items[0]` with `type`, `productId`, `amount`, `price`;
  - `order.payments[0]` with `paymentTypeKind`, `sum`, `paymentTypeId`, `isProcessedExternally: true`;
  - `order.deliveryPoint` is missing when it is nil;
  - the header `Authorization: Bearer <token>`.
- `TestCreateDeliveryWithAddress`: `deliveryPoint.address` is `{type: legacy, street: {name, city}, house}`.
- `TestCreateDeliveryErrorStatus` (table):
  - `errorInfo.message` set gives that reason;
  - only `description` set gives the description;
  - only `code` set gives the code;
  - no errorInfo gives `unknown error`.

  Each case gives a `*DeliveryError` (`errors.As`), and `Permanent(err)` is true.
- `TestCreateDeliveryPolls`: create gives `InProgress` with `correlationId: "c1"`. Status answers `InProgress`, then
  `Success`. Expect the id from create, 2 status calls and a status body of `{organizationId, correlationId: "c1"}`.
- `TestCreateDeliveryPollError`: status gives `Error` with `errorReason: "Product not found"`. Expect a `*DeliveryError`
  with that reason.
- `TestCreateDeliveryPending`: `pollTimes = 3`, and status always gives `InProgress`. Expect `ErrDeliveryPending` and
  exactly 3 status calls.
- `TestCreateDeliveryUnknownStatus`: `creationStatus: "Weird"` gives an error that contains `Weird`.
- `TestCreateDeliveryBadRequest`: a 400 with `{"errorDescription": "Invalid phone"}` gives a `*StatusError` with
  `Status 400` and the message `iiko: 1/deliveries/create: unexpected status 400: Invalid phone`, and `Permanent` is
  true. A 500 without a body gives `Permanent` false. `Permanent(nil)` and `Permanent(errors.New("x"))` are false.
- `TestCreateDeliveryContextCanceled`: `pollEvery = time.Hour`, create gives `InProgress`, and the context is canceled
  right after the call starts (`context.WithTimeout` of 20 ms). Expect `errors.Is(err, context.DeadlineExceeded)`.
- `TestCreateDeliveryRefreshesToken`: the first create gives 401, then the token is fetched again and the second
  create succeeds (this checks that it goes through `withToken`).

## Done when
CONVENTIONS "Done means" pass, and coverage of `internal/iiko` is at least 80%.

## Out of scope
- Building the order from our data, config, wiring and retries (task 62).
- Product sizes (`productSizeId`), modifiers, combos, `completeBefore`, discounts.
- Any other iiko endpoint: `terminal_groups`, `payment_types`, `deliveries/by_id`, ...
