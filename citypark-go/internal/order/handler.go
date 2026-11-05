package order

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
	"github.com/drewp1ter/citypark-go/internal/workinghours"
)

const validationMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"

type HandlerOptions struct {
	Hours           HoursSource
	DefaultPageSize int
}

// Handler serves the order endpoints.
type Handler struct {
	svc      *Service
	tokens   *auth.Tokens // used by the read and admin routes (tasks 28-30)
	opts     HandlerOptions
	access   func(http.Handler) http.Handler
	optional func(http.Handler) http.Handler
}

func NewHandler(svc *Service, tokens *auth.Tokens, opts HandlerOptions, adapters ...func(http.Handler) http.Handler) *Handler {
	h := &Handler{svc: svc, tokens: tokens, opts: opts, access: tokens.RequireRole(auth.RoleUser), optional: tokens.Optional}
	if len(adapters) == 2 {
		h.access = adapters[0]
		h.optional = adapters[1]
	}
	return h
}

// Routes registers POST /orders, which is public and limited to the working hours,
// the read endpoints for users and admins, and the admin routes that change an order.
func (h *Handler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(privateOrders)
		h.routes(r)
	})
}

func (h *Handler) routes(r chi.Router) {
	r.With(h.optional).Post("/orders/quote", httpx.Handle(h.quote))
	r.With(h.optional, WorkingHours(h.svc.clock, h.opts.Hours)).
		Post("/orders", httpx.Handle(h.create))

	r.Post("/webhooks/moneta", h.moneta)
	r.Get("/webhooks/moneta", h.moneta)
	r.Post("/webhooks/moneta/check", h.monetaCheck)
	r.Get("/webhooks/moneta/check", h.monetaCheck)

	r.Group(func(r chi.Router) {
		r.Use(h.access)
		r.Get("/orders/my", httpx.Handle(h.listMine))
		r.Post("/orders/{id}/payment-retry", httpx.Handle(h.retryPayment))
		r.Get("/orders/{id}", httpx.Handle(h.get))
	})
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Get("/orders", httpx.Handle(h.list))
		r.Post("/orders/{id}/items", httpx.Handle(h.addItem))
		r.Delete("/orders/{id}/items/{productId}", httpx.Handle(h.removeItem))
		r.Delete("/orders/{id}", httpx.Handle(h.delete))
	})
}

// moneta answers the payment callback. Moneta expects the plain text SUCCESS;
// any other answer makes it repeat the call, so this route does not use httpx.Handle.
func (h *Handler) moneta(w http.ResponseWriter, r *http.Request) {
	cb, err := parseMoneta(w, r)
	if err != nil || cb.Command != "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "FAIL")
		return
	}

	slog.Default().Info("moneta callback", "order_id", cb.TransactionID, "operation_id", cb.OperationID)

	status, body := http.StatusOK, "SUCCESS"
	if err := h.svc.PaymentConfirmed(r.Context(), cb); err != nil {
		status, body = http.StatusInternalServerError, "FAIL"
		var he *httpx.Error
		if errors.As(err, &he) {
			status = he.Status
		} else {
			slog.Default().Error("moneta callback failed", "order_id", cb.TransactionID, "error", err)
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

type listResponse struct {
	Orders []Order `json:"orders"`
	Count  int     `json:"count"`
}

func (h *Handler) listMine(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := httpx.Pagination(r, h.opts.DefaultPageSize)
	if err != nil {
		return err
	}
	id, _ := auth.FromContext(r.Context())
	orders, count, err := NewStore(h.svc.db).ListByAccount(r.Context(), id.UserID, limit, offset)
	if err != nil {
		return err
	}
	customerOrders, err := h.svc.customerOrders(r.Context(), orders)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, struct {
		Orders []customerOrder `json:"orders"`
		Count  int             `json:"count"`
	}{Orders: customerOrders, Count: count})
	return nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := httpx.Pagination(r, h.opts.DefaultPageSize)
	if err != nil {
		return err
	}
	f, err := h.parseFilter(r)
	if err != nil {
		return err
	}
	orders, count, err := NewStore(h.svc.db).List(r.Context(), f, limit, offset)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, listResponse{Orders: orders, Count: count})
	return nil
}

// parseFilter reads the search parameters of GET /orders. An empty parameter counts as missing.
func (h *Handler) parseFilter(r *http.Request) (ListFilter, error) {
	var f ListFilter
	fields := map[string]string{}
	qs := r.URL.Query()

	f.Query = strings.TrimSpace(qs.Get("q"))
	if utf8.RuneCountInString(f.Query) > 100 {
		fields["q"] = "не более 100 символов"
	}

	loc := h.svc.clock.Now().Location()
	const dateErr = "дата в формате ГГГГ-ММ-ДД"
	var from time.Time
	if v := qs.Get("from"); v != "" {
		d, err := time.ParseInLocation("2006-01-02", v, loc)
		if err != nil {
			fields["from"] = dateErr
		} else {
			from = d
			f.From = &d
		}
	}
	if v := qs.Get("to"); v != "" {
		d, err := time.ParseInLocation("2006-01-02", v, loc)
		switch {
		case err != nil:
			fields["to"] = dateErr
		case f.From != nil && d.Before(from):
			fields["to"] = "не раньше даты from"
		default:
			before := d.AddDate(0, 0, 1)
			f.Before = &before
		}
	}

	if v := qs.Get("status"); v != "" {
		if !slices.Contains([]string{StatusNew, StatusPending, StatusPaid, StatusCanceled, StatusConfirmed}, v) {
			fields["status"] = "недопустимое значение"
		}
		f.Status = v
	}

	if len(fields) > 0 {
		e := httpx.BadRequest("Некорректные параметры поиска")
		e.Fields = fields
		return ListFilter{}, e
	}
	return f, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	o, err := NewStore(h.svc.db).ByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	who, _ := auth.FromContext(r.Context())
	if who.Role != auth.RoleAdmin && (o.AccountID == nil || *o.AccountID != who.UserID) {
		// do not reveal that someone else's order exists
		return httpx.NotFound()
	}
	if who.Role == auth.RoleAdmin {
		httpx.JSON(w, http.StatusOK, o)
	} else {
		orders, err := h.svc.customerOrders(r.Context(), []Order{o})
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, orders[0])
	}
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		CreateInput
		Expected json.RawMessage `json:"expectedPayableTotal"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	in := req.CreateInput
	if len(req.Expected) > 0 && !bytes.Equal(bytes.TrimSpace(req.Expected), []byte("null")) {
		var amount money.Kopecks
		if err := json.Unmarshal(req.Expected, &amount); err != nil {
			return expectedTotalError()
		}
		in.ExpectedPayableTotal = &amount
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	if in.DeliveryTime != "" {
		hours, _, err := h.opts.Hours.Get(r.Context())
		if err != nil {
			return err
		}
		if !hours.Contains(in.DeliveryTime) {
			e := httpx.NewError(http.StatusBadRequest, validationMessage)
			e.Fields = map[string]string{"deliveryTime": "Мы работаем с " +
				workinghours.Display(hours.Start) + " до " + workinghours.Display(hours.End)}
			return e
		}
	}
	if _, signedIn := auth.FromContext(r.Context()); !signedIn {
		var guestID string
		if cookie, err := r.Cookie("citypark_guest_checkout"); err == nil {
			guestID, _ = h.tokens.GuestCheckoutID(cookie.Value)
		}
		if guestID == "" {
			proof, err := h.tokens.IssueGuestCheckout()
			if err != nil {
				return err
			}
			guestID, err = h.tokens.GuestCheckoutID(proof)
			if err != nil {
				return err
			}
			http.SetCookie(w, &http.Cookie{Name: "citypark_guest_checkout", Value: proof,
				Path: "/api/v1/orders", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
		}
		in.GuestCheckoutID = guestID
	}
	res, err := h.svc.Create(r.Context(), in)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, res)
	return nil
}

func (h *Handler) addItem(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var in ItemInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
	}
	ctx := r.Context()
	orders := NewStore(h.svc.db)
	if o, err := orders.ByID(ctx, id); errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	} else if err != nil {
		return err
	} else if o.PromoCode != nil {
		return httpx.NewError(409, "Нельзя изменять состав заказа с промокодом.")
	}
	products, err := catalog.NewStore(h.svc.db).ProductsByIDs(ctx, []int64{in.ProductID})
	if err != nil {
		return err
	}
	if len(products) == 0 {
		return httpx.BadRequest("Продукт не найден.")
	}
	err = orders.UpsertItem(ctx, id, InsertItem{ProductID: in.ProductID, Amount: in.Amount, OfferedPrice: products[0].Price})
	if err != nil {
		return err
	}
	o, err := orders.ByID(ctx, id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, o)
	return nil
}

func (h *Handler) removeItem(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	productID, err := httpx.PathID(r, "productId")
	if err != nil {
		return err
	}
	if err := NewStore(h.svc.db).RemoveItem(r.Context(), id, productID); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	if err := NewStore(h.svc.db).Delete(r.Context(), id); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

// privateOrders also covers authentication failures, before a handler is called.
func privateOrders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		next.ServeHTTP(w, r)
	})
}

func parseMoneta(w http.ResponseWriter, r *http.Request) (PaymentCallback, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		return PaymentCallback{}, err
	}
	for key, values := range r.Form {
		if strings.HasPrefix(key, "MNT_") && len(values) != 1 {
			return PaymentCallback{}, fmt.Errorf("duplicate Moneta field")
		}
	}
	return PaymentCallback{
		MerchantID: r.Form.Get("MNT_ID"), TransactionID: r.Form.Get("MNT_TRANSACTION_ID"),
		OperationID: r.Form.Get("MNT_OPERATION_ID"), Amount: r.Form.Get("MNT_AMOUNT"),
		Currency: r.Form.Get("MNT_CURRENCY_CODE"), SubscriberID: r.Form.Get("MNT_SUBSCRIBER_ID"),
		TestMode: r.Form.Get("MNT_TEST_MODE"), Command: r.Form.Get("MNT_COMMAND"), Signature: r.Form.Get("MNT_SIGNATURE"),
	}, nil
}

func (h *Handler) monetaCheck(w http.ResponseWriter, r *http.Request) {
	cb, err := parseMoneta(w, r)
	if err != nil || cb.Command != "CHECK" {
		http.Error(w, "FAIL", http.StatusBadRequest)
		return
	}
	response, err := h.svc.PaymentCheck(r.Context(), cb)
	if err != nil {
		status := http.StatusInternalServerError
		var he *httpx.Error
		if errors.As(err, &he) {
			status = he.Status
		}
		http.Error(w, "FAIL", status)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = io.WriteString(w, xml.Header)
	_ = xml.NewEncoder(w).Encode(response)
}
