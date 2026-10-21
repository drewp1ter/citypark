package iiko

import (
	"crypto/subtle"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/httpx"
)

// Handler serves the endpoint that starts a menu sync.
type Handler struct {
	syncer *Syncer
	token  string
}

// NewHandler creates a Handler. token is the value of IIKO_UPDATE_TOKEN; ""
// disables the endpoint.
func NewHandler(s *Syncer, token string) *Handler {
	return &Handler{syncer: s, token: token}
}

// Routes registers POST /iiko/update-products.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/iiko/update-products", httpx.Handle(h.updateProducts))
}

func (h *Handler) updateProducts(w http.ResponseWriter, r *http.Request) error {
	if h.token == "" {
		return httpx.NotFound()
	}
	given := []byte(r.Header.Get("X-Update-Token"))
	if subtle.ConstantTimeCompare(given, []byte(h.token)) != 1 {
		return httpx.Unauthorized()
	}

	res, err := h.syncer.Sync(r.Context())
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, res)
	return nil
}
