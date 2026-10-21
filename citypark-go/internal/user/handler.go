package user

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

// Handler serves the current user's profile.
type Handler struct {
	store  *Store
	tokens *auth.Tokens
}

func NewHandler(s *Store, tokens *auth.Tokens) *Handler {
	return &Handler{store: s, tokens: tokens}
}

// Routes registers GET /user and PUT /user, both behind tokens.RequireRole(auth.RoleUser).
func (h *Handler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleUser))
		r.Get("/user", httpx.Handle(h.me))
		r.Put("/user", httpx.Handle(h.update))
	})
}

type updateRequest struct {
	Name  string `json:"name" validate:"required,max=80"`
	Email string `json:"email" validate:"omitempty,email"`
	Phone string `json:"phone" validate:"omitempty,ruphone"`
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) error {
	u, err := h.current(r)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, u.Response())
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	var req updateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := validate.Struct(req); err != nil {
		return err
	}

	u, err := h.current(r)
	if err != nil {
		return err
	}

	params := UpdateParams{Name: req.Name, Phone: u.Phone}
	if e := normalizeEmail(req.Email); e != "" {
		params.Email = &e
	}
	if req.Phone != "" {
		params.Phone, _ = validate.NormalizePhone(req.Phone)
	}

	updated, err := h.store.Update(r.Context(), u.ID, params)
	if store.IsUniqueViolation(err) {
		return httpx.NewError(http.StatusConflict, "email или телефон уже используется")
	}
	if errors.Is(err, store.ErrNotFound) {
		return httpx.Unauthorized()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, updated.Response())
	return nil
}

// current loads the user of the request. A token whose user was deleted is a 401.
func (h *Handler) current(r *http.Request) (User, error) {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return User{}, httpx.Unauthorized()
	}
	u, err := h.store.ByID(r.Context(), id.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return User{}, httpx.Unauthorized()
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}
