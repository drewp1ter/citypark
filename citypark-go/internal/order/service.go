package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/deliveryprice"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/salewindow"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

// Notifier tells staff about orders. Implementations must not block for long and must not fail the request.
type Notifier interface {
	OrderCreated(ctx context.Context, o Order)   // a cash order was placed
	OrderPaid(ctx context.Context, o Order)      // an online order was paid
	OrderConfirmed(ctx context.Context, o Order) // an admin confirmed an order
}

type PaymentRequest struct {
	OrderID int64
	Amount  money.Kopecks
}

// PaymentForm is what the frontend needs to open the Moneta payment form.
type PaymentForm struct {
	Amount    string // rubles with two decimals, "790.00"
	Signature string
}

// PaymentCallback is what Moneta sends after a payment (used in task 30).
type PaymentCallback struct {
	TransactionID string // our order id, as text
	OperationID   string // Moneta's id of the payment
	Amount        string
	Signature     string
}

// Payments signs payment forms and checks payment callbacks. It makes no network calls.
type Payments interface {
	Form(r PaymentRequest) PaymentForm
	Verify(c PaymentCallback) bool
}

type CreateInput struct {
	Phone       string         `json:"phone" validate:"required,ruphone"`
	Name        string         `json:"name" validate:"required,max=80"`
	Comment     string         `json:"comment" validate:"max=8192"`
	PaymentType string         `json:"paymentType" validate:"required,oneof=cash online"`
	Address     *address.Input `json:"address" validate:"omitempty"`
	Items       []ItemInput    `json:"items" validate:"required,min=1,dive"`
}

// CreateResult: a cash order fills RedirectURL; an online order fills Amount and MntSignature.
type CreateResult struct {
	OrderID      int64  `json:"orderId"`
	RedirectURL  string `json:"redirectUrl,omitempty"`
	Amount       string `json:"amount,omitempty"`
	MntSignature string `json:"mntSignature,omitempty"`
}

type Options struct {
	ThankYouCash string
}

type Service struct {
	db       *sqlx.DB
	clock    clock.Clock
	notifier Notifier
	payments Payments
	opts     Options
}

func NewService(db *sqlx.DB, c clock.Clock, n Notifier, p Payments, opts Options) *Service {
	return &Service{db: db, clock: c, notifier: n, payments: p, opts: opts}
}

// Create places an order. in must already be validated.
func (s *Service) Create(ctx context.Context, in CreateInput) (CreateResult, error) {
	phone, ok := validate.NormalizePhone(in.Phone)
	if !ok {
		return CreateResult{}, httpx.BadRequest("Неверный номер телефона.")
	}
	name := strings.TrimSpace(in.Name)
	comment := strings.TrimSpace(in.Comment)
	items := mergeItems(in.Items)
	hash := ItemsHash(items)

	var delivery money.Kopecks
	if in.Address != nil {
		dp, err := deliveryprice.NewStore(s.db).ByName(ctx, in.Address.City)
		if errors.Is(err, store.ErrNotFound) {
			return CreateResult{}, httpx.BadRequest("Указано не существующее имя населенного пункта.")
		}
		if err != nil {
			return CreateResult{}, err
		}
		delivery = dp.Price
	}

	orders := NewStore(s.db)

	var (
		o     Order
		found bool
	)
	if in.PaymentType == PaymentOnline {
		var dupAddr *Address
		if in.Address != nil {
			dupAddr = &Address{City: in.Address.City, Address: in.Address.Address}
		}
		id, dup, err := orders.FindOpenDuplicate(ctx, phone, dupAddr, comment, hash)
		if err != nil {
			return CreateResult{}, err
		}
		if dup {
			o, err = orders.ByID(ctx, id)
			if err != nil {
				return CreateResult{}, err
			}
			found = true
		}
	}

	if !found {
		id, err := s.save(ctx, saveParams{
			name:        name,
			phone:       phone,
			comment:     comment,
			paymentType: in.PaymentType,
			address:     in.Address,
			items:       items,
			hash:        hash,
			delivery:    delivery,
		})
		if err != nil {
			return CreateResult{}, err
		}
		o, err = orders.ByID(ctx, id)
		if err != nil {
			return CreateResult{}, err
		}
	}

	if in.PaymentType == PaymentCash {
		s.notifier.OrderCreated(ctx, o)
		return CreateResult{OrderID: o.ID, RedirectURL: s.opts.ThankYouCash}, nil
	}

	form := s.payments.Form(PaymentRequest{OrderID: o.ID, Amount: o.Total + o.DeliveryPrice})
	if o.Status == StatusNew {
		if err := orders.SetStatus(ctx, o.ID, StatusPending); err != nil {
			return CreateResult{}, err
		}
	}
	return CreateResult{OrderID: o.ID, Amount: form.Amount, MntSignature: form.Signature}, nil
}

// Confirm sets status "confirmed" and notifies staff. Returns the updated order.
func (s *Service) Confirm(ctx context.Context, id int64) (Order, error) {
	orders := NewStore(s.db)
	if err := orders.SetStatus(ctx, id, StatusConfirmed); errors.Is(err, store.ErrNotFound) {
		return Order{}, httpx.NotFound()
	} else if err != nil {
		return Order{}, err
	}
	o, err := orders.ByID(ctx, id)
	if err != nil {
		return Order{}, err
	}
	s.notifier.OrderConfirmed(ctx, o)
	return o, nil
}

// PaymentConfirmed checks a payment callback and marks the order as paid.
func (s *Service) PaymentConfirmed(ctx context.Context, cb PaymentCallback) error {
	id, err := strconv.ParseInt(cb.TransactionID, 10, 64)
	if err != nil || id < 1 {
		return httpx.BadRequest("Некорректное уведомление")
	}
	// The signature is checked before the order is loaded, so an unsigned
	// request cannot learn which orders exist.
	if !s.payments.Verify(cb) {
		return httpx.NewError(http.StatusForbidden, "Неверная подпись")
	}

	orders := NewStore(s.db)
	o, err := orders.ByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NewError(http.StatusNotFound, "Заказ не найден")
	}
	if err != nil {
		return err
	}

	expected := s.payments.Form(PaymentRequest{OrderID: o.ID, Amount: o.Total + o.DeliveryPrice}).Amount
	if expected != cb.Amount {
		slog.Default().Error("moneta callback amount mismatch",
			"order_id", o.ID, "expected", expected, "got", cb.Amount)
		return httpx.BadRequest("Неверная сумма")
	}

	changed, err := orders.MarkPaid(ctx, o.ID, cb.OperationID)
	if err != nil {
		return err
	}
	if !changed {
		// Moneta may deliver the same callback several times; staff are told once.
		if o.Status == StatusCanceled {
			slog.Default().Warn("canceled order was paid", "order_id", o.ID, "operation_id", cb.OperationID)
		}
		return nil
	}

	o, err = orders.ByID(ctx, o.ID)
	if err != nil {
		return err
	}
	s.notifier.OrderPaid(ctx, o)
	return nil
}

type saveParams struct {
	name, phone, comment, paymentType string
	address                           *address.Input
	items                             []ItemInput
	hash                              string
	delivery                          money.Kopecks
}

// save checks the products and writes the customer, the address and the order in one transaction.
func (s *Service) save(ctx context.Context, p saveParams) (int64, error) {
	var orderID int64
	err := store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		ids := make([]int64, len(p.items))
		for i, it := range p.items {
			ids[i] = it.ProductID
		}
		list, err := catalog.NewStore(tx).ProductsByIDs(ctx, ids)
		if err != nil {
			return err
		}
		if len(list) < len(ids) {
			return httpx.BadRequest("Продукт не найден.")
		}
		byID := make(map[int64]catalog.Product, len(list))
		for _, pr := range list {
			byID[pr.ID] = pr
		}

		now := s.clock.Now()
		insertItems := make([]InsertItem, 0, len(p.items))
		for _, it := range p.items {
			pr := byID[it.ProductID]
			if err := checkWindows(pr, now); err != nil {
				return err
			}
			insertItems = append(insertItems, InsertItem{ProductID: pr.ID, Amount: it.Amount, OfferedPrice: pr.Price})
		}

		customer, err := user.NewStore(tx).UpsertCustomer(ctx, p.name, p.phone)
		if err != nil {
			return err
		}

		var addressID *int64
		if p.address != nil {
			addrStore := address.NewStore(tx)
			a, err := addrStore.FindOrCreate(ctx, p.address.City, p.address.Address)
			if err != nil {
				return err
			}
			if err := addrStore.LinkUser(ctx, customer.ID, a.ID); err != nil {
				return err
			}
			addressID = &a.ID
		}

		orderID, err = NewStore(tx).Insert(ctx, InsertParams{
			CustomerID:    customer.ID,
			AddressID:     addressID,
			Comment:       p.comment,
			PaymentType:   p.paymentType,
			ItemsHash:     p.hash,
			DeliveryPrice: p.delivery,
			Items:         insertItems,
		})
		return err
	})
	if err != nil {
		return 0, err
	}
	return orderID, nil
}

// mergeItems adds up the amounts of repeated products, keeping the order of first appearance.
func mergeItems(in []ItemInput) []ItemInput {
	out := make([]ItemInput, 0, len(in))
	pos := make(map[int64]int, len(in))
	for _, it := range in {
		if i, ok := pos[it.ProductID]; ok {
			out[i].Amount += it.Amount
			continue
		}
		pos[it.ProductID] = len(out)
		out = append(out, it)
	}
	return out
}

// checkWindows returns a 503 when the product or its category is closed at now.
func checkWindows(p catalog.Product, now time.Time) error {
	if w := p.CategoryWindow(); !w.OpenOnDay(now) {
		return daysClosed(p.CategoryName, w)
	} else if !w.OpenAtTime(now) {
		return timeClosed(p.CategoryName, w)
	}
	if w := p.Window(); !w.OpenOnDay(now) {
		return daysClosed(p.Name, w)
	} else if !w.OpenAtTime(now) {
		return timeClosed(p.Name, w)
	}
	return nil
}

func daysClosed(name string, w salewindow.Window) *httpx.Error {
	return httpx.NewError(http.StatusServiceUnavailable,
		fmt.Sprintf(`Возможность заказа "%s" доступна только в следующие дни недели: %s`, name, w.DaysString()))
}

func timeClosed(name string, w salewindow.Window) *httpx.Error {
	return httpx.NewError(http.StatusServiceUnavailable,
		fmt.Sprintf(`Время заказа "%s" ограничено с %s до %s.`, name, w.StartString(), w.EndString()))
}
