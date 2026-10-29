package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
)

// selectOrders is the base SELECT for orders with customer and address.
// Callers append a WHERE clause.
const selectOrders = `
SELECT o.id, o.created_at, o.updated_at, o.status, o.comment, o.delivery_time, o.items_hash, o.payment_id,
       o.delivery_price_kopecks, o.iiko_order_id, o.account_id,
       u.id AS customer_id, COALESCE(o.contact_name, u.name) AS customer_name,
       COALESCE(o.contact_phone, u.phone) AS customer_phone,
       o.address_id, COALESCE(o.delivery_city, a.city) AS address_city,
       COALESCE(o.delivery_address, a.address) AS address_text
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
	CustomerID        int64
	AccountID         *int64
	GuestCheckoutHash string
	ContactName       *string
	ContactPhone      *string
	AddressID         *int64
	Comment           string
	DeliveryTime      string
	ItemsHash         string
	DeliveryPrice     money.Kopecks
	Items             []InsertItem
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// orderRow is one flat row of selectOrders.
type orderRow struct {
	ID            int64     `db:"id"`
	AccountID     *int64    `db:"account_id"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	Status        string    `db:"status"`
	Comment       string    `db:"comment"`
	DeliveryTime  string    `db:"delivery_time"`
	ItemsHash     string    `db:"items_hash"`
	PaymentID     string    `db:"payment_id"`
	DeliveryPrice int64     `db:"delivery_price_kopecks"`
	IikoOrderID   string    `db:"iiko_order_id"`
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
		AccountID:     r.AccountID,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		Status:        r.Status,
		Comment:       r.Comment,
		DeliveryTime:  r.DeliveryTime,
		DeliveryPrice: money.Kopecks(r.DeliveryPrice),
		Customer:      Customer{ID: r.CustomerID, Name: r.CustomerName, Phone: r.CustomerPhone},
		Items:         []Item{},
		ItemsHash:     r.ItemsHash,
		PaymentID:     r.PaymentID,
		IikoOrderID:   r.IikoOrderID,
	}
	if r.AddressCity != nil && r.AddressText != nil {
		o.Address = &Address{City: *r.AddressCity, Address: *r.AddressText}
		if r.AddressID != nil {
			o.Address.ID = *r.AddressID
		}
	}
	return o
}

// Insert writes the order row and its items and returns the order id.
// Call it with a store built from a transaction.
func (s *Store) Insert(ctx context.Context, p InsertParams) (int64, error) {
	var id int64
	err := s.db.GetContext(ctx, &id,
		`INSERT INTO orders (customer_id, address_id, comment, delivery_time, items_hash, delivery_price_kopecks,
           account_id, guest_checkout_hash, contact_name, contact_phone, delivery_city, delivery_address)
         VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''),
                COALESCE($9, (SELECT name FROM users WHERE id = $1)),
                COALESCE($10, (SELECT phone FROM users WHERE id = $1)),
                (SELECT city FROM addresses WHERE id = $2),
                (SELECT address FROM addresses WHERE id = $2)) RETURNING id`,
		p.CustomerID, p.AddressID, p.Comment, p.DeliveryTime, p.ItemsHash, int64(p.DeliveryPrice),
		p.AccountID, p.GuestCheckoutHash, p.ContactName, p.ContactPhone)
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

// ListFilter narrows List. Zero values mean "no condition".
type ListFilter struct {
	Query  string     // trimmed; matched against id, customer phone and customer name
	From   *time.Time // inclusive: o.created_at >= From
	Before *time.Time // exclusive: o.created_at < Before
	Status string     // one of the Status* constants
}

var digitsOnly = regexp.MustCompile(`^\d+$`)

// likeEscaper escapes the LIKE wildcards so that user text is matched literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// where builds the WHERE clause (with a trailing space, or empty) and its arguments.
func (f ListFilter) where() (string, []any) {
	var conds []string
	var args []any
	add := func(cond string, arg any) {
		args = append(args, arg)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if q := strings.TrimSpace(f.Query); q != "" {
		if digitsOnly.MatchString(q) && len(q) <= 18 {
			id, _ := strconv.ParseInt(q, 10, 64) // 18 digits always fit in int64
			args = append(args, id, "%"+q+"%")
			conds = append(conds, fmt.Sprintf("(o.id = $%d OR COALESCE(o.contact_phone, u.phone) LIKE $%d)", len(args)-1, len(args)))
		} else {
			add("(COALESCE(o.contact_name, u.name) ILIKE $%[1]d OR COALESCE(o.contact_phone, u.phone) LIKE $%[1]d)", "%"+likeEscaper.Replace(q)+"%")
		}
	}
	if f.From != nil {
		add("o.created_at >= $%d", *f.From)
	}
	if f.Before != nil {
		add("o.created_at < $%d", *f.Before)
	}
	if f.Status != "" {
		add("o.status = $%d", f.Status)
	}
	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND ") + " ", args
}

// List returns the orders that match f, newest first, and how many match in total.
func (s *Store) List(ctx context.Context, f ListFilter, limit, offset int) ([]Order, int, error) {
	where, args := f.where()
	var total int
	err := s.db.GetContext(ctx, &total,
		`SELECT count(*) FROM orders o JOIN users u ON u.id = o.customer_id `+where, args...)
	if err != nil {
		return nil, 0, err
	}
	var rows []orderRow
	err = s.db.SelectContext(ctx, &rows,
		selectOrders+where+fmt.Sprintf(`ORDER BY o.created_at DESC, o.id DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2),
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	orders, err := s.withItems(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListByAccount returns one account's orders, newest first, and the total count.
func (s *Store) ListByAccount(ctx context.Context, accountID int64, limit, offset int) ([]Order, int, error) {
	var total int
	err := s.db.GetContext(ctx, &total, `SELECT count(*) FROM orders WHERE account_id = $1`, accountID)
	if err != nil {
		return nil, 0, err
	}
	var rows []orderRow
	err = s.db.SelectContext(ctx, &rows,
		selectOrders+`WHERE o.account_id = $1 ORDER BY o.id DESC LIMIT $2 OFFSET $3`,
		accountID, limit, offset)
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

// DuplicateScope identifies the authenticated owner or a server-issued guest checkout.
type DuplicateScope struct {
	AccountID         *int64
	GuestCheckoutHash string
}

// FindOpenDuplicate never matches by phone alone or an empty guest scope.
func (s *Store) FindOpenDuplicate(ctx context.Context, scope DuplicateScope, name, phone string, addr *Address, comment, deliveryTime, itemsHash string) (id int64, found bool, err error) {
	var city, addressText *string
	if addr != nil {
		city, addressText = &addr.City, &addr.Address
	}
	err = s.db.GetContext(ctx, &id, `SELECT o.id FROM orders o
 WHERE o.status IN ('new', 'pending')
 AND (($1::bigint IS NOT NULL AND o.account_id = $1)
      OR ($1::bigint IS NULL AND o.account_id IS NULL AND $2 <> '' AND o.guest_checkout_hash = $2))
 AND o.contact_name = $3 AND o.contact_phone = $4 AND o.comment = $5
 AND o.delivery_time = $6 AND o.items_hash = $7
 AND o.delivery_city IS NOT DISTINCT FROM $8::text
 AND o.delivery_address IS NOT DISTINCT FROM $9::text
 ORDER BY o.id DESC LIMIT 1`, scope.AccountID, scope.GuestCheckoutHash, name, phone, comment, deliveryTime, itemsHash, city, addressText)
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

// SetIikoOrderID saves the id of the order in iiko. It returns store.ErrNotFound when the order does not exist.
func (s *Store) SetIikoOrderID(ctx context.Context, id int64, iikoID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orders SET iiko_order_id = $2, updated_at = now() WHERE id = $1`, id, iikoID)
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
		`SELECT oi.order_id, oi.product_id, p.name, c.name AS category_name, oi.amount, oi.offered_price_kopecks,
		        COALESCE(p.iiko_id, '') AS iiko_id
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
