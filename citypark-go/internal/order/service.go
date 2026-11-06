package order

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/deliveryprice"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/moneta"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/promo"
	"github.com/drewp1ter/citypark-go/internal/salewindow"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

// Notifier tells staff about orders. Implementations must not block for long and must not fail the request.
type Notifier interface {
	OrderPending(ctx context.Context, o Order) // an order was created and waits for payment
	OrderPaid(ctx context.Context, o Order)    // an order was paid
}

type PaymentRequest struct {
	SubscriberID string // authenticated account identity; empty for guests

	TransactionID string // the order's PaymentUUID
	Amount        money.Kopecks
}

// PaymentForm is what the customer needs to pay an online order.
type PaymentForm struct {
	Amount    string // rubles with two decimals, "790.00"
	Signature string
	URL       string // full MONETA.Assistant address
}

// PaymentCallback is what Moneta sends after a payment (used in task 30).
type PaymentCallback = moneta.Notification

// Payments signs payment forms and checks payment callbacks. It makes no network calls.
type Payments interface {
	Form(r PaymentRequest) PaymentForm
	Verify(c PaymentCallback) bool
	CheckResponse(transactionID, code, amount string) moneta.CheckResponse
}

type CreateInput struct {
	PromoCode            string         `json:"promoCode"`
	ExpectedPayableTotal *money.Kopecks `json:"expectedPayableTotal"`
	GuestCheckoutID      string         `json:"-"`
	Phone                string         `json:"phone" validate:"required,ruphone"`
	Name                 string         `json:"name" validate:"required,max=80"`
	Comment              string         `json:"comment" validate:"max=8192"`
	DeliveryTime         string         `json:"deliveryTime" validate:"omitempty,hhmm"` // "" = as soon as possible
	Address              *address.Input `json:"address" validate:"omitempty"`
	Items                []ItemInput    `json:"items" validate:"required,min=1,dive"`
}

// CreateResult: RedirectURL is the Moneta payment page.
type CreateResult struct {
	OrderID     int64  `json:"orderId"`
	RedirectURL string `json:"redirectUrl"`
}

type Service struct {
	paymentHours        HoursSource
	promoDiscountTypeID string
	db                  *sqlx.DB
	clock               clock.Clock
	notifier            Notifier
	payments            Payments
}

func NewService(db *sqlx.DB, c clock.Clock, n Notifier, p Payments) *Service {
	return &Service{db: db, clock: c, notifier: n, payments: p}
}

// Create places an order. in must already be validated.
func (s *Service) Create(ctx context.Context, in CreateInput) (CreateResult, error) {
	var scope DuplicateScope
	if who, ok := auth.FromContext(ctx); ok {
		u, err := user.NewStore(s.db).ByID(ctx, who.UserID)
		if errors.Is(err, store.ErrNotFound) {
			return CreateResult{}, httpx.Unauthorized()
		}
		if err != nil {
			return CreateResult{}, err
		}
		if u.Role != who.Role {
			return CreateResult{}, httpx.Unauthorized()
		}
		if who.Role == auth.RoleUser {
			scope.AccountID = &who.UserID
		}
	} else if in.GuestCheckoutID != "" {
		hash := sha256.Sum256([]byte(in.GuestCheckoutID))
		scope.GuestCheckoutHash = hex.EncodeToString(hash[:])
	}
	phone, ok := validate.NormalizePhone(in.Phone)
	if !ok {
		return CreateResult{}, httpx.BadRequest("Неверный номер телефона.")
	}
	name := strings.TrimSpace(in.Name)
	comment := strings.TrimSpace(in.Comment)
	counts := map[int64]int{}
	for _, it := range in.Items {
		if it.Amount <= 0 || counts[it.ProductID] > math.MaxInt-it.Amount {
			return CreateResult{}, httpx.BadRequest("Некорректное количество.")
		}
		counts[it.ProductID] += it.Amount
	}
	items := mergeItems(in.Items)
	hash := ItemsHash(items)

	if in.ExpectedPayableTotal != nil && *in.ExpectedPayableTotal <= 0 {
		return CreateResult{}, expectedTotalError()
	}
	code := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 32
		}
		return r
	}, strings.TrimSpace(in.PromoCode))
	if code != "" {
		valid := len(code) >= 3 && len(code) <= 32
		for _, c := range code {
			if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
				valid = false
			}
		}
		if !valid {
			return CreateResult{}, quoteCodeError("Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.")
		}
		if in.ExpectedPayableTotal == nil || *in.ExpectedPayableTotal <= 0 {
			return CreateResult{}, expectedTotalError()
		}
	}

	orders := NewStore(s.db)

	var (
		o     Order
		found bool
	)
	var dupAddr *Address
	if in.Address != nil {
		dupAddr = &Address{City: in.Address.City, Address: in.Address.Address}
	}
	id, dup, err := orders.FindOpenDuplicate(ctx, scope, name, phone, dupAddr, comment, in.DeliveryTime, hash, code)
	if err != nil {
		return CreateResult{}, err
	}
	if dup {
		ids := make([]int64, len(items))
		for i, it := range items {
			ids[i] = it.ProductID
		}
		products, err := catalog.NewStore(s.db).ProductsByIDs(ctx, ids)
		if err != nil {
			return CreateResult{}, err
		}
		for _, pr := range products {
			if err := checkProductBlock(pr); err != nil {
				return CreateResult{}, err
			}
		}
		o, err = orders.ByID(ctx, id)
		if err != nil {
			return CreateResult{}, err
		}
		found = true
	}

	if !found {
		id, err := s.save(ctx, saveParams{
			scope:        scope,
			name:         name,
			phone:        phone,
			comment:      comment,
			deliveryTime: in.DeliveryTime,
			address:      in.Address,
			items:        items,
			hash:         hash,
			promoCode:    code, expected: in.ExpectedPayableTotal,
		})
		if err != nil {
			return CreateResult{}, err
		}
		o, err = orders.ByID(ctx, id)
		if err != nil {
			return CreateResult{}, err
		}
	}

	if in.ExpectedPayableTotal != nil && *in.ExpectedPayableTotal != o.Total+o.DeliveryPrice {
		return CreateResult{}, changedTotalError()
	}
	form := s.payments.Form(PaymentRequest{TransactionID: o.PaymentUUID, Amount: o.Total + o.DeliveryPrice, SubscriberID: paymentSubscriberID(scope.AccountID)})
	if o.Status == StatusNew {
		if err := orders.SetStatus(ctx, o.ID, StatusPending); err != nil {
			return CreateResult{}, err
		}
		o.Status = StatusPending
		s.notifier.OrderPending(ctx, o)
	}
	return CreateResult{OrderID: o.ID, RedirectURL: form.URL}, nil
}

var transactionIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// parseTransactionID accepts only the canonical dashed UUID form (any letter case) and returns it lowercase.
// Old numeric transaction ids do not match.
func parseTransactionID(s string) (string, bool) {
	s = strings.ToLower(s)
	return s, transactionIDPattern.MatchString(s)
}

// PaymentConfirmed checks a payment callback and marks the order as paid.
func (s *Service) PaymentConfirmed(ctx context.Context, cb PaymentCallback) error {
	uuid, ok := parseTransactionID(cb.TransactionID)
	if !ok {
		return httpx.BadRequest("Некорректное уведомление")
	}
	// The signature is checked before the order is loaded, so an unsigned
	// request cannot learn which orders exist.
	if !s.payments.Verify(cb) {
		return httpx.NewError(http.StatusForbidden, "Неверная подпись")
	}

	orders := NewStore(s.db)
	o, err := orders.ByPaymentUUID(ctx, uuid)
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NewError(http.StatusNotFound, "Заказ не найден")
	}
	if err != nil {
		return err
	}

	expected := s.payments.Form(PaymentRequest{TransactionID: o.PaymentUUID, Amount: o.Total + o.DeliveryPrice}).Amount
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
	scope                              DuplicateScope
	name, phone, comment, deliveryTime string
	address                            *address.Input
	items                              []ItemInput
	hash                               string
	promoCode                          string
	expected                           *money.Kopecks
}

// save checks the products and writes the customer, the address and the order in one transaction.
func (s *Service) save(ctx context.Context, p saveParams) (int64, error) {
	var orderID int64
	err := store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		var campaign *promo.Campaign
		if p.promoCode != "" {
			if err := checkPromoEnabled(ctx, tx); err != nil {
				return err
			}
			if s.promoDiscountTypeID == "" {
				return httpx.NewError(503, "Заказы с промокодом временно недоступны.")
			}
			c, e := promo.NewStore(tx).ByCode(ctx, p.promoCode)
			if errors.Is(e, store.ErrNotFound) {
				return quoteCodeError("Промокод недействителен.")
			}
			if e != nil {
				return e
			}
			campaign = &c
		}

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
		var subtotal money.Kopecks
		lines := make([]promo.Line, 0, len(p.items))
		for _, it := range p.items {
			pr := byID[it.ProductID]
			if err := checkWindows(pr, now); err != nil {
				return err
			}
			insertItems = append(insertItems, InsertItem{ProductID: pr.ID, Amount: it.Amount, OfferedPrice: pr.Price})
			if it.Amount <= 0 || pr.Price < 0 || int64(pr.Price) > math.MaxInt64/int64(it.Amount) || subtotal > money.Kopecks(math.MaxInt64)-pr.Price.Mul(it.Amount) {
				return fmt.Errorf("order pricing overflow")
			}
			subtotal += pr.Price.Mul(it.Amount)
			lines = append(lines, promo.Line{ProductID: pr.ID, CategoryID: pr.CategoryID, Amount: it.Amount, UnitPrice: pr.Price})
		}

		var discount money.Kopecks
		if campaign != nil {
			pricing, e := promo.Calculate(*campaign, lines, now)
			if e != nil {
				var v *promo.ValidationError
				if errors.As(e, &v) {
					messages := map[string]string{"unavailable": "Промокод недействителен.", "no eligible items": "В корзине нет блюд для этого промокода.", "minimum": "Недостаточная сумма подходящих блюд для промокода."}
					return quoteCodeError(messages[v.Message])
				}
				return e
			}
			discount = pricing.Discount
			for i := range insertItems {
				it := &insertItems[i]
				it.Discount = pricing.LineDiscounts[it.ProductID]
				pr := byID[it.ProductID]
				it.PromoEligible = campaign.Target == "all" || slices.Contains(campaign.ProductIDs, pr.ID) || slices.Contains(campaign.CategoryIDs, pr.CategoryID)
			}
		}
		var dp deliveryprice.DeliveryPrice
		if p.address != nil {
			var e error
			dp, e = deliveryprice.NewStore(tx).ByName(ctx, p.address.City)
			if errors.Is(e, store.ErrNotFound) {
				return httpx.BadRequest("Указано не существующее имя населенного пункта.")
			}
			if e != nil {
				return e
			}
		}
		delivery := deliveryCost(dp, subtotal-discount)
		if subtotal-discount > money.Kopecks(math.MaxInt64)-delivery {
			return fmt.Errorf("order pricing overflow")
		}
		if p.expected != nil && *p.expected != subtotal-discount+delivery {
			return changedTotalError()
		}
		var campaignID *int64
		var campaignCode *string
		var percentage *int
		if campaign != nil {
			campaignID = &campaign.ID
			campaignCode = &campaign.Code
			percentage = &campaign.Percentage
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
			addressID = &a.ID
		}

		orderID, err = NewStore(tx).Insert(ctx, InsertParams{
			CustomerID:        customer.ID,
			AccountID:         p.scope.AccountID,
			GuestCheckoutHash: p.scope.GuestCheckoutHash,
			ContactName:       &p.name,
			ContactPhone:      &p.phone,
			AddressID:         addressID,
			Comment:           p.comment,
			DeliveryTime:      p.deliveryTime,
			ItemsHash:         p.hash,
			DeliveryPrice:     delivery, PromoCodeID: campaignID, PromoCode: campaignCode, PromoPercentage: percentage, Discount: discount,
			Items: insertItems,
		})
		return err
	})
	if err != nil {
		return 0, err
	}
	return orderID, nil
}

// deliveryCost is the price of the town, or 0 when the order reaches the town's free-delivery threshold.
func deliveryCost(dp deliveryprice.DeliveryPrice, subtotal money.Kopecks) money.Kopecks {
	if dp.FreeFrom > 0 && subtotal >= dp.FreeFrom {
		return 0
	}
	return dp.Price
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
func checkProductBlock(p catalog.Product) error {
	if p.IsOrderBlocked {
		return httpx.NewError(http.StatusServiceUnavailable, fmt.Sprintf("Заказ блюда «%s» временно недоступен.", p.Name))
	}
	return nil
}

func checkWindows(p catalog.Product, now time.Time) error {
	if err := checkProductBlock(p); err != nil {
		return err
	}

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

// Reasons a MONETA CHECK failed. They are logged and sent to Sentry as a tag, so keep them fixed.
const (
	CheckBadRequest      = "bad_request"            // the form is not valid or MNT_COMMAND is not CHECK; HTTP 400
	CheckBadSignature    = "bad_signature"          // HTTP 403
	CheckInternalError   = "internal_error"         // any other error from PaymentCheck; HTTP 500
	CheckInvalidID       = "invalid_transaction_id" // result 500: the id is not a canonical UUID
	CheckOrderNotFound   = "order_not_found"        // result 500: no order has this UUID
	CheckOrderCanceled   = "order_canceled"         // result 500
	CheckNotPayable      = "order_not_payable"      // result 500: the status does not allow payment
	CheckRetryNotAllowed = "retry_not_allowed"      // result 500: new or pending order that cannot be paid now
	CheckAmountMismatch  = "amount_mismatch"        // result 500: cb.Amount differs from the order amount
)

// CheckOutcome is the result of PaymentCheck. Reason is empty for result codes 100, 200 and 402.
type CheckOutcome struct {
	Response       moneta.CheckResponse
	Reason         string // one of the Check* constants, or ""
	OrderID        int64  // 0 when no order was found
	OrderStatus    string // "" when no order was found
	RetryReason    string // PaymentRetry.Reason when Reason is CheckRetryNotAllowed
	ExpectedAmount string // the amount we computed, "" when no order was found
}

// PaymentCheck reads persisted order state without confirming payment.
func (s *Service) PaymentCheck(ctx context.Context, cb PaymentCallback) (CheckOutcome, error) {
	if !s.payments.Verify(cb) {
		return CheckOutcome{}, httpx.NewError(http.StatusForbidden, "Неверная подпись")
	}
	code, amount := "500", ""
	out := CheckOutcome{Reason: CheckInvalidID}
	if uuid, ok := parseTransactionID(cb.TransactionID); ok {
		out.Reason = CheckOrderNotFound
		o, err := NewStore(s.db).ByPaymentUUID(ctx, uuid)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return CheckOutcome{}, err
		}
		if err == nil {
			amount = moneta.FormatAmount(o.Total + o.DeliveryPrice)
			out.OrderID, out.OrderStatus, out.ExpectedAmount = o.ID, o.Status, amount
			out.Reason = CheckNotPayable
			switch {
			case o.Status == StatusCanceled:
				out.Reason = CheckOrderCanceled
			case o.PaymentID != "":
				code, out.Reason = "200", ""
			case o.Status == StatusNew || o.Status == StatusPending:
				eligible, err := s.customerOrders(ctx, []Order{o})
				if err != nil {
					return CheckOutcome{}, err
				}
				if !eligible[0].PaymentRetry.Allowed {
					out.Reason, out.RetryReason = CheckRetryNotAllowed, eligible[0].PaymentRetry.Reason
					break
				}
				if cb.Amount == "" {
					code, out.Reason = "100", ""
				} else if cb.Amount == amount {
					code, out.Reason = "402", ""
				} else {
					out.Reason = CheckAmountMismatch
				}
			}
		}
	}
	out.Response = s.payments.CheckResponse(cb.TransactionID, code, amount)
	return out, nil
}

func expectedTotalError() error {
	e := httpx.NewError(400, validationMessage)
	e.Fields = map[string]string{"expectedPayableTotal": "Подтвердите итоговую сумму заказа."}
	return e
}
func changedTotalError() error {
	return httpx.NewError(409, "Сумма заказа изменилась. Проверьте итог и повторите оформление.")
}

// ConfigurePromo sets the deployment's RMS discount ID before serving requests.
// Empty disables new promo orders; saved pending retries still use their snapshots.
func (s *Service) ConfigurePromo(discountTypeID string) { s.promoDiscountTypeID = discountTypeID }

// paymentSubscriberID never uses recipient contact records as payment identity.
func paymentSubscriberID(accountID *int64) string {
	if accountID == nil {
		return ""
	}
	return fmt.Sprintf("citypark-user-%d", *accountID)
}
