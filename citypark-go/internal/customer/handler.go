package customer

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/drewp1ter/citypark-go/internal/validate"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	users    *user.Store
	methods  *Store
	sessions *Service
	adapter  *Adapter
}

func NewHandler(users *user.Store, methods *Store, sessions *Service, adapter *Adapter) *Handler {
	return &Handler{users: users, methods: methods, sessions: sessions, adapter: adapter}
}
func (h *Handler) Routes(r chi.Router) {
	availability := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "private, no-store")
			if !h.adapter.Enabled {
				httpx.WriteError(w, r, httpx.NewError(http.StatusServiceUnavailable, "Вход временно недоступен"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	r.Group(func(r chi.Router) {
		r.Use(availability)
		r.Use(h.adapter.Required(false))
		r.Get("/customer/me", httpx.Handle(h.me))
		r.Put("/customer/me", httpx.Handle(h.update))
		r.Post("/auth/customer/logout", httpx.Handle(h.logout))
	})
	r.Group(func(r chi.Router) {
		r.Use(availability)
		r.Use(h.adapter.Required(true))
		r.Get("/customer/methods", httpx.Handle(h.linkedMethods))
	})
}
func (h *Handler) linkedMethods(w http.ResponseWriter, r *http.Request) error {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return httpx.Unauthorized()
	}
	if id.Role != auth.RoleUser {
		return httpx.Forbidden()
	}
	methods, err := h.methods.Methods(r.Context(), id.UserID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, struct {
		Methods []Method `json:"methods"`
	}{methods})
	return nil
}
func (h *Handler) current(r *http.Request) (user.User, error) {
	id, ok := auth.FromContext(r.Context())
	if !ok || id.Role != auth.RoleUser {
		return user.User{}, httpx.Unauthorized()
	}
	u, err := h.users.ByID(r.Context(), id.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return user.User{}, httpx.Unauthorized()
	}
	return u, err
}
func (h *Handler) me(w http.ResponseWriter, r *http.Request) error {
	u, err := h.current(r)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, u.Response())
	return nil
}

type updateRequest struct {
	Name  string `json:"name" validate:"required,max=80"`
	Phone string `json:"phone" validate:"omitempty,ruphone"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	var fields map[string]json.RawMessage
	if err := httpx.DecodeJSON(w, r, &fields); err != nil {
		return err
	}
	for k := range fields {
		if k != "name" && k != "phone" {
			return httpx.NewError(http.StatusBadRequest, "Недопустимые поля профиля")
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	var req updateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return httpx.BadRequest("Некорректный JSON")
	}
	if err := validate.Struct(req); err != nil {
		return err
	}
	u, err := h.current(r)
	if err != nil {
		return err
	}
	phone := u.Phone
	if _, supplied := fields["phone"]; supplied {
		phone = req.Phone
	}
	if phone != "" {
		phone, _ = validate.NormalizePhone(phone)
	}
	updated, err := h.users.Update(r.Context(), u.ID, user.UpdateParams{Name: req.Name, Phone: phone})
	if store.IsUniqueViolation(err) {
		return httpx.NewError(http.StatusConflict, "Телефон уже используется")
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
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {
	if _, err := h.current(r); err != nil {
		return err
	}
	token := r.Header.Get("Authorization")[7:]
	if err := h.sessions.Revoke(r.Context(), token); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}
