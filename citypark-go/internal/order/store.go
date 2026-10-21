package order

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
)

// selectOrders is the base SELECT for orders with customer and address.
// Callers append a WHERE clause.
const selectOrders = `
SELECT o.id, o.created_at, o.updated_at, o.status, o.comment, o.payment_type, o.items_hash, o.payment_id,
       o.delivery_price_kopecks,
       u.id AS customer_id, u.name AS customer_name, u.phone AS customer_phone,
       a.id AS address_id, a.city AS address_city, a.address AS address_text
FROM orders o
JOIN users u ON u.id = o.customer_id
LEFT JOIN addresses a ON a.id = o.address_id
`

type InsertItem struct {
	ProductID    int64
	Amount       int
	OfferedPrice money.Kopecks
}

type InsertParams struct {
	CustomerID    int64
	AddressID     *int64
	Comment       string
	PaymentType   string
	ItemsHash     string
	DeliveryPrice money.Kopecks
	Items         []InsertItem
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// orderRow is one flat row of selectOrders.
type orderRow struct {
	ID            int64     `db:"id"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	Status        string    `db:"status"`
	Comment       string    `db:"comment"`
	PaymentType   string    `db:"payment_type"`
	ItemsHash     string    `db:"items_hash"`
	PaymentID     string    `db:"payment_id"`
	DeliveryPrice int64     `db:"delivery_price_kopecks"`
	CustomerID    int64     `db:"customer_id"`
	CustomerName  string    `db:"customer_name"`
	CustomerPhone string    `db:"customer_phone"`
	AddressID     *int64    `db:"address_id"`
	AddressCity   *string   `db:"address_city"`
	AddressText   *string   `db:"address_text"`
}

func (r orderRow) toOrder() Order {
	o := Order{
		ID:            r.ID,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		Status:        r.Status,
		Comment:       r.Comment,
		PaymentType:   r.PaymentType,
		DeliveryPrice: money.Kopecks(r.DeliveryPrice),
		Customer:      Customer{ID: r.CustomerID, Name: r.CustomerName, Phone: r.CustomerPhone},
		Items:         []Item{},
		ItemsHash:     r.ItemsHash,
		PaymentID:     r.PaymentID,
	}
	if r.AddressID != nil {
		// city and address are NOT NULL in addresses, so they are set whenever the id is.
		o.Address = &Address{ID: *r.AddressID, City: *r.AddressCity, Address: *r.AddressText}
	}
	return o
}

// Insert writes the order row and its items and returns the order id.
// Call it with a store built from a transaction.
func (s *Store) Insert(ctx context.Context, p InsertParams) (int64, error) {
	var id int64
	err := s.db.GetContext(ctx, &id,
		`INSERT INTO orders (customer_id, address_id, comment, payment_type, items_hash, delivery_price_kopecks)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		p.CustomerID, p.AddressID, p.Comment, p.PaymentType, p.ItemsHash, int64(p.DeliveryPrice))
	if err != nil {
		return 0, err
	}
	for _, it := range p.Items {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO order_items (order_id, product_id, amount, offered_price_kopecks)
			 VALUES ($1, $2, $3, $4)`,
			id, it.ProductID, it.Amount, int64(it.OfferedPrice))
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

// ByID returns store.ErrNotFound when missing.
func (s *Store) ByID(ctx context.Context, id int64) (Order, error) {
	var row orderRow
	if err := s.db.GetContext(ctx, &row, selectOrders+`WHERE o.id = $1`, id); err != nil {
		return Order{}, store.NotFound(err)
	}
	orders := []Order{row.toOrder()}
	if err := s.loadItems(ctx, orders); err != nil {
		return Order{}, err
	}
	return orders[0], nil
}

// List returns all orders, newest first, and the total count.
func (s *Store) List(ctx context.Context, limit, offset int) ([]Order, int, error) {
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT count(*) FROM orders`); err != nil {
		return nil, 0, err
	}
	var rows []orderRow
	err := s.db.SelectContext(ctx, &rows,
		selectOrders+`ORDER BY o.created_at DESC, o.id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	orders, err := s.withItems(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListByCustomer returns one customer's orders, newest first, and the total count.
func (s *Store) ListByCustomer(ctx context.Context, customerID int64, limit, offset int) ([]Order, int, error) {
	var total int
	err := s.db.GetContext(ctx, &total, `SELECT count(*) FROM orders WHERE customer_id = $1`, customerID)
	if err != nil {
		return nil, 0, err
	}
	var rows []orderRow
	err = s.db.SelectContext(ctx, &rows,
		selectOrders+`WHERE o.customer_id = $1 ORDER BY o.id DESC LIMIT $2 OFFSET $3`,
		customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	orders, err := s.withItems(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// withItems converts rows to orders and loads their items. The result is never nil.
func (s *Store) withItems(ctx context.Context, rows []orderRow) ([]Order, error) {
	orders := make([]Order, len(rows))
	for i, r := range rows {
		orders[i] = r.toOrder()
	}
	if len(orders) == 0 {
		return orders, nil
	}
	if err := s.loadItems(ctx, orders); err != nil {
		return nil, err
	}
	return orders, nil
}

const (
	selectDuplicatePickup = `
SELECT o.id
FROM orders o
JOIN users u ON u.id = o.customer_id
LEFT JOIN addresses a ON a.id = o.address_id
WHERE o.payment_type = 'online'
  AND o.status IN ('new', 'pending')
  AND u.phone = $1
  AND o.comment = $2
  AND o.items_hash = $3
  AND o.address_id IS NULL
ORDER BY o.id DESC
LIMIT 1`

	selectDuplicateDelivery = `
SELECT o.id
FROM orders o
JOIN users u ON u.id = o.customer_id
LEFT JOIN addresses a ON a.id = o.address_id
WHERE o.payment_type = 'online'
  AND o.status IN ('new', 'pending')
  AND u.phone = $1
  AND o.comment = $2
  AND o.items_hash = $3
  AND a.city = $4 AND a.address = $5
ORDER BY o.id DESC
LIMIT 1`
)

// FindOpenDuplicate looks for an online order with status new or pending and the same
// customer phone, address (nil = no address), comment and items hash.
// Only City and Address of addr are used.
func (s *Store) FindOpenDuplicate(ctx context.Context, phone string, addr *Address, comment, itemsHash string) (id int64, found bool, err error) {
	if addr == nil {
		err = s.db.GetContext(ctx, &id, selectDuplicatePickup, phone, comment, itemsHash)
	} else {
		err = s.db.GetContext(ctx, &id, selectDuplicateDelivery, phone, comment, itemsHash, addr.City, addr.Address)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// SetStatus changes the status. It returns store.ErrNotFound when the order does not exist.
func (s *Store) SetStatus(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

// MarkPaid sets status "paid" and stores the payment id (Moneta's operation id),
// but only when the order is still new or pending. A single UPDATE lets the database
// pick the winner when the same callback arrives twice at once.
// changed is false when no row was updated: the order is unknown or was already handled.
func (s *Store) MarkPaid(ctx context.Context, id int64, paymentID string) (changed bool, err error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders
		 SET status = 'paid', payment_id = $2, updated_at = now()
		 WHERE id = $1 AND status IN ('new', 'pending')`,
		id, paymentID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// UpsertItem adds a product to the order. If the order already has the product,
// only its amount is replaced (the stored offered price is kept).
func (s *Store) UpsertItem(ctx context.Context, orderID int64, it InsertItem) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO order_items (order_id, product_id, amount, offered_price_kopecks) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (order_id, product_id) DO UPDATE SET amount = EXCLUDED.amount`,
		orderID, it.ProductID, it.Amount, int64(it.OfferedPrice))
	return err
}

// RemoveItem deletes one item. No error when it does not exist.
func (s *Store) RemoveItem(ctx context.Context, orderID, productID int64) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM order_items WHERE order_id = $1 AND product_id = $2`, orderID, productID)
	return err
}

// Delete removes the order; its items go with it (ON DELETE CASCADE). No error when it does not exist.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM orders WHERE id = $1`, id)
	return err
}

// loadItems fills Items and Total for every order with a single query.
func (s *Store) loadItems(ctx context.Context, orders []Order) error {
	ids := make([]int64, len(orders))
	index := make(map[int64]int, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
		index[o.ID] = i
	}
	var items []Item
	err := s.db.SelectContext(ctx, &items,
		`SELECT oi.order_id, oi.product_id, p.name, c.name AS category_name, oi.amount, oi.offered_price_kopecks
		 FROM order_items oi
		 JOIN products p ON p.id = oi.product_id
		 JOIN categories c ON c.id = p.category_id
		 WHERE oi.order_id = ANY($1)
		 ORDER BY oi.order_id, oi.product_id`,
		ids)
	if err != nil {
		return err
	}
	for _, it := range items {
		o := &orders[index[it.OrderID]]
		o.Items = append(o.Items, it)
		o.Total += it.OfferedPrice.Mul(it.Amount)
	}
	return nil
}
