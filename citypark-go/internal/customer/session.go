package customer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/mailer"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

var ErrInvalidSession = errors.New("invalid customer session")

type Session struct {
	UserID          int64     `db:"user_id"`
	TokenHash       []byte    `db:"token_hash"`
	AuthenticatedAt time.Time `db:"authenticated_at"`
	ExpiresAt       time.Time `db:"expires_at"`
	Method          string    `db:"method"`
	Surface         string    `db:"surface"`
}

func (s *Store) insert(ctx context.Context, v Session) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO customer_sessions (token_hash,user_id,authenticated_at,expires_at,method,surface) VALUES ($1,$2,$3,$4,$5,$6)`, v.TokenHash, v.UserID, v.AuthenticatedAt, v.ExpiresAt, v.Method, v.Surface)
	return err
}
func (s *Store) active(ctx context.Context, hash []byte, now time.Time) (Session, error) {
	var v Session
	err := s.db.GetContext(ctx, &v, `SELECT user_id,token_hash,authenticated_at,expires_at,method,surface FROM customer_sessions WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>$2`, hash, now)
	return v, store.NotFound(err)
}
func (s *Store) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.db.ExecContext(ctx, `UPDATE customer_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, tokenHash)
	return err
}
func (s *Store) RevokeAllSessions(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE customer_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}

type Service struct {
	store        *Store
	db           *sqlx.DB
	clock        clock.Clock
	random       io.Reader
	mailer       mailer.Mailer
	publicOrigin string
	log          *slog.Logger
}

func NewService(db *sqlx.DB, c clock.Clock) *Service {
	return &Service{store: NewStore(db), db: db, clock: c, random: rand.Reader}
}
func (s *Service) CreateSession(ctx context.Context, userID int64, method, surface string) (string, time.Time, error) {
	if userID <= 0 || !((method == "password" || method == "google") && surface == "web" || method == "telegram" && surface == "mini") {
		return "", time.Time{}, ErrInvalidSession
	}
	var role string
	if err := s.store.db.GetContext(ctx, &role, `SELECT role FROM users WHERE id=$1`, userID); err != nil || role != auth.RoleUser {
		if err != nil && !errors.Is(store.NotFound(err), store.ErrNotFound) {
			return "", time.Time{}, err
		}
		return "", time.Time{}, ErrInvalidSession
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := "cs_" + base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256(raw)
	now := s.clock.Now()
	exp := now.Add(30 * 24 * time.Hour)
	if err := s.store.insert(ctx, Session{UserID: userID, TokenHash: hash[:], AuthenticatedAt: now, ExpiresAt: exp, Method: method, Surface: surface}); err != nil {
		return "", time.Time{}, err
	}
	return token, exp, nil
}
func HashToken(token string) ([]byte, error) {
	if len(token) != 46 || !strings.HasPrefix(token, "cs_") {
		return nil, ErrInvalidSession
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token[3:])
	if err != nil || len(raw) != 32 {
		return nil, ErrInvalidSession
	}
	hash := sha256.Sum256(raw)
	return hash[:], nil
}
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	hash, err := HashToken(token)
	if err != nil {
		return Session{}, ErrInvalidSession
	}
	v, err := s.store.active(ctx, hash, s.clock.Now())
	if errors.Is(err, store.ErrNotFound) {
		return Session{}, ErrInvalidSession
	}
	if err != nil {
		return Session{}, err
	}
	var role string
	if err := s.store.db.GetContext(ctx, &role, `SELECT role FROM users WHERE id=$1`, v.UserID); err != nil || role != auth.RoleUser {
		if err != nil && !errors.Is(store.NotFound(err), store.ErrNotFound) {
			return Session{}, err
		}
		return Session{}, ErrInvalidSession
	}
	return v, nil
}
func (s *Service) Revoke(ctx context.Context, token string) error {
	hash, err := HashToken(token)
	if err != nil {
		return err
	}
	return s.store.RevokeSession(ctx, hash)
}

// Adapter authenticates customer sessions, while preserving staff access on existing routes.
type Adapter struct {
	Sessions *Service
	Tokens   *auth.Tokens
	Enabled  bool
}

func (a *Adapter) credential(w http.ResponseWriter, r *http.Request, optional, staff bool, next http.Handler) {
	w.Header().Set("Cache-Control", "private, no-store")
	values := r.Header.Values("Authorization")
	if len(values) == 0 && optional {
		next.ServeHTTP(w, r)
		return
	}
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") || strings.TrimPrefix(values[0], "Bearer ") == "" {
		httpx.WriteError(w, r, httpx.Unauthorized())
		return
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if strings.HasPrefix(token, "cs_") {
		if !a.Enabled {
			httpx.WriteError(w, r, httpx.NewError(http.StatusServiceUnavailable, "Вход временно недоступен"))
			return
		}
		session, err := a.Sessions.Authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, ErrInvalidSession) {
				httpx.WriteError(w, r, httpx.Unauthorized())
			} else {
				httpx.WriteError(w, r, err)
			}
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.NewContext(r.Context(), auth.Identity{UserID: session.UserID, Role: auth.RoleUser})))
		return
	}
	claims, err := a.Tokens.Parse(token)
	if err != nil || (a.Enabled && claims.Role != auth.RoleAdmin) || (!staff && a.Enabled) {
		httpx.WriteError(w, r, httpx.Unauthorized())
		return
	}
	if !optional && !auth.Allows(claims.Role, auth.RoleUser) {
		httpx.WriteError(w, r, httpx.Forbidden())
		return
	}
	next.ServeHTTP(w, r.WithContext(auth.NewContext(r.Context(), auth.Identity{UserID: claims.UserID, Role: claims.Role})))
}
func (a *Adapter) Required(staff bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { a.credential(w, r, false, staff, next) })
	}
}
func (a *Adapter) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { a.credential(w, r, true, false, next) })
}
