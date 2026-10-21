// Package app wires stores, services, handlers and external clients into one HTTP router.
package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/drewp1ter/citypark-go/internal/order"
	"github.com/drewp1ter/citypark-go/internal/telegram"
)

// sendTimeout bounds one notification, counted from the moment the goroutine starts.
const sendTimeout = 30 * time.Second

// MessageSender posts a text message to a chat. *telegram.Client implements it.
type MessageSender interface {
	SendMessage(ctx context.Context, chatID, text string) error
}

// TelegramNotifier implements order.Notifier. Every method returns at once and sends in a goroutine.
type TelegramNotifier struct {
	sender        MessageSender
	chatCreated   string
	chatConfirmed string
	log           *slog.Logger
	wg            sync.WaitGroup
}

var _ order.Notifier = (*TelegramNotifier)(nil)

func NewTelegramNotifier(sender MessageSender, chatCreated, chatConfirmed string, log *slog.Logger) *TelegramNotifier {
	return &TelegramNotifier{sender: sender, chatCreated: chatCreated, chatConfirmed: chatConfirmed, log: log}
}

// Wait blocks until all sends that were started have finished. Used on shutdown and in tests.
func (n *TelegramNotifier) Wait() { n.wg.Wait() }

// OrderCreated reports a cash order. Online orders are announced later, when they are paid.
func (n *TelegramNotifier) OrderCreated(ctx context.Context, o order.Order) {
	if o.PaymentType != order.PaymentCash {
		return
	}
	n.send(ctx, o, n.chatCreated)
}

func (n *TelegramNotifier) OrderPaid(ctx context.Context, o order.Order) {
	n.send(ctx, o, n.chatConfirmed)
}

func (n *TelegramNotifier) OrderConfirmed(ctx context.Context, o order.Order) {
	n.send(ctx, o, n.chatConfirmed)
}

// send delivers the formatted order to chatID and then the customer's phone to the "created" chat.
// The phone goes second as its own message so staff can copy it.
func (n *TelegramNotifier) send(ctx context.Context, o order.Order, chatID string) {
	if chatID == "" {
		n.log.Warn("telegram chat id is empty, order notification skipped", "order_id", o.ID)
		return
	}
	text := telegram.FormatOrder(orderMessage(o))

	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		// The request that caused the notification may finish before the send does.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sendTimeout)
		defer cancel()

		if err := n.sender.SendMessage(ctx, chatID, text); err != nil {
			n.log.Error("telegram send failed", "order_id", o.ID, "error", err)
			return
		}
		if err := n.sender.SendMessage(ctx, n.chatCreated, o.Customer.Phone); err != nil {
			n.log.Error("telegram send failed", "order_id", o.ID, "error", err)
		}
	}()
}

func orderMessage(o order.Order) telegram.OrderMessage {
	m := telegram.OrderMessage{
		ID:          o.ID,
		Name:        o.Customer.Name,
		Phone:       o.Customer.Phone,
		PaymentType: o.PaymentType,
		Total:       o.Total,
		Delivery:    o.DeliveryPrice,
		Comment:     o.Comment,
		Items:       make([]telegram.OrderItem, 0, len(o.Items)),
	}
	if o.Address != nil {
		m.Address = o.Address.City + ", " + o.Address.Address
	}
	for _, it := range o.Items {
		m.Items = append(m.Items, telegram.OrderItem{Category: it.CategoryName, Name: it.Name, Amount: it.Amount})
	}
	return m
}
