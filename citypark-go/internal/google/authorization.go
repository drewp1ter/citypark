package google

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

const authorizationEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"

type StartResult struct {
	AuthorizationURL string    `json:"authorizationUrl"`
	FlowToken        string    `json:"flowToken"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type Service struct {
	db                              *sqlx.DB
	clock                           clock.Clock
	clientID, callbackURL, endpoint string
	random                          io.Reader
	clientSecret, discoveryURL      string
	httpClient                      *http.Client
	provider                        providerCache
}

func NewService(db *sqlx.DB, c clock.Clock, clientID, callbackURL, endpoint string) *Service {
	if endpoint == "" {
		endpoint = authorizationEndpoint
	}
	return &Service{db: db, clock: c, clientID: clientID, callbackURL: callbackURL, endpoint: endpoint, random: rand.Reader, discoveryURL: defaultDiscoveryURL, httpClient: &http.Client{Timeout: 5 * time.Second}}
}

// ConfigureCompletion supplies the server-only secret and provider discovery endpoint.
func (s *Service) ConfigureCompletion(secret, discoveryURL string, client *http.Client) {
	s.clientSecret = secret
	if discoveryURL != "" {
		s.discoveryURL = discoveryURL
	}
	if client != nil {
		s.httpClient = client
	}
}

func (s *Service) Start(ctx context.Context, intent string, session *customer.Session) (StartResult, error) {
	if intent != "login" && intent != "link" {
		return StartResult{}, httpx.BadRequest("Некорректный способ входа")
	}
	if intent == "link" {
		if session == nil {
			return StartResult{}, httpx.Unauthorized()
		}
		if session.Surface != "web" {
			return StartResult{}, httpx.NewError(403, "Откройте сайт для подключения Google")
		}
		age := s.clock.Now().Sub(session.AuthenticatedAt)
		if session.UserID <= 0 || len(session.TokenHash) != 32 || age < 0 || age > 5*time.Minute {
			return StartResult{}, httpx.NewError(403, "Подтвердите вход повторно")
		}
		if err := s.validLink(ctx, attempt{UserID: &session.UserID, SessionHash: session.TokenHash}, session); err != nil {
			return StartResult{}, err
		}
	}
	proof := func() ([]byte, string, error) {
		b := make([]byte, 32)
		if _, err := io.ReadFull(s.random, b); err != nil {
			return nil, "", err
		}
		return b, base64.RawURLEncoding.EncodeToString(b), nil
	}
	stateRaw, state, err := proof()
	if err != nil {
		return StartResult{}, err
	}
	nonceRaw, nonce, err := proof()
	if err != nil {
		return StartResult{}, err
	}
	flowRaw, flow, err := proof()
	if err != nil {
		return StartResult{}, err
	}
	_, verifier, err := proof()
	if err != nil {
		return StartResult{}, err
	}
	stateHash, nonceHash, flowHash := sha256.Sum256(stateRaw), sha256.Sum256(nonceRaw), sha256.Sum256(flowRaw)
	challenge := sha256.Sum256([]byte(verifier))
	expires := s.clock.Now().Add(10 * time.Minute)
	var userID any
	var sessionHash any
	if intent == "link" {
		userID, sessionHash = session.UserID, session.TokenHash
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM customer_oauth_attempts WHERE state_hash IN (SELECT state_hash FROM customer_oauth_attempts WHERE expires_at <= $1 ORDER BY expires_at LIMIT 100)`, s.clock.Now())
	if err != nil {
		return StartResult{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO customer_oauth_attempts(state_hash,nonce_hash,flow_hash,pkce_verifier,intent,user_id,session_hash,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, stateHash[:], nonceHash[:], flowHash[:], verifier, intent, userID, sessionHash, expires)
	if err != nil {
		return StartResult{}, err
	}
	u, err := url.Parse(s.endpoint)
	if err != nil {
		return StartResult{}, err
	}
	q := u.Query()
	q.Set("client_id", s.clientID)
	q.Set("redirect_uri", s.callbackURL)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return StartResult{u.String(), flow, expires}, nil
}

type Handler struct {
	service  *Service
	sessions *customer.Service
	limits   *customer.Limits
	enabled  bool
}

func NewHandler(service *Service, sessions *customer.Service, limits *customer.Limits, enabled bool) *Handler {
	return &Handler{service, sessions, limits, enabled}
}
func (h *Handler) Routes(r chi.Router) {
	r.Post("/auth/customer/google/start", httpx.Handle(h.start))
	r.Post("/auth/customer/google/complete", httpx.Handle(h.complete))
}
func (h *Handler) start(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", "private, no-store")
	if !h.enabled {
		return httpx.NewError(503, "Вход через Google недоступен")
	}
	var body struct {
		Intent string `json:"intent"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if body.Intent != "login" && body.Intent != "link" {
		return httpx.BadRequest("Некорректный способ входа")
	}
	var session *customer.Session
	values := r.Header.Values("Authorization")
	if len(values) != 0 {
		if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer cs_") {
			return httpx.Unauthorized()
		}
		v, err := h.sessions.Authenticate(r.Context(), strings.TrimPrefix(values[0], "Bearer "))
		if errors.Is(err, customer.ErrInvalidSession) {
			return httpx.Unauthorized()
		}
		if err != nil {
			return err
		}
		session = &v
	}
	if body.Intent == "link" && session == nil {
		return httpx.Unauthorized()
	}
	if ok, retry := h.limits.GoogleStart(r); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		return httpx.NewError(429, "Слишком много попыток. Попробуйте позже")
	}
	result, err := h.service.Start(r.Context(), body.Intent, session)
	if err != nil {
		return err
	}
	httpx.JSON(w, 200, result)
	return nil
}
