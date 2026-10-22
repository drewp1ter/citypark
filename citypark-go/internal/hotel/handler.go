package hotel

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

// Handler serves the free-room counts and the hotel prices.
type Handler struct {
	store  *Store
	secret string
	tokens *auth.Tokens
}

// NewHandler creates a Handler. secret is the value of HOTEL_WEBHOOK_SECRET; ""
// disables POST /hotel/rooms. tokens guards PUT /hotel/prices. The store must be built on a *sqlx.DB
// because PUT /hotel/prices needs a transaction.
func NewHandler(s *Store, secret string, tokens *auth.Tokens) *Handler {
	return &Handler{store: s, secret: secret, tokens: tokens}
}

type updateRequest struct {
	Rooms *roomsInput `json:"rooms" validate:"required"`
}

// roomsInput uses pointers so that 0 is valid and a missing field is an error.
type roomsInput struct {
	VIP           *int `json:"vip" validate:"required,min=0,max=1"`
	SingleBalcony *int `json:"singleBalcony" validate:"required,min=0,max=1"`
	Single        *int `json:"single" validate:"required,min=0,max=2"`
	DoubleTwin    *int `json:"doubleTwin" validate:"required,min=0,max=4"`
	DoubleEuro    *int `json:"doubleEuro" validate:"required,min=0,max=4"`
}

type response struct {
	Rooms     Rooms     `json:"rooms"`
	Free      int       `json:"free"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newResponse(st State) response {
	return response{Rooms: st.Rooms, Free: st.Free(), UpdatedAt: st.UpdatedAt}
}

// Routes registers GET /hotel/rooms (public), POST /hotel/rooms (shared secret), GET /hotel/prices (public)
// and PUT /hotel/prices (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/hotel/rooms", httpx.Handle(h.get))
	r.Post("/hotel/rooms", httpx.Handle(h.update))
	r.Get("/hotel/prices", httpx.Handle(h.getPrices))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Put("/hotel/prices", httpx.Handle(h.putPrices))
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	st, err := h.store.Get(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, newResponse(st))
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	if h.secret == "" {
		return httpx.NotFound()
	}
	given := []byte(r.Header.Get("X-Webhook-Secret"))
	if subtle.ConstantTimeCompare(given, []byte(h.secret)) != 1 {
		return httpx.Unauthorized()
	}

	var req updateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := validate.Struct(req); err != nil {
		return err
	}

	in := req.Rooms
	st, err := h.store.Set(r.Context(), Rooms{
		VIP:           *in.VIP,
		SingleBalcony: *in.SingleBalcony,
		Single:        *in.Single,
		DoubleTwin:    *in.DoubleTwin,
		DoubleEuro:    *in.DoubleEuro,
	})
	if err != nil {
		return err
	}
	slog.Info("hotel rooms updated",
		"vip", st.VIP, "singleBalcony", st.SingleBalcony, "single", st.Single,
		"doubleTwin", st.DoubleTwin, "doubleEuro", st.DoubleEuro)
	httpx.JSON(w, http.StatusOK, newResponse(st))
	return nil
}
