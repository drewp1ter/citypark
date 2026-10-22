package order

import (
	"errors"
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
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

type HandlerOptions struct {
	WorkingHoursStart string
	WorkingHoursEnd   string
	DefaultPageSize   int
}

// Handler serves the order endpoints.
type Handler struct {
	svc    *Service
	tokens *auth.Tokens // used by the read and admin routes (tasks 28-30)
	opts   HandlerOptions
}

func NewHandler(svc *Service, tokens *auth.Tokens, opts HandlerOptions) *Handler {
	return &Handler{svc: svc, tokens: tokens, opts: opts}
}

// Routes registers POST /orders, which is public and limited to the working hours,
// the read endpoints for users and admins, and the admin routes that change an order.
func (h *Handler) Routes(r chi.Router) {
	r.With(WorkingHours(h.svc.clock, h.opts.WorkingHoursStart, h.opts.WorkingHoursEnd)).
		Post("/orders", httpx.Handle(h.create))

	r.Post("/webhooks/moneta", h.moneta)

	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleUser))
		r.Get("/orders/my", httpx.Handle(h.listMine))
		r.Get("/orders/{id}", httpx.Handle(h.get))
	})
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Get("/orders", httpx.Handle(h.list))
		r.Post("/orders/{id}/items", httpx.Handle(h.addItem))
		r.Delete("/orders/{id}/items/{productId}", httpx.Handle(h.removeItem))
		r.Delete("/orders/{id}", httpx.Handle(h.delete))
		r.Post("/orders/{id}/confirm", httpx.Handle(h.confirm))
	})
}

// moneta answers the payment callback. Moneta expects the plain text SUCCESS;
// any other answer makes it repeat the call, so this route does not use httpx.Handle.
func (h *Handler) moneta(w http.ResponseWriter, r *http.Request) {
	cb := PaymentCallback{
		TransactionID: r.FormValue("MNT_TRANSACTION_ID"),
		OperationID:   r.FormValue("MNT_OPERATION_ID"),
		Amount:        r.FormValue("MNT_AMOUNT"),
		Signature:     r.FormValue("MNT_SIGNATURE"),
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
	orders, count, err := NewStore(h.svc.db).ListByCustomer(r.Context(), id.UserID, limit, offset)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, listResponse{Orders: orders, Count: count})
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
	if v := qs.Get("paymentType"); v != "" {
		if v != PaymentCash && v != PaymentOnline {
			fields["paymentType"] = "недопустимое значение"
		}
		f.PaymentType = v
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
	if who.Role != auth.RoleAdmin && o.Customer.ID != who.UserID {
		// do not reveal that someone else's order exists
		return httpx.NotFound()
	}
	httpx.JSON(w, http.StatusOK, o)
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var in CreateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := validate.Struct(in); err != nil {
		return err
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
	if _, err := orders.ByID(ctx, id); errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	} else if err != nil {
		return err
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

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	o, err := h.svc.Confirm(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, o)
	return nil
}
