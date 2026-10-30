package customer

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/go-chi/chi/v5"
)

type PairingHandler struct {
	sessions *Service
	limits   *Limits
	enabled  bool
}

func NewPairingHandler(sessions *Service, limits *Limits, enabled bool) *PairingHandler {
	return &PairingHandler{sessions: sessions, limits: limits, enabled: enabled}
}
func (h *PairingHandler) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "private, no-store")
				if !h.enabled {
					httpx.WriteError(w, r, httpx.NewError(503, "Вход временно недоступен"))
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		r.Post("/customer/pairings", httpx.Handle(h.create))
		r.Post("/customer/pairings/redeem", httpx.Handle(h.redeem))
		r.Get("/customer/pairings/{id}", httpx.Handle(h.get))
		r.Post("/customer/pairings/{id}/confirm", httpx.Handle(h.confirm))
		r.Delete("/customer/pairings/{id}", httpx.Handle(h.cancel))
	})
}
func (h *PairingHandler) session(r *http.Request) (Session, error) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer cs_") {
		return Session{}, httpx.Unauthorized()
	}
	s, err := h.sessions.Authenticate(r.Context(), strings.TrimPrefix(values[0], "Bearer "))
	if errors.Is(err, ErrInvalidSession) {
		return Session{}, httpx.Unauthorized()
	}
	return s, err
}
func (h *PairingHandler) recent(s Session) error {
	now := h.sessions.clock.Now()
	if now.Before(s.AuthenticatedAt) || now.Sub(s.AuthenticatedAt) > 5*time.Minute {
		return pairingHTTPError(ErrPairingStale)
	}
	return nil
}
func pairingHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidSession):
		return httpx.Unauthorized()
	case errors.Is(err, ErrPairingStale):
		return httpx.NewError(403, "Подтвердите вход повторно")
	case errors.Is(err, ErrPairingSurface):
		return httpx.NewError(403, "Подтвердите связь в нужном интерфейсе")
	case errors.Is(err, ErrPairingInvalid):
		return httpx.BadRequest("Код недействителен или устарел")
	case errors.Is(err, ErrPairingSameAccount):
		return httpx.NewError(409, "Аккаунты уже связаны")
	case errors.Is(err, ErrPairingNotFound):
		return httpx.NotFound()
	case errors.Is(err, ErrMergeConflict):
		return httpx.NewError(409, "Способ входа уже используется")
	case errors.Is(err, ErrMergeForbidden):
		return httpx.NewError(403, "Объединение аккаунтов недоступно")
	default:
		return err
	}
}
func (h *PairingHandler) limit(w http.ResponseWriter, allowed bool, seconds int) error {
	if allowed {
		return nil
	}
	retryAfter(w, seconds)
	return httpx.NewError(429, "Слишком много попыток. Попробуйте позже")
}
func (h *PairingHandler) create(w http.ResponseWriter, r *http.Request) error {
	s, err := h.session(r)
	if err != nil {
		return err
	}
	if s.Surface != "mini" || s.Method != "telegram" {
		return pairingHTTPError(ErrPairingSurface)
	}
	if err := h.recent(s); err != nil {
		return err
	}
	if ok, seconds := h.limits.PairCreate(s.UserID); !ok {
		return h.limit(w, ok, seconds)
	}
	v, err := h.sessions.CreatePairing(r.Context(), s)
	if err != nil {
		return pairingHTTPError(err)
	}
	httpx.JSON(w, 201, v)
	return nil
}
func (h *PairingHandler) redeem(w http.ResponseWriter, r *http.Request) error {
	s, err := h.session(r)
	if err != nil {
		return err
	}
	if s.Surface != "web" {
		return pairingHTTPError(ErrPairingSurface)
	}
	if err := h.recent(s); err != nil {
		return err
	}
	var v struct {
		Code string `json:"code" validate:"required"`
	}
	if err := httpx.DecodeJSON(w, r, &v); err != nil {
		return err
	}
	if ok, seconds := h.limits.PairRedeem(s.UserID, r); !ok {
		return h.limit(w, ok, seconds)
	}
	status, err := h.sessions.RedeemPairing(r.Context(), v.Code, s)
	if err != nil {
		return pairingHTTPError(err)
	}
	httpx.JSON(w, 200, status)
	return nil
}
func (h *PairingHandler) get(w http.ResponseWriter, r *http.Request) error {
	s, err := h.session(r)
	if err != nil {
		return err
	}
	v, err := h.sessions.GetPairing(r.Context(), chi.URLParam(r, "id"), s)
	if err != nil {
		return pairingHTTPError(err)
	}
	httpx.JSON(w, 200, v)
	return nil
}
func (h *PairingHandler) confirm(w http.ResponseWriter, r *http.Request) error {
	s, err := h.session(r)
	if err != nil {
		return err
	}
	if err := h.recent(s); err != nil {
		return err
	}
	var body map[string]json.RawMessage
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if body == nil || len(body) != 0 {
		return httpx.BadRequest("Некорректный JSON")
	}
	if ok, seconds := h.limits.PairConfirm(s.UserID); !ok {
		return h.limit(w, ok, seconds)
	}
	v, err := h.sessions.ConfirmPairing(r.Context(), chi.URLParam(r, "id"), s)
	if err != nil {
		return pairingHTTPError(err)
	}
	httpx.JSON(w, 200, v)
	return nil
}
func (h *PairingHandler) cancel(w http.ResponseWriter, r *http.Request) error {
	s, err := h.session(r)
	if err != nil {
		return err
	}
	if err := h.sessions.CancelPairing(r.Context(), chi.URLParam(r, "id"), s); err != nil {
		return pairingHTTPError(err)
	}
	httpx.NoContent(w)
	return nil
}
