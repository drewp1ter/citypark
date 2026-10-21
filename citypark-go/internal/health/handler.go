package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/httpx"
)

// Routes registers GET /health.
func Routes(r chi.Router) {
	r.Get("/health", httpx.Handle(func(w http.ResponseWriter, _ *http.Request) error {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return nil
	}))
}
