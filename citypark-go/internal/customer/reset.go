package customer

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
	"github.com/jmoiron/sqlx"
)

const genericRecoveryMessage = "Если аккаунт найден, письмо для восстановления отправлено"

type resetRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

func (h *EmailHandler) forgotPassword(w http.ResponseWriter, r *http.Request) error {
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
	if h.limits.Send(email, r) {
		id, err := NewStore(h.db).VerifiedEmailOwner(r.Context(), email)
		if err == nil {
			u, err := h.users.ByID(r.Context(), id)
			if err == nil && u.Role == auth.RoleUser && u.IsEmailConfirmed && u.Email != nil &&
				strings.EqualFold(strings.TrimSpace(*u.Email), email) && (u.PasswordHash != "" || u.LegacyPasswordHash != "") {
				_, _ = h.sessions.IssueEmailToken(r.Context(), id, "reset", email)
			}
		}
	}
	httpx.JSON(w, http.StatusAccepted, map[string]string{"message": genericRecoveryMessage})
	return nil
}

func (h *EmailHandler) resetPassword(w http.ResponseWriter, r *http.Request) error {
	if err := h.available(w); err != nil {
		return err
	}
	var v resetRequest
	if err := httpx.DecodeJSON(w, r, &v); err != nil {
		return err
	}
	if err := validate.Struct(v); err != nil {
		return err
	}
	if ok, seconds := h.limits.Reset(r); !ok {
		retryAfter(w, seconds)
		return httpx.NewError(http.StatusTooManyRequests, "Слишком много попыток. Попробуйте позже")
	}
	err := h.sessions.ResetPassword(r.Context(), v.Token, v.Password)
	if errors.Is(err, ErrInvalidEmailToken) {
		return httpx.BadRequest(badEmailProof)
	}
	if err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

// ResetPassword consumes a reset proof and revokes every session atomically.
func (s *Service) ResetPassword(ctx context.Context, rawToken, password string) error {
	hash, err := HashEmailToken(rawToken)
	if err != nil {
		return err
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	return store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		tokens := NewStore(tx)
		id, err := tokens.EmailTokenOwner(ctx, hash, "reset", now)
		if err != nil {
			return err
		}
		email, confirmed, role, hasPassword, err := tokens.LockedEmailUser(ctx, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return ErrInvalidEmailToken
			}
			return err
		}
		token, err := tokens.ConsumeEmailToken(ctx, hash, "reset", now)
		if err != nil {
			return err
		}
		owner, err := tokens.VerifiedEmailOwner(ctx, token.Email)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return err
		}
		if role != auth.RoleUser || !confirmed || !hasPassword ||
			strings.ToLower(strings.TrimSpace(email)) != token.Email || owner != id {
			return ErrInvalidEmailToken
		}
		if _, err = tx.ExecContext(ctx, `UPDATE users SET password_hash=$2,legacy_password_hash='',updated_at=now() WHERE id=$1`, id, passwordHash); err != nil {
			return err
		}
		if err = tokens.InvalidateEmailTokens(ctx, id, "reset", now); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE customer_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL`, id, now)
		return err
	})
}
