package user

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

// AuthHandler serves sign-up and sign-in.
type AuthHandler struct {
	store  *Store
	tokens *auth.Tokens
}

func NewAuthHandler(s *Store, tokens *auth.Tokens) *AuthHandler {
	return &AuthHandler{store: s, tokens: tokens}
}

// Routes registers POST /auth/sign-up and POST /auth/sign-in.
func (h *AuthHandler) Routes(r chi.Router) {
	r.Post("/auth/sign-up", httpx.Handle(h.signUp))
	r.Post("/auth/sign-in", httpx.Handle(h.signIn))
}

type signUpRequest struct {
	Name     string `json:"name" validate:"required,max=80"`
	Email    string `json:"email" validate:"omitempty,email"`
	Phone    string `json:"phone" validate:"required,ruphone"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

type signInRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type signInResponse struct {
	Response
	Token string `json:"token"`
}

func (h *AuthHandler) signUp(w http.ResponseWriter, r *http.Request) error {
	var req signUpRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := validate.Struct(req); err != nil {
		return err
	}

	phone, _ := validate.NormalizePhone(req.Phone)
	var email *string
	if e := normalizeEmail(req.Email); e != "" {
		email = &e
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return err
	}
	_, err = h.store.Create(r.Context(), CreateParams{
		Name:         req.Name,
		Email:        email,
		Phone:        phone,
		PasswordHash: hash,
		Role:         auth.RoleUser,
	})
	// A taken email or phone is ignored on purpose: the response must not reveal
	// whether an account exists.
	if err != nil && !store.IsUniqueViolation(err) {
		return err
	}

	httpx.JSON(w, http.StatusCreated, struct{}{})
	return nil
}

func (h *AuthHandler) signIn(w http.ResponseWriter, r *http.Request) error {
	var req signInRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := validate.Struct(req); err != nil {
		return err
	}

	ctx := r.Context()
	u, err := h.store.ByEmail(ctx, normalizeEmail(req.Email))
	if errors.Is(err, store.ErrNotFound) {
		return errBadCredentials()
	}
	if err != nil {
		return err
	}

	switch {
	case u.PasswordHash != "":
		if !auth.CheckPassword(u.PasswordHash, req.Password) {
			return errBadCredentials()
		}
	case u.LegacyPasswordHash != "":
		if !auth.CheckLegacyPassword(u.LegacyPasswordHash, req.Password) {
			return errBadCredentials()
		}
		// Upgrade the account to bcrypt; this also clears the legacy hash.
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			return err
		}
		if err := h.store.SetPasswordHash(ctx, u.ID, hash); err != nil {
			return err
		}
	default:
		return errBadCredentials()
	}

	token, err := h.tokens.Issue(u.ID, u.Role)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, signInResponse{Response: u.Response(), Token: token})
	return nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// errBadCredentials is the single answer for an unknown email and a wrong password.
func errBadCredentials() *httpx.Error {
	return httpx.NewError(http.StatusUnauthorized, "неверный логин или пароль.")
}
