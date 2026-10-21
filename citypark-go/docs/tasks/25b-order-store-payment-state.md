# Task 25b: Order store — payment state and duplicate lookup

## Goal
Add the store methods that the payment flow needs: find an open duplicate, set the status, mark an order as paid.

## Depends on
25.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/types.go`, `internal/order/store.go`, `internal/order/store_test.go` (for the `seed` and `insertOrder` helpers).

## Files to change
- `internal/order/store.go`

## Files to create
- `internal/order/store_payment_test.go`

## Spec

```go
// FindOpenDuplicate looks for an online order with status new or pending and the same
// customer phone, address (nil = no address), comment and items hash.
// Only City and Address of addr are used.
func (s *Store) FindOpenDuplicate(ctx context.Context, phone string, addr *Address, comment, itemsHash string) (id int64, found bool, err error)

// SetStatus changes the status.
func (s *Store) SetStatus(ctx context.Context, id int64, status string) error

// MarkPaid sets status "paid" and stores the payment id (Moneta's operation id),
// but only when the order is still new or pending.
// changed is false when no row was updated: the order is unknown or was already handled.
func (s *Store) MarkPaid(ctx context.Context, id int64, paymentID string) (changed bool, err error)
```

Rules:
- `SetStatus` returns `store.ErrNotFound` when no row was updated (`RowsAffected() == 0`). It sets `updated_at = now()`.
- An invalid status reaches the database and fails its `CHECK`; return that error unchanged.

`MarkPaid` SQL:
```sql
UPDATE orders
SET status = 'paid', payment_id = $2, updated_at = now()
WHERE id = $1 AND status IN ('new', 'pending')
```
`changed` is `RowsAffected() == 1`. An unknown id is **not** an error here: it gives `false, nil`.

Why one statement: Moneta may send the same payment callback twice, even at the same moment. The `WHERE` lets the database decide which call "wins", so exactly one caller sees `changed == true` and staff are notified once.

`FindOpenDuplicate` SQL:
```sql
SELECT o.id
FROM orders o
JOIN users u ON u.id = o.customer_id
LEFT JOIN addresses a ON a.id = o.address_id
WHERE o.payment_type = 'online'
  AND o.status IN ('new', 'pending')
  AND u.phone = $1
  AND o.comment = $2
  AND o.items_hash = $3
  AND <address condition>
ORDER BY o.id DESC
LIMIT 1
```
Address condition:
- `addr == nil`: `o.address_id IS NULL`
- otherwise: `a.city = $4 AND a.address = $5`

Build the two variants as two complete query strings; do not concatenate user values into SQL. "No rows" means `found == false` with a nil error.

Why this exists: a customer who presses "pay" twice must get the same order back, not a second one.

## Tests to write
`db := testutil.NewDB(t)`; reuse `seed` and `insertOrder` from `store_test.go`.

- `TestSetStatus`: to `pending`; `updated_at` is not before `created_at`.
- `TestSetStatusUnknownOrder`: `store.ErrNotFound`.
- `TestSetStatusInvalid`: status `bogus` → `store.IsCheckViolation`.
- `TestMarkPaid`: table test on the order's starting status, with payment id `555`.

  | Starting status | `changed` | Status after | `payment_id` after |
  |---|---|---|---|
  | `new` | true | `paid` | `555` |
  | `pending` | true | `paid` | `555` |
  | `paid` (payment id `111`) | false | `paid` | `111` |
  | `canceled` | false | `canceled` | empty |
  | `confirmed` | false | `confirmed` | empty |

- `TestMarkPaidTwice`: the second call returns `false, nil` and keeps the first payment id.
- `TestMarkPaidUnknownOrder`: `false, nil`.
- `TestFindOpenDuplicate`: table test. Base order: online, status `new`, phone P, address A, comment C, hash H.

| Case | Found |
|---|---|
| same P, A, C, H | yes |
| base order has status `pending` | yes |
| base order has status `paid` | no |
| base order is a cash order | no |
| another comment | no |
| another hash | no |
| another phone | no |
| `addr == nil`, base order has an address | no |
| `addr` given, base order has no address | no |
| `addr == nil`, base order has no address | yes |
| two matching orders | yes, the higher id |

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
The service logic that calls these methods (tasks 26 and 30).
