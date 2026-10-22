package iiko

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
)

var errSyncRunning = httpx.NewError(http.StatusConflict, "Обновление меню уже выполняется")

// Handler serves the endpoints that start a menu sync.
type Handler struct {
	syncer *Syncer
	token  string
	tokens *auth.Tokens
}

// NewHandler creates a Handler. token is IIKO_UPDATE_TOKEN ("" disables POST /iiko/update-products).
func NewHandler(s *Syncer, token string, tokens *auth.Tokens) *Handler {
	return &Handler{syncer: s, token: token, tokens: tokens}
}

// Routes registers POST /iiko/update-products (shared secret) and POST /iiko/sync (admin).
func (h *Handler) Routes(r chi.Router) {
	r.Post("/iiko/update-products", httpx.Handle(h.updateProducts))
	r.Group(func(r chi.Router) {
		r.Use(h.tokens.RequireRole(auth.RoleAdmin))
		r.Post("/iiko/sync", httpx.Handle(h.sync))
	})
}

func (h *Handler) updateProducts(w http.ResponseWriter, r *http.Request) error {
	if h.token == "" {
		return httpx.NotFound()
	}
	given := []byte(r.Header.Get("X-Update-Token"))
	if subtle.ConstantTimeCompare(given, []byte(h.token)) != 1 {
		return httpx.Unauthorized()
	}
	res, err := h.run(r)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, res)
	return nil
}

func (h *Handler) sync(w http.ResponseWriter, r *http.Request) error {
	res, err := h.run(r)
	if err != nil {
		return err
	}
	var by int64
	if id, ok := auth.FromContext(r.Context()); ok {
		by = id.UserID
	}
	slog.Info("menu sync done", "totalCategories", res.TotalCategories, "totalProducts", res.TotalProducts, "by", by)
	httpx.JSON(w, http.StatusOK, res)
	return nil
}

// run starts a sync; a running sync becomes 409.
func (h *Handler) run(r *http.Request) (Result, error) {
	res, err := h.syncer.Sync(r.Context())
	if errors.Is(err, ErrSyncRunning) {
		return Result{}, errSyncRunning
	}
	return res, err
}
