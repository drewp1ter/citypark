package homepagemenu

import (
	"errors"
	"net/http"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	store  *Store
	tokens *auth.Tokens
}

func NewHandler(s *Store, tokens *auth.Tokens) *Handler { return &Handler{s, tokens} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/homepage-menu", httpx.Handle(h.get))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Put("/homepage-menu", httpx.Handle(h.put))
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	out, err := h.store.Get(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) error {
	var in Configuration
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := h.store.Put(r.Context(), in); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return httpx.NotFound()
		}
		return err
	}
	httpx.JSON(w, http.StatusOK, in)
	return nil
}
