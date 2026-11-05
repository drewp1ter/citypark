package order

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
)

// PaymentRetryEligibility describes current payment availability, not persisted state.
type PaymentRetryEligibility struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

type customerOrder struct {
	Order
	PaymentRetry PaymentRetryEligibility `json:"paymentRetry"`
}

// SetPaymentHours configures payment checks before serving requests.
func (s *Service) SetPaymentHours(hours HoursSource) { s.paymentHours = hours }

func paymentState(o Order, now time.Time) PaymentRetryEligibility {
	if (o.Status != StatusNew && o.Status != StatusPending) || o.PaymentID != "" {
		return PaymentRetryEligibility{Reason: "not_payable"}
	}
	if o.CreatedAt.In(now.Location()).Format(time.DateOnly) != now.Format(time.DateOnly) {
		return PaymentRetryEligibility{Reason: "expired"}
	}
	return PaymentRetryEligibility{Allowed: true}
}

// customerOrders reads hours at most once, only when a same-day unpaid order needs them.
func (s *Service) customerOrders(ctx context.Context, orders []Order) ([]customerOrder, error) {
	now := s.clock.Now()
	result := make([]customerOrder, len(orders))
	loaded, open := false, false
	for i, o := range orders {
		eligibility := paymentState(o, now)
		if eligibility.Allowed {
			if !loaded {
				if s.paymentHours == nil {
					return nil, fmt.Errorf("payment hours not configured")
				}
				hours, _, err := s.paymentHours.Get(ctx)
				if err != nil {
					return nil, err
				}
				loaded, open = true, hours.OpenAt(now)
			}
			if !open {
				eligibility = PaymentRetryEligibility{Reason: "closed"}
			}
		}
		result[i] = customerOrder{Order: o, PaymentRetry: eligibility}
	}
	return result, nil
}

// RetryPayment signs the saved order without changing it or repeating notifications.
func (s *Service) RetryPayment(ctx context.Context, orderID, accountID int64) (CreateResult, error) {
	o, err := NewStore(s.db).ByID(ctx, orderID)
	if errors.Is(err, store.ErrNotFound) {
		return CreateResult{}, httpx.NotFound()
	}
	if err != nil {
		return CreateResult{}, err
	}
	if o.AccountID == nil || *o.AccountID != accountID {
		return CreateResult{}, httpx.NotFound()
	}
	orders, err := s.customerOrders(ctx, []Order{o})
	if err != nil {
		return CreateResult{}, err
	}
	switch orders[0].PaymentRetry.Reason {
	case "not_payable":
		return CreateResult{}, httpx.NewError(409, "Этот заказ уже оплачен или недоступен для оплаты.")
	case "expired":
		return CreateResult{}, httpx.NewError(409, "Срок оплаты заказа истёк. Оформите новый заказ.")
	case "closed":
		return CreateResult{}, httpx.NewError(503, "Оплата доступна только в часы работы ресторана.")
	}
	form := s.payments.Form(PaymentRequest{TransactionID: o.PaymentUUID, Amount: o.Total + o.DeliveryPrice})
	if form.URL == "" {
		return CreateResult{}, fmt.Errorf("empty payment URL")
	}
	return CreateResult{OrderID: o.ID, RedirectURL: form.URL}, nil
}

func (h *Handler) retryPayment(w http.ResponseWriter, r *http.Request) error {
	who, _ := auth.FromContext(r.Context())
	if who.Role != auth.RoleUser {
		return httpx.Forbidden()
	}
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	result, err := h.svc.RetryPayment(r.Context(), id, who.UserID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, result)
	return nil
}
