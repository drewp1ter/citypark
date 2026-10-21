package catalog

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
)

// Handler serves the public catalog routes.
type Handler struct {
	store           *Store
	clock           clock.Clock
	defaultPageSize int
}

// NewHandler builds the catalog handler. defaultPageSize is used by the
// product search route.
func NewHandler(s *Store, c clock.Clock, defaultPageSize int) *Handler {
	return &Handler{store: s, clock: c, defaultPageSize: defaultPageSize}
}

// Routes registers the public catalog routes.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/categories", httpx.Handle(h.listCategories))
	r.Get("/categories/{id}", httpx.Handle(h.getCategory))
	r.Get("/categories/{id}/products", httpx.Handle(h.categoryProducts))
	r.Get("/products", httpx.Handle(h.searchProducts))
}

// maxSearchWords caps the number of words used by the product search.
const maxSearchWords = 20

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) error {
	list, err := h.store.ListCategories(r.Context())
	if err != nil {
		return err
	}
	now := h.clock.Now()
	views := make([]CategoryView, len(list))
	for i, c := range list {
		views[i] = c.View(now)
	}
	httpx.JSON(w, http.StatusOK, views)
	return nil
}

func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	c, err := h.store.CategoryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return httpx.NotFound()
		}
		return err
	}
	httpx.JSON(w, http.StatusOK, c.View(h.clock.Now()))
	return nil
}

func (h *Handler) categoryProducts(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		return err
	}
	list, err := h.store.ProductsByCategory(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, h.productViews(list))
	return nil
}

func (h *Handler) searchProducts(w http.ResponseWriter, r *http.Request) error {
	limit, offset, err := httpx.Pagination(r, h.defaultPageSize)
	if err != nil {
		return err
	}
	words := strings.Fields(strings.ToLower(r.URL.Query().Get("query")))
	if len(words) > maxSearchWords {
		words = words[:maxSearchWords]
	}
	if words == nil {
		words = []string{}
	}
	list, err := h.store.SearchProducts(r.Context(), words, limit, offset)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, h.productViews(list))
	return nil
}

// productViews builds the JSON shapes at the current time. It never returns nil, so JSON shows [].
func (h *Handler) productViews(list []Product) []ProductView {
	now := h.clock.Now()
	views := make([]ProductView, len(list))
	for i, p := range list {
		views[i] = p.View(now)
	}
	return views
}
