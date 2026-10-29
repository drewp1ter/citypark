package customer

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/drewp1ter/citypark-go/internal/validate"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

const genericEmailMessage = "Если адрес доступен, письмо с подтверждением отправлено"
const badEmailProof = "Ссылка недействительна или устарела"
const badCredentials = "неверный логин или пароль."

type EmailHandler struct {
	db        *sqlx.DB
	users     *user.Store
	sessions  *Service
	limits    *Limits
	enabled   bool
	dummyHash string
}

func NewEmailHandler(db *sqlx.DB, sessions *Service, limits *Limits, enabled bool) *EmailHandler {
	dummy, _ := auth.HashPassword("dummy-password-value")
	return &EmailHandler{db: db, users: user.NewStore(db), sessions: sessions, limits: limits, enabled: enabled, dummyHash: dummy}
}
func (h *EmailHandler) Routes(r chi.Router) {
	r.Post("/auth/customer/register", httpx.Handle(h.register))
	r.Post("/auth/customer/verification/resend", httpx.Handle(h.resend))
	r.Post("/auth/customer/verification/confirm", httpx.Handle(h.verify))
	r.Post("/auth/customer/sign-in", httpx.Handle(h.signIn))
	r.Post("/customer/email/change", httpx.Handle(h.change))
	r.Post("/customer/email/confirm", httpx.Handle(h.confirmChange))
}
func (h *EmailHandler) available(w http.ResponseWriter) error {
	w.Header().Set("Cache-Control", "private, no-store")
	if !h.enabled {
		return httpx.NewError(503, "Вход временно недоступен")
	}
	return nil
}

type registerRequest struct {
	Name     string `json:"name" validate:"required,max=80"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"omitempty,ruphone"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}
type emailRequest struct {
	Email string `json:"email" validate:"required,email"`
}
type proofRequest struct {
	Token string `json:"token" validate:"required"`
}
type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func normalizedRequestEmail(raw string) (string, error) { return normalizedEmail(raw) }
func generic(w http.ResponseWriter) {
	httpx.JSON(w, 202, map[string]string{"message": genericEmailMessage})
}
func (h *EmailHandler) register(w http.ResponseWriter, r *http.Request) error {
	if err := h.available(w); err != nil {
		return err
	}
	var v registerRequest
	if err := httpx.DecodeJSON(w, r, &v); err != nil {
		return err
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	if err := validate.Struct(v); err != nil {
		return err
	}
	email, err := normalizedRequestEmail(v.Email)
	if err != nil {
		return httpx.BadRequest("Некорректный адрес электронной почты")
	}
	if !h.limits.Send(email, r) {
		generic(w)
		return nil
	}
	phone := ""
	if v.Phone != "" {
		phone, _ = validate.NormalizePhone(v.Phone)
	}
	hash, err := auth.HashPassword(v.Password)
	if err != nil {
		return err
	}
	var id int64
	err = store.InTx(r.Context(), h.db, func(tx *sqlx.Tx) error {
		u, e := user.NewStore(tx).Create(r.Context(), user.CreateParams{Name: v.Name, Email: &email, Phone: phone, PasswordHash: hash, Role: auth.RoleUser})
		if e != nil {
			return e
		}
		id = u.ID
		return NewStore(tx).AddIdentity(r.Context(), Identity{UserID: id, Provider: "email", Subject: email})
	})
	if store.IsUniqueViolation(err) {
		generic(w)
		return nil
	}
	if err != nil {
		return err
	}
	_, err = h.sessions.IssueEmailToken(r.Context(), id, "verify", email)
	if err != nil && !errors.Is(err, ErrEmailUnavailable) {
		return err
	}
	generic(w)
	return nil
}
func (h *EmailHandler) resend(w http.ResponseWriter, r *http.Request) error {
	if err := h.available(w); err != nil {
		return err
	}
	var v emailRequest
	if err := httpx.DecodeJSON(w, r, &v); err != nil {
		return err
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	if err := validate.Struct(v); err != nil {
		return err
	}
	email, err := normalizedRequestEmail(v.Email)
	if err != nil {
		return httpx.BadRequest("Некорректный адрес электронной почты")
	}
	if !h.limits.Send(email, r) {
		generic(w)
		return nil
	}
	id, err := NewStore(h.db).VerifiedEmailOwner(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		generic(w)
		return nil
	}
	if err != nil {
		return err
	}
	u, err := h.users.ByID(r.Context(), id)
	if err != nil {
		return err
	}
	if u.Role == auth.RoleUser && !u.IsEmailConfirmed && u.Email != nil && strings.EqualFold(strings.TrimSpace(*u.Email), email) && (u.PasswordHash != "" || u.LegacyPasswordHash != "") {
		_, err = h.sessions.IssueEmailToken(r.Context(), id, "verify", email)
		if err != nil && !errors.Is(err, ErrEmailUnavailable) {
			return err
		}
	}
	generic(w)
	return nil
}
func (h *EmailHandler) consume(ctx context.Context, raw, purpose string, keepHash []byte, expectedUser int64) error {
	hash, err := HashEmailToken(raw)
	if err != nil {
		return err
	}
	now := h.sessions.clock.Now()
	return store.InTx(ctx, h.db, func(tx *sqlx.Tx) error {
		s := NewStore(tx)
		token, e := s.ConsumeEmailToken(ctx, hash, purpose, now)
		if e != nil {
			return e
		}
		if expectedUser != 0 && token.UserID != expectedUser {
			return ErrInvalidEmailToken
		}
		email, _, role, password, e := s.LockedEmailUser(ctx, token.UserID)
		if e != nil {
			return ErrInvalidEmailToken
		}
		if role != auth.RoleUser || !password {
			return ErrInvalidEmailToken
		}
		if purpose == "verify" {
			if strings.ToLower(strings.TrimSpace(email)) != token.Email {
				return ErrInvalidEmailToken
			}
			return s.ConfirmEmail(ctx, token.UserID, token.Email)
		}
		if email == "" {
			return ErrInvalidEmailToken
		}
		return s.ChangeEmail(ctx, token.UserID, token.Email, now, keepHash)
	})
}
func (h *EmailHandler) verify(w http.ResponseWriter, r *http.Request) error {
	if err := h.available(w); err != nil {
		return err
	}
	var v proofRequest
	if err := httpx.DecodeJSON(w, r, &v); err != nil {
		return err
	}
	if err := validate.Struct(v); err != nil {
		return err
	}
	if ok, seconds := h.limits.Confirm(r); !ok {
		retryAfter(w, seconds)
		return httpx.NewError(429, "Слишком много попыток. Попробуйте позже")
	}
	err := h.consume(r.Context(), v.Token, "verify", nil, 0)
	if errors.Is(err, ErrInvalidEmailToken) {
		return httpx.BadRequest(badEmailProof)
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, 200, map[string]bool{"verified": true})
	return nil
}
func (h *EmailHandler) signIn(w http.ResponseWriter, r *http.Request) error {
	if err := h.available(w); err != nil {
		return err
	}
	var v loginRequest
	if err := httpx.DecodeJSON(w, r, &v); err != nil {
		return err
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	if err := validate.Struct(v); err != nil {
		return err
	}
	email, err := normalizedRequestEmail(v.Email)
	if err != nil {
		return httpx.BadRequest("Некорректный адрес электронной почты")
	}
	if ok, seconds := h.limits.SignIn(email, r); !ok {
		retryAfter(w, seconds)
		return httpx.NewError(429, "Слишком много попыток. Попробуйте позже")
	}
	id, err := NewStore(h.db).VerifiedEmailOwner(r.Context(), email)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if err != nil {
		auth.CheckPassword(h.dummyHash, v.Password)
		return httpx.NewError(401, badCredentials)
	}
	u, err := h.users.ByID(r.Context(), id)
	if err != nil {
		return err
	}
	valid := false
	if u.PasswordHash != "" {
		valid = auth.CheckPassword(u.PasswordHash, v.Password)
	} else if u.LegacyPasswordHash != "" {
		valid = auth.CheckLegacyPassword(u.LegacyPasswordHash, v.Password)
		if !valid {
			auth.CheckPassword(h.dummyHash, v.Password)
		}
		if valid && u.IsEmailConfirmed && u.Role == auth.RoleUser {
			hash, e := auth.HashPassword(v.Password)
			if e != nil {
				return e
			}
			if e = h.users.SetPasswordHash(r.Context(), id, hash); e != nil {
				return e
			}
		}
	} else {
		auth.CheckPassword(h.dummyHash, v.Password)
	}
	if !valid || !u.IsEmailConfirmed || u.Role != auth.RoleUser || u.Email == nil || strings.ToLower(strings.TrimSpace(*u.Email)) != email {
		return httpx.NewError(401, badCredentials)
	}
	token, exp, err := h.sessions.CreateSession(r.Context(), id, "password", "web")
	if err != nil {
		return err
	}
	httpx.JSON(w, 200, struct {
		User      user.Response `json:"user"`
		Token     string        `json:"token"`
		ExpiresAt time.Time     `json:"expiresAt"`
	}{u.Response(), token, exp})
	return nil
}
func (h *EmailHandler) recent(r *http.Request) (Session, []byte, error) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer cs_") {
		return Session{}, nil, httpx.Unauthorized()
	}
	raw := strings.TrimPrefix(values[0], "Bearer ")
	s, err := h.sessions.Authenticate(r.Context(), raw)
	if err != nil {
		return Session{}, nil, httpx.Unauthorized()
	}
	if s.Surface != "web" || h.sessions.clock.Now().Sub(s.AuthenticatedAt) > 5*time.Minute || h.sessions.clock.Now().Before(s.AuthenticatedAt) {
		return Session{}, nil, httpx.NewError(403, "Подтвердите вход повторно")
	}
	hash, _ := HashToken(raw)
	return s, hash, nil
}
func (h *EmailHandler) change(w http.ResponseWriter, r *http.Request) error {
	if err := h.available(w); err != nil {
		return err
	}
	s, _, err := h.recent(r)
	if err != nil {
		return err
	}
	var v emailRequest
	if err = httpx.DecodeJSON(w, r, &v); err != nil {
		return err
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	if err = validate.Struct(v); err != nil {
		return err
	}
	email, err := normalizedRequestEmail(v.Email)
	if err != nil {
		return httpx.BadRequest("Некорректный адрес электронной почты")
	}
	if !h.limits.Send(email, r) {
		generic(w)
		return nil
	}
	u, err := h.users.ByID(r.Context(), s.UserID)
	if err != nil {
		return err
	}
	if u.Role != auth.RoleUser || u.PasswordHash == "" || u.Email == nil || !u.IsEmailConfirmed {
		return httpx.NewError(403, "Подтвердите вход повторно")
	}
	if _, err = NewStore(h.db).VerifiedEmailOwner(r.Context(), email); err == nil {
		generic(w)
		return nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	_, err = h.sessions.IssueEmailToken(r.Context(), s.UserID, "change-email", email)
	if err != nil && !errors.Is(err, ErrEmailUnavailable) {
		return err
	}
	generic(w)
	return nil
}
func (h *EmailHandler) confirmChange(w http.ResponseWriter, r *http.Request) error {
	if err := h.available(w); err != nil {
		return err
	}
	s, hash, err := h.recent(r)
	if err != nil {
		return err
	}
	var v proofRequest
	if err = httpx.DecodeJSON(w, r, &v); err != nil {
		return err
	}
	if err = validate.Struct(v); err != nil {
		return err
	}
	if ok, seconds := h.limits.Confirm(r); !ok {
		retryAfter(w, seconds)
		return httpx.NewError(429, "Слишком много попыток. Попробуйте позже")
	}
	err = h.consume(r.Context(), v.Token, "change-email", hash, s.UserID)
	if store.IsUniqueViolation(err) {
		return httpx.NewError(409, "Способ входа уже используется")
	}
	if errors.Is(err, ErrInvalidEmailToken) {
		return httpx.BadRequest(badEmailProof)
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, 200, map[string]bool{"verified": true})
	return nil
}
