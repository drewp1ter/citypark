package deliverysettings

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

const (
	validationMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"
	maxBelowMin       = "Максимальная цена должна быть не меньше минимальной"
)

type Handler struct {
	store  *Store
	tokens *auth.Tokens
}

func NewHandler(s *Store, tokens *auth.Tokens) *Handler {
	return &Handler{store: s, tokens: tokens}
}

type response struct {
	Settings
	UpdatedAt time.Time `json:"updatedAt"`
}

// Routes registers GET /delivery-settings (public) and PUT /delivery-settings (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/delivery-settings", httpx.Handle(h.get))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Put("/delivery-settings", httpx.Handle(h.put))
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	settings, at, err := h.store.Get(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, response{Settings: settings, UpdatedAt: at})
	return nil
}

type input struct {
	Cooking        string `json:"cooking" validate:"required,max=100"`
	OutskirtsLabel string `json:"outskirtsLabel" validate:"required,max=60"`
	OutskirtsMin   *int64 `json:"outskirtsMin" validate:"required,min=0,max=100000"`
	OutskirtsMax   *int64 `json:"outskirtsMax" validate:"required,min=0,max=100000"`
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) error {
	var in input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	in.Cooking = strings.TrimSpace(in.Cooking)
	in.OutskirtsLabel = strings.TrimSpace(in.OutskirtsLabel)
	if err := validate.Struct(in); err != nil {
		return err
	}
	if *in.OutskirtsMin > *in.OutskirtsMax {
		e := httpx.NewError(http.StatusBadRequest, validationMessage)
		e.Fields = map[string]string{"outskirtsMax": maxBelowMin}
		return e
	}
	settings := Settings{Cooking: in.Cooking, OutskirtsLabel: in.OutskirtsLabel, OutskirtsMin: money.Kopecks(*in.OutskirtsMin * 100), OutskirtsMax: money.Kopecks(*in.OutskirtsMax * 100)}
	at, err := h.store.Put(r.Context(), settings)
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, response{Settings: settings, UpdatedAt: at})
	return nil
}
