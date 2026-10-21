package address

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

// Handler serves the signed-in user's delivery addresses.
type Handler struct {
	store  *Store
	tokens *auth.Tokens
}

func NewHandler(s *Store, tokens *auth.Tokens) *Handler {
	return &Handler{store: s, tokens: tokens}
}

// Input is exported because the order package reuses it (task 26).
type Input struct {
	City    string `json:"city" validate:"required,max=80"`
	Address string `json:"address" validate:"required,max=1000"`
}

// Routes registers GET /addresses, POST /addresses and DELETE /addresses/{id},
// all behind tokens.RequireRole(auth.RoleUser).
func (h *Handler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleUser))
		r.Get("/addresses", httpx.Handle(h.list))
		r.Post("/addresses", httpx.Handle(h.create))
		r.Delete("/addresses/{id}", httpx.Handle(h.remove))
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return httpx.Unauthorized()
	}
	list, err := h.store.ListByUser(r.Context(), id.UserID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, list)
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return httpx.Unauthorized()
	}
	var in Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	// Trim before validating so that a whitespace-only value fails "required".
	in.City = strings.TrimSpace(in.City)
	in.Address = strings.TrimSpace(in.Address)
	if err := validate.Struct(in); err != nil {
		return err
	}

	a, err := h.store.FindOrCreate(r.Context(), in.City, in.Address)
	if err != nil {
		return err
	}
	if err := h.store.LinkUser(r.Context(), id.UserID, a.ID); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, a)
	return nil
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) error {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return httpx.Unauthorized()
	}
	addressID, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	if err := h.store.UnlinkUser(r.Context(), id.UserID, addressID); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}
