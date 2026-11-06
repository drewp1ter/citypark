package address

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
	"github.com/go-chi/chi/v5"
)

type BookHandler struct {
	store   *Store
	service *Service
	access  func(http.Handler) http.Handler
}

func NewBookHandler(s *Store, service *Service, access func(http.Handler) http.Handler) *BookHandler {
	return &BookHandler{store: s, service: service, access: access}
}
func (h *BookHandler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "private, no-store")
				next.ServeHTTP(w, r)
			})
		})
		r.Use(h.access)
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, ok := auth.FromContext(r.Context())
				if !ok {
					httpx.WriteError(w, r, httpx.Unauthorized())
					return
				}
				if id.Role != auth.RoleUser {
					httpx.WriteError(w, r, httpx.Forbidden())
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		r.Get("/customer/addresses", httpx.Handle(h.list))
		r.Post("/customer/addresses", httpx.Handle(h.create))
		r.Put("/customer/addresses/{id}", httpx.Handle(h.update))
		r.Put("/customer/addresses/{id}/default", httpx.Handle(h.setDefault))
		r.Delete("/customer/addresses/{id}", httpx.Handle(h.remove))
	})
}
func bookError(err error) error {
	if err == nil {
		return nil
	}
	var invalid *InvalidFieldsError
	switch {
	case errors.As(err, &invalid):
		return &httpx.Error{Status: 400, Message: "Проверьте адрес", Fields: invalid.Fields}
	case errors.Is(err, ErrOwnerUnavailable):
		return httpx.Unauthorized()
	case errors.Is(err, store.ErrNotFound):
		return httpx.NotFound()
	case errors.Is(err, ErrDuplicate):
		return &httpx.Error{Status: 409, Message: "Такой адрес уже сохранён", Fields: map[string]string{"street": "Такой адрес уже сохранён"}}
	default:
		// Never forward database errors that may contain submitted address fields to
		// the shared logger or monitoring capture.
		return errors.New("saved address operation failed")
	}
}
func decodeBookFields(w http.ResponseWriter, r *http.Request) (Fields, error) {
	bad := func() (Fields, error) { return Fields{}, httpx.BadRequest("Некорректный JSON") }
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil || !json.Valid(data) {
		return bad()
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	var object map[string]json.RawMessage
	if err := httpx.DecodeJSON(w, r, &object); err != nil || object == nil {
		return bad()
	}
	var in Fields
	allowed := map[string]*string{"recipientName": &in.RecipientName, "recipientPhone": &in.RecipientPhone, "name": &in.Name, "city": &in.City, "street": &in.Street, "apartment": &in.Apartment, "entrance": &in.Entrance}
	for key, raw := range object {
		target, ok := allowed[key]
		if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return bad()
		}
		if err := json.Unmarshal(raw, target); err != nil {
			return bad()
		}
	}
	in = CleanFields(in)
	if err := validate.Struct(in); err != nil {
		return Fields{}, bookError(&InvalidFieldsError{Fields: FieldErrors(in)})
	}
	return in, nil
}
func bookOwner(r *http.Request) int64 { id, _ := auth.FromContext(r.Context()); return id.UserID }
func (h *BookHandler) list(w http.ResponseWriter, r *http.Request) error {
	entries, err := h.store.ListEntries(r.Context(), bookOwner(r))
	if err != nil {
		return bookError(err)
	}
	httpx.JSON(w, 200, struct {
		Addresses []Entry `json:"addresses"`
	}{entries})
	return nil
}
func (h *BookHandler) create(w http.ResponseWriter, r *http.Request) error {
	in, err := decodeBookFields(w, r)
	if err != nil {
		return err
	}
	entry, created, err := h.service.CreateEntry(r.Context(), bookOwner(r), in)
	if err != nil {
		return bookError(err)
	}
	status := 200
	if created {
		status = 201
	}
	httpx.JSON(w, status, struct {
		Entry   Entry `json:"entry"`
		Created bool  `json:"created"`
	}{entry, created})
	return nil
}
func (h *BookHandler) update(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	in, err := decodeBookFields(w, r)
	if err != nil {
		return err
	}
	entry, err := h.service.UpdateEntry(r.Context(), bookOwner(r), id, in)
	if err != nil {
		return bookError(err)
	}
	httpx.JSON(w, 200, struct {
		Entry Entry `json:"entry"`
	}{entry})
	return nil
}
func (h *BookHandler) setDefault(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	entry, err := h.service.SetDefault(r.Context(), bookOwner(r), id)
	if err != nil {
		return bookError(err)
	}
	httpx.JSON(w, 200, struct {
		Entry Entry `json:"entry"`
	}{entry})
	return nil
}
func (h *BookHandler) remove(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	if err := h.service.DeleteEntry(r.Context(), bookOwner(r), id); err != nil {
		return bookError(err)
	}
	httpx.NoContent(w)
	return nil
}
