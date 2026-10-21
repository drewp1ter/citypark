# Task 29: Order admin endpoints

## Goal
Admin routes to change an order: add or change an item, remove an item, delete the order, confirm the order.

## Depends on
28.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/store.go`, `internal/order/handler.go`, `internal/order/service.go`.

## Files to change
- `internal/order/store.go`: add `UpsertItem`, `RemoveItem`, `Delete`.
- `internal/order/service.go`: add `Confirm`.
- `internal/order/handler.go`: add four routes inside the existing admin group.

## Files to create
- `internal/order/admin_handler_test.go`

## Spec

### Store
```go
// UpsertItem adds a product to the order. If the order already has the product,
// only its amount is replaced (the stored offered price is kept).
func (s *Store) UpsertItem(ctx context.Context, orderID int64, it InsertItem) error

// RemoveItem deletes one item. No error when it does not exist.
func (s *Store) RemoveItem(ctx context.Context, orderID, productID int64) error

// Delete removes the order; its items go with it (ON DELETE CASCADE). No error when it does not exist.
func (s *Store) Delete(ctx context.Context, id int64) error
```
`UpsertItem` SQL:
```sql
INSERT INTO order_items (order_id, product_id, amount, offered_price_kopecks) VALUES ($1, $2, $3, $4)
ON CONFLICT (order_id, product_id) DO UPDATE SET amount = EXCLUDED.amount
```

### Service
```go
// Confirm sets status "confirmed" and notifies staff. Returns the updated order.
func (s *Service) Confirm(ctx context.Context, id int64) (Order, error)
```
1. `SetStatus(id, StatusConfirmed)`; `store.ErrNotFound` → `httpx.NotFound()`.
2. `ByID(id)`.
3. `s.notifier.OrderConfirmed(ctx, order)`.
4. Return the order.

### Routes (all admin)
| Route | Behaviour |
|---|---|
| `POST /orders/{id}/items` | body `ItemInput` (`{"productId": 1, "amount": 2}`), validated. Order missing → 404. Product missing or deleted (`catalog.NewStore(db).ProductsByIDs`) → `httpx.BadRequest("Продукт не найден.")`. `UpsertItem` with the product's current price. 200 with the updated `Order`. |
| `DELETE /orders/{id}/items/{productId}` | `RemoveItem`; always 204 |
| `DELETE /orders/{id}` | `Delete`; always 204 |
| `POST /orders/{id}/confirm` | `svc.Confirm`; 200 with the `Order` |

Use `httpx.PathID` for `id` and `productId`.

Deleting an order must not delete its customer or its address.

## Tests to write
`db := testutil.NewDB(t)`. Seed as in task 28: an admin, a user, a category with three products, one order with one item (product 1, amount 1). Fakes for `Notifier` and `Payments` from `service_test.go`.

- `TestAddItem`: admin posts `{"productId": 2, "amount": 3}` → 200; the body is the order with two items; `order_items` has two rows; the new item's `offeredPrice` is the product's price; `total` is updated.
- `TestAddItemExisting`: admin posts product 1 with amount 5 → 200; still one row for product 1, amount 5; its offered price is unchanged even if the product's price was changed with plain SQL before the request.
- `TestAddItemUnknownProduct`: `productId: 999` → 400 `Продукт не найден.`
- `TestAddItemUnknownOrder`: `/orders/999/items` → 404.
- `TestAddItemValidation`: `{"productId": 2, "amount": 0}` → 400 with `errors.amount`.
- `TestAddItemForbidden`: user token → 403. `TestAddItemUnauthorized`: 401.
- `TestRemoveItem`: 204; the row is gone. Removing again → 204.
- `TestRemoveItemForbidden`: user token → 403 and the row is still there.
- `TestDeleteOrder`: 204; no order row, no item rows; the counts of `users` and `addresses` are unchanged. Deleting again → 204.
- `TestDeleteOrderForbidden`: user token → 403 and the order is still there.
- `TestConfirm`: 200; body `status` is `confirmed`; the database row has status `confirmed`; the fake notifier got `OrderConfirmed` once with that order id.
- `TestConfirmMissing`: 404 and the notifier was not called.
- `TestConfirmForbidden`: user token → 403, status unchanged.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Changing an order's address, customer or payment type. Recomputing `items_hash` after item changes (it is only used when the order is created).
