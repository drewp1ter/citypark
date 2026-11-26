package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/httpx"
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
	PromoDiscountTypeID                             string
	TerminalGroupID, PaymentTypeID, PaymentTypeKind string
	City, Street, House                             string          // placeholder iiko address
	Delays                                          []time.Duration // waits before each retry
}

// IikoOrders creates deliveries from approved submissions.
type IikoOrders struct {
	creator   DeliveryCreator
	db        *sqlx.DB
	alerts    MessageSender
	alertChat string
	settings  IikoOrderSettings
	log       *slog.Logger
}

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

// Prepare snapshots the delivery and all generated identifiers before the job is committed.
func (n *IikoOrders) Prepare(o order.Order) ([]byte, error) {
	if !n.configured() {
		return nil, httpx.NewError(http.StatusServiceUnavailable, "Отправка в iiko недоступна.")
	}
	d, err := deliveryOrder(o, n.settings, newUUID())
	if err != nil {
		n.log.Error("prepare iiko delivery", "order_id", o.ID, "error", err)
		return nil, httpx.NewError(http.StatusConflict, "Не удалось подготовить заказ для iiko.")
	}
	return json.Marshal(d)
}

func (n *IikoOrders) submit(ctx context.Context, id int64, d iiko.DeliveryOrder) (string, string, error) {
	for i := 0; ; i++ {
		if i > 0 {
			timer := time.NewTimer(n.settings.Delays[i-1])
			select {
			case <-ctx.Done():
				timer.Stop()
				return "review", "", ctx.Err()
			case <-timer.C:
			}
		}
		res, err := n.attempt(ctx, d)
		if err == nil && res.ID != "" {
			return "created", res.ID, nil
		}
		if err == nil {
			err = errors.New("iiko returned no delivery ID")
		}
		n.log.Error("iiko delivery attempt failed", "order_id", id, "iiko_order_id", d.ID, "error", err)
		switch {
		case iiko.DeliveryRateLimited(err):
			if i == len(n.settings.Delays) {
				return "failed", "", err
			}
		case iiko.DeliveryRejected(err):
			return "failed", "", err
		default:
			return "review", "", err
		}
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

func (ns Notifiers) OrderPending(ctx context.Context, o order.Order) {
	for _, n := range ns {
		n.OrderPending(ctx, o)
	}
}

func (ns Notifiers) OrderPaid(ctx context.Context, o order.Order) {
	for _, n := range ns {
		n.OrderPaid(ctx, o)
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

	positions := []string{}
	total, discount := o.Total, o.Discount
	for _, it := range o.Items {
		if it.ExcludedFromIiko {
			total -= it.OfferedPrice.Mul(it.Amount) - it.Discount
			discount -= it.Discount
		}
	}
	for _, it := range o.Items {
		if it.ExcludedFromIiko {
			continue
		}
		if it.IikoID == "" {
			return iiko.DeliveryOrder{}, fmt.Errorf("product %d has no iiko id", it.ProductID)
		}
		d.Items = append(d.Items, iiko.DeliveryItem{
			Comment: it.IikoComment, PositionID: newUUID(), Type: "Product", ProductID: it.IikoID, Amount: float64(it.Amount), Price: it.OfferedPrice.Rubles(),
		})
		if it.PromoEligible {
			positions = append(positions, d.Items[len(d.Items)-1].PositionID)
		}
	}
	if discount > 0 {
		if o.PromoCode == nil || s.PromoDiscountTypeID == "" {
			return iiko.DeliveryOrder{}, fmt.Errorf("promo RMS discount is not configured")
		}
		if len(positions) == 0 {
			return iiko.DeliveryOrder{}, fmt.Errorf("promo order has no saved eligible positions")
		}
		d.DiscountsInfo = &iiko.DeliveryDiscountsInfo{Discounts: []iiko.DeliveryRMSDiscount{{Type: "RMS", DiscountTypeID: s.PromoDiscountTypeID, Sum: discount.Rubles(), SelectivePositions: positions}}}
	}
	d.Payments = []iiko.DeliveryPayment{{
		PaymentTypeKind:       s.PaymentTypeKind,
		Sum:                   total.Rubles(),
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
