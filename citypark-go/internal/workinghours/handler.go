package workinghours

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

const (
	validationMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"
	endBeforeStart    = "Время окончания должно быть позже начала"
)

type Handler struct {
	store  *Store
	tokens *auth.Tokens
}

func NewHandler(s *Store, tokens *auth.Tokens) *Handler {
	return &Handler{store: s, tokens: tokens}
}

type response struct {
	Hours
	UpdatedAt time.Time `json:"updatedAt"`
}

// Routes registers GET /working-hours (public) and PUT /working-hours (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/working-hours", httpx.Handle(h.get))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Put("/working-hours", httpx.Handle(h.put))
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	hours, at, err := h.store.Get(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, response{Hours: hours, UpdatedAt: at})
	return nil
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) error {
	var in Hours
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	in.Start = strings.TrimSpace(in.Start)
	in.End = strings.TrimSpace(in.End)
	if err := validate.Struct(in); err != nil {
		return err
	}
	if in.Start >= in.End {
		e := httpx.NewError(http.StatusBadRequest, validationMessage)
		e.Fields = map[string]string{"end": endBeforeStart}
		return e
	}
	at, err := h.store.Put(r.Context(), in)
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, response{Hours: in, UpdatedAt: at})
	return nil
}
