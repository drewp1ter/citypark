package deliveryprice

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

// Handler serves delivery prices. It takes *sqlx.DB because POST needs a transaction.
type Handler struct {
	db     *sqlx.DB
	tokens *auth.Tokens
}

func NewHandler(db *sqlx.DB, tokens *auth.Tokens) *Handler {
	return &Handler{db: db, tokens: tokens}
}

type createRequest struct {
	DeliveryPrices []Input `json:"deliveryPrices" validate:"required,min=1,dive"`
}

var errNotUnique = httpx.NewError(http.StatusConflict, "name должно быть уникальным")

// Routes registers GET /delivery-prices (public) and POST, PATCH, DELETE (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/delivery-prices", httpx.Handle(h.list))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Post("/delivery-prices", httpx.Handle(h.create))
		r.Patch("/delivery-prices/{id}", httpx.Handle(h.update))
		r.Delete("/delivery-prices/{id}", httpx.Handle(h.remove))
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	list, err := NewStore(h.db).List(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, list)
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var req createRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	for i := range req.DeliveryPrices {
		trim(&req.DeliveryPrices[i])
	}
	if err := validate.Struct(req); err != nil {
		return err
	}

	created := make([]DeliveryPrice, 0, len(req.DeliveryPrices))
	err := store.InTx(r.Context(), h.db, func(tx *sqlx.Tx) error {
		s := NewStore(tx)
		for _, in := range req.DeliveryPrices {
			d, err := s.Create(r.Context(), in)
			if err != nil {
				return err
			}
			created = append(created, d)
		}
		return nil
	})
	if err != nil {
		return writeErr(err)
	}
	httpx.JSON(w, http.StatusCreated, created)
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	var in Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	trim(&in)
	if err := validate.Struct(in); err != nil {
		return err
	}

	d, err := NewStore(h.db).Update(r.Context(), id, in)
	if err != nil {
		return writeErr(err)
	}
	httpx.JSON(w, http.StatusOK, d)
	return nil
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	if err := NewStore(h.db).Delete(r.Context(), id); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

// trim runs before validation so that a whitespace-only name fails "required".
func trim(in *Input) {
	in.Name = strings.TrimSpace(in.Name)
	in.Value = strings.TrimSpace(in.Value)
}

// writeErr maps store errors of a write to API errors.
func writeErr(err error) error {
	switch {
	case store.IsUniqueViolation(err):
		return errNotUnique
	case errors.Is(err, store.ErrNotFound):
		return httpx.NotFound()
	}
	return err
}
