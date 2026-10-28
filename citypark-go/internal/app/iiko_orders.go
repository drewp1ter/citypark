package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/iiko"
	"github.com/drewp1ter/citypark-go/internal/order"
)

// createTimeout bounds one CreateDelivery attempt, including iiko's own status checks.
const createTimeout = 2 * time.Minute

// DeliveryCreator sends a delivery to iiko. *iiko.Client implements it.
type DeliveryCreator interface {
	CreateDelivery(ctx context.Context, terminalGroupID string, o iiko.DeliveryOrder) (iiko.DeliveryResult, error)
}

type IikoOrderSettings struct {
	TerminalGroupID, PaymentTypeID, PaymentTypeKind string
	City, Street, House                             string          // placeholder iiko address
	Delays                                          []time.Duration // waits before each retry
}

// IikoOrders implements order.Notifier: a paid order is created in iiko in a goroutine.
type IikoOrders struct {
	creator   DeliveryCreator
	db        *sqlx.DB
	alerts    MessageSender
	alertChat string
	settings  IikoOrderSettings
	log       *slog.Logger
	wg        sync.WaitGroup
}

var _ order.Notifier = (*IikoOrders)(nil)

func NewIikoOrders(c DeliveryCreator, db *sqlx.DB, alerts MessageSender, alertChat string, s IikoOrderSettings, log *slog.Logger) *IikoOrders {
	n := &IikoOrders{creator: c, db: db, alerts: alerts, alertChat: alertChat, settings: s, log: log}
	if !n.configured() {
		log.Warn("iiko orders are not configured")
	}
	return n
}

func (n *IikoOrders) configured() bool {
	return n.settings.TerminalGroupID != "" && n.settings.PaymentTypeID != ""
}

// Wait blocks until all started sends have finished. Used on shutdown and in tests.
func (n *IikoOrders) Wait() { n.wg.Wait() }

// OrderCreated does nothing: cash orders are not sent to iiko.
func (n *IikoOrders) OrderCreated(ctx context.Context, o order.Order) {}

func (n *IikoOrders) OrderConfirmed(ctx context.Context, o order.Order) {}

func (n *IikoOrders) OrderPaid(ctx context.Context, o order.Order) {
	if !n.configured() {
		return
	}
	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		n.send(context.WithoutCancel(ctx), o)
	}()
}

// send creates the order in iiko. The iiko order id is made once, so every retry sends the same
// order and iiko cannot create a second one.
func (n *IikoOrders) send(ctx context.Context, o order.Order) {
	d, err := deliveryOrder(o, n.settings, newUUID())
	if err != nil {
		n.fail(ctx, o, err)
		return
	}

	for i := 0; i <= len(n.settings.Delays); i++ {
		if i > 0 {
			time.Sleep(n.settings.Delays[i-1])
		}
		res, err := n.attempt(ctx, d)
		switch {
		case err == nil:
			if err := order.NewStore(n.db).SetIikoOrderID(ctx, o.ID, res.ID); err != nil {
				n.log.Error("save iiko order id failed", "order_id", o.ID, "iiko_id", res.ID, "error", err)
			}
			n.log.Info("order created in iiko", "order_id", o.ID, "iiko_id", res.ID)
			return
		case errors.Is(err, iiko.ErrDeliveryPending):
			// iiko may still create it, so a retry could make a duplicate.
			n.alert(ctx, o, fmt.Sprintf("Заказ №%d отправлен в iiko, но создание не подтверждено. Проверьте его в iikoFront.", o.ID))
			return
		case iiko.Permanent(err) || i == len(n.settings.Delays):
			n.fail(ctx, o, err)
			return
		}
		n.log.Warn("iiko delivery failed, will retry", "order_id", o.ID, "attempt", i+1, "error", err)
	}
}

func (n *IikoOrders) attempt(ctx context.Context, d iiko.DeliveryOrder) (iiko.DeliveryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	return n.creator.CreateDelivery(ctx, n.settings.TerminalGroupID, d)
}

func (n *IikoOrders) fail(ctx context.Context, o order.Order, err error) {
	n.alert(ctx, o, fmt.Sprintf("Заказ №%d не создан в iiko: %v. Внесите его в iikoFront вручную.", o.ID, err))
}

// alert tells staff to act. A failed send is only logged: there is nobody left to tell.
func (n *IikoOrders) alert(ctx context.Context, o order.Order, text string) {
	n.log.Error("order was not created in iiko", "order_id", o.ID, "alert", text)
	if n.alertChat == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	if err := n.alerts.SendMessage(ctx, n.alertChat, text); err != nil {
		n.log.Error("telegram send failed", "order_id", o.ID, "error", err)
	}
}

// Notifiers sends each event to every notifier, in order.
type Notifiers []order.Notifier

var _ order.Notifier = Notifiers(nil)

func (ns Notifiers) OrderCreated(ctx context.Context, o order.Order) {
	for _, n := range ns {
		n.OrderCreated(ctx, o)
	}
}

func (ns Notifiers) OrderPaid(ctx context.Context, o order.Order) {
	for _, n := range ns {
		n.OrderPaid(ctx, o)
	}
}

func (ns Notifiers) OrderConfirmed(ctx context.Context, o order.Order) {
	for _, n := range ns {
		n.OrderConfirmed(ctx, o)
	}
}

// deliveryOrder builds the iiko order. id is the iiko order GUID.
func deliveryOrder(o order.Order, s IikoOrderSettings, id string) (iiko.DeliveryOrder, error) {
	d := iiko.DeliveryOrder{
		ID:             id,
		ExternalNumber: strconv.FormatInt(o.ID, 10),
		Phone:          o.Customer.Phone,
		Customer:       iiko.DeliveryCustomer{Type: "one-time", Name: o.Customer.Name},
		Items:          make([]iiko.DeliveryItem, 0, len(o.Items)),
	}

	lines := []string{fmt.Sprintf("Заказ с сайта №%d, оплачен онлайн", o.ID)}
	if o.DeliveryTime != "" {
		lines = append(lines, "Ко времени: "+o.DeliveryTime)
	}
	if o.Address == nil {
		d.OrderServiceType = iiko.ServiceDeliveryByClient
	} else {
		full := o.Address.City + ", " + o.Address.Address
		d.OrderServiceType = iiko.ServiceDeliveryByCourier
		d.DeliveryPoint = &iiko.DeliveryPoint{
			Address: iiko.DeliveryAddress{Type: "legacy", Street: iiko.DeliveryStreet{Name: s.Street, City: s.City}, House: s.House},
			Comment: full,
		}
		lines = append(lines, "Адрес: "+full)
	}
	if o.DeliveryPrice > 0 {
		lines = append(lines, "Доставка: "+o.DeliveryPrice.String()+" ₽ (оплачена)")
	}
	if o.Comment != "" {
		lines = append(lines, "Комментарий: "+o.Comment)
	}
	d.Comment = strings.Join(lines, "\n")

	for _, it := range o.Items {
		if it.IikoID == "" {
			return iiko.DeliveryOrder{}, fmt.Errorf("product %d has no iiko id", it.ProductID)
		}
		d.Items = append(d.Items, iiko.DeliveryItem{
			Type: "Product", ProductID: it.IikoID, Amount: float64(it.Amount), Price: it.OfferedPrice.Rubles(),
		})
	}
	d.Payments = []iiko.DeliveryPayment{{
		PaymentTypeKind:       s.PaymentTypeKind,
		Sum:                   o.Total.Rubles(),
		PaymentTypeID:         s.PaymentTypeID,
		IsProcessedExternally: true,
	}}
	return d, nil
}

// newUUID returns a random version 4 UUID made with crypto/rand.
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // never fails since Go 1.24
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
