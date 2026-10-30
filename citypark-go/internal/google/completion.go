package google

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/jmoiron/sqlx"
)

var invalidFlow = httpx.BadRequest("Не удалось подтвердить вход через Google")
var unavailable = httpx.NewError(503, "Вход через Google недоступен")
var conflict = httpx.NewError(409, "Способ входа уже используется")

type CompletionResult struct {
	User      *user.Response `json:"user,omitempty"`
	Token     string         `json:"token,omitempty"`
	ExpiresAt *time.Time     `json:"expiresAt,omitempty"`
	Linked    bool           `json:"linked,omitempty"`
}
type attempt struct {
	NonceHash   []byte `db:"nonce_hash"`
	Verifier    string `db:"pkce_verifier"`
	Intent      string `db:"intent"`
	UserID      *int64 `db:"user_id"`
	SessionHash []byte `db:"session_hash"`
}

func sha256Sum(v []byte) [32]byte { return sha256.Sum256(v) }
func equalBytes(a, b []byte) bool { return subtle.ConstantTimeCompare(a, b) == 1 }
func proofHash(raw string) ([]byte, error) {
	b, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil || len(b) != 32 || base64.RawURLEncoding.EncodeToString(b) != raw {
		return nil, invalidFlow
	}
	h := sha256Sum(b)
	return h[:], nil
}
func (s *Service) consume(ctx context.Context, state, flow string) (attempt, error) {
	sh, err := proofHash(state)
	if err != nil {
		return attempt{}, err
	}
	fh, err := proofHash(flow)
	if err != nil {
		return attempt{}, err
	}
	var a attempt
	err = store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		if e := tx.GetContext(ctx, &a, `UPDATE customer_oauth_attempts SET consumed_at=$3 WHERE state_hash=$1 AND flow_hash=$2 AND consumed_at IS NULL AND expires_at>$3 RETURNING nonce_hash,pkce_verifier,intent,user_id,session_hash`, sh, fh, s.clock.Now()); e != nil {
			return e
		}
		_, e := tx.ExecContext(ctx, `UPDATE customer_oauth_attempts SET pkce_verifier='' WHERE state_hash=$1`, sh)
		return e
	})
	if errors.Is(err, store.ErrNotFound) || err != nil && strings.Contains(err.Error(), "no rows") {
		return attempt{}, invalidFlow
	}
	return a, err
}
func (s *Service) validLink(ctx context.Context, a attempt, session *customer.Session) error {
	if session == nil {
		return httpx.Unauthorized()
	}
	if session.Surface != "web" {
		return httpx.NewError(403, "Откройте сайт для подключения Google")
	}
	age := s.clock.Now().Sub(session.AuthenticatedAt)
	if age < 0 || age > 5*time.Minute || a.UserID == nil || *a.UserID != session.UserID || !equalBytes(a.SessionHash, session.TokenHash) {
		return httpx.NewError(403, "Подтвердите вход повторно")
	}
	var count int
	err := s.db.GetContext(ctx, &count, `SELECT count(*) FROM customer_sessions cs JOIN users u ON u.id=cs.user_id WHERE cs.token_hash=$1 AND cs.user_id=$2 AND cs.revoked_at IS NULL AND cs.expires_at>$3 AND cs.surface='web' AND cs.authenticated_at=$4 AND u.role='user'`, a.SessionHash, *a.UserID, s.clock.Now(), session.AuthenticatedAt)
	if err != nil {
		return err
	}
	if count != 1 {
		return httpx.NewError(403, "Подтвердите вход повторно")
	}
	return nil
}
func (s *Service) Complete(ctx context.Context, state, flowToken, code, providerError string, session *customer.Session) (CompletionResult, error) {
	if (code == "") == (providerError == "") {
		return CompletionResult{}, invalidFlow
	}
	a, err := s.consume(ctx, state, flowToken)
	if err != nil {
		return CompletionResult{}, err
	}
	if a.Intent == "link" {
		if err = s.validLink(ctx, a, session); err != nil {
			return CompletionResult{}, err
		}
	}
	if providerError != "" {
		if providerError == "access_denied" {
			return CompletionResult{}, httpx.BadRequest("Вход через Google отменён")
		}
		return CompletionResult{}, invalidFlow
	}
	tokenURL, _, err := s.endpoints(ctx)
	if err != nil {
		return CompletionResult{}, unavailable
	}
	form := url.Values{"code": {code}, "client_id": {s.clientID}, "client_secret": {s.clientSecret}, "redirect_uri": {s.callbackURL}, "code_verifier": {a.Verifier}, "grant_type": {"authorization_code"}}
	var exchange struct {
		IDToken string `json:"id_token"`
	}
	_, err = s.getJSON(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", &exchange)
	if err != nil {
		if errors.Is(err, errProviderRejected) {
			return CompletionResult{}, invalidFlow
		}
		return CompletionResult{}, unavailable
	}
	if exchange.IDToken == "" {
		return CompletionResult{}, invalidFlow
	}
	claims, err := s.validateIDToken(ctx, exchange.IDToken, a.NonceHash)
	if err != nil {
		if errors.Is(err, errInvalidToken) {
			return CompletionResult{}, invalidFlow
		}
		return CompletionResult{}, unavailable
	}
	if a.Intent == "link" {
		err = store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
			if _, e := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "google:"+claims.Subject); e != nil {
				return e
			}
			var userID int64
			e := tx.GetContext(ctx, &userID, `SELECT cs.user_id FROM customer_sessions cs JOIN users u ON u.id=cs.user_id WHERE cs.token_hash=$1 AND cs.user_id=$2 AND cs.revoked_at IS NULL AND cs.expires_at>$3 AND cs.surface='web' AND cs.authenticated_at=$4 AND u.role='user' FOR UPDATE OF cs,u`, a.SessionHash, *a.UserID, s.clock.Now(), session.AuthenticatedAt)
			if errors.Is(store.NotFound(e), store.ErrNotFound) {
				return httpx.NewError(403, "Подтвердите вход повторно")
			}
			if e != nil {
				return e
			}
			var owner int64
			e = tx.GetContext(ctx, &owner, `SELECT user_id FROM customer_identities WHERE provider='google' AND subject=$1`, claims.Subject)
			if e == nil {
				if owner == *a.UserID {
					return nil
				}
				return conflict
			}
			if !errors.Is(store.NotFound(e), store.ErrNotFound) {
				return e
			}
			_, e = tx.ExecContext(ctx, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'google',$2)`, *a.UserID, claims.Subject)
			if store.IsUniqueViolation(e) {
				return conflict
			}
			return e
		})
		if err != nil {
			return CompletionResult{}, err
		}
		return CompletionResult{Linked: true}, nil
	}
	var id int64
	err = store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		// Serialise first logins for a subject so a failed reservation leaves no user.
		if _, e := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "google:"+claims.Subject); e != nil {
			return e
		}
		e := tx.GetContext(ctx, &id, `SELECT user_id FROM customer_identities WHERE provider='google' AND subject=$1`, claims.Subject)
		if e == nil {
			return nil
		}
		if !errors.Is(store.NotFound(e), store.ErrNotFound) {
			return e
		}
		name := strings.TrimSpace(claims.Name)
		if name == "" {
			name = "Пользователь"
		}
		for utf8.RuneCountInString(name) > 80 {
			_, size := utf8.DecodeLastRuneInString(name)
			name = name[:len(name)-size]
		}
		e = tx.GetContext(ctx, &id, `INSERT INTO users(name,email,phone,password_hash,role) VALUES($1,NULL,NULL,'','user') RETURNING id`, name)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'google',$2)`, id, claims.Subject)
		return e
	})
	if err != nil {
		return CompletionResult{}, err
	}
	u, err := user.NewStore(s.db).ByID(ctx, id)
	if err != nil {
		return CompletionResult{}, err
	}
	sessions := customer.NewService(s.db, s.clock)
	token, exp, err := sessions.CreateSession(ctx, id, "google", "web")
	if err != nil {
		return CompletionResult{}, err
	}
	response := u.Response()
	return CompletionResult{User: &response, Token: token, ExpiresAt: &exp}, nil
}
func (h *Handler) complete(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", "private, no-store")
	if !h.enabled {
		return unavailable
	}
	var body struct {
		State     string `json:"state"`
		FlowToken string `json:"flowToken"`
		Code      string `json:"code"`
		Error     string `json:"error"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		return err
	}
	if ok, retry := h.limits.GoogleComplete(r); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		return httpx.NewError(429, "Слишком много попыток. Попробуйте позже")
	}
	var session *customer.Session
	values := r.Header.Values("Authorization")
	if len(values) > 0 {
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
	result, err := h.service.Complete(r.Context(), body.State, body.FlowToken, body.Code, body.Error, session)
	if err != nil {
		requestID := make([]byte, 8)
		_, _ = rand.Read(requestID)
		category := "internal"
		var publicErr *httpx.Error
		if errors.As(err, &publicErr) {
			category = strconv.Itoa(publicErr.Status)
		}
		slog.Default().WarnContext(r.Context(), "google completion failed", "category", category, "request_id", hex.EncodeToString(requestID))
		return err
	}
	httpx.JSON(w, 200, result)
	return nil
}
