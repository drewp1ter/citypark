package customer

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/mailer"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

var ErrInvalidEmailToken = errors.New("invalid email token")
var ErrEmailUnavailable = errors.New("email delivery unavailable")

type EmailToken struct {
	UserID  int64  `db:"user_id"`
	Purpose string `db:"purpose"`
	Email   string `db:"email"`
}

// ConfigureEmail supplies the transport and public origin. Call before serving requests.
func (s *Service) ConfigureEmail(m mailer.Mailer, publicOrigin string, random io.Reader, log *slog.Logger) {
	s.mailer, s.publicOrigin, s.log = m, publicOrigin, log
	if random != nil {
		s.random = random
	}
}

func tokenLifetime(purpose string) (time.Duration, bool) {
	switch purpose {
	case "verify", "change-email":
		return 24 * time.Hour, true
	case "reset":
		return time.Hour, true
	}
	return 0, false
}
func normalizedEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if strings.ContainsAny(email, "\r\n") {
		return "", ErrInvalidEmailToken
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", ErrInvalidEmailToken
	}
	return email, nil
}

func (s *Service) IssueEmailToken(ctx context.Context, userID int64, purpose, email string) (raw string, err error) {
	lifetime, ok := tokenLifetime(purpose)
	if !ok || userID <= 0 {
		return "", ErrInvalidEmailToken
	}
	email, err = normalizedEmail(email)
	if err != nil {
		return "", err
	}
	if s.mailer == nil || s.publicOrigin == "" {
		return "", ErrEmailUnavailable
	}
	proof := make([]byte, 32)
	if _, err := io.ReadFull(s.random, proof); err != nil {
		return "", ErrEmailUnavailable
	}
	raw = base64.RawURLEncoding.EncodeToString(proof)
	hash := sha256.Sum256(proof)
	now := s.clock.Now()
	err = store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		tokenStore := NewStore(tx)
		if err := tokenStore.InvalidateEmailTokens(ctx, userID, purpose, now); err != nil {
			return err
		}
		return tokenStore.InsertEmailToken(ctx, hash[:], EmailToken{UserID: userID, Purpose: purpose, Email: email}, now.Add(lifetime))
	})
	if err != nil {
		return "", err
	}
	paths := map[string]string{"verify": "/verify-email", "reset": "/reset-password", "change-email": "/change-email"}
	link := s.publicOrigin + paths[purpose] + "?token=" + url.QueryEscape(raw)
	// The proof appears only in the delivered message and the return value.
	subject := map[string]string{"verify": "Подтвердите адрес электронной почты", "reset": "Восстановление пароля", "change-email": "Подтвердите новый адрес электронной почты"}[purpose]
	if err := s.mailer.Send(ctx, email, subject, link); err != nil {
		_ = s.store.InvalidateEmailToken(ctx, hash[:], s.clock.Now())
		if s.log != nil {
			s.log.Error("email delivery failed", "purpose", purpose, "operation_id", fmt.Sprintf("%x", hash[:4]))
		}
		return "", ErrEmailUnavailable
	}
	return raw, nil
}

func (s *Store) InvalidateEmailTokens(ctx context.Context, userID int64, purpose string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE customer_email_tokens SET consumed_at=$3 WHERE user_id=$1 AND purpose=$2 AND consumed_at IS NULL`, userID, purpose, now)
	return err
}
func (s *Store) InsertEmailToken(ctx context.Context, hash []byte, token EmailToken, expires time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO customer_email_tokens(token_hash,user_id,purpose,email,expires_at) VALUES ($1,$2,$3,$4,$5)`, hash, token.UserID, token.Purpose, token.Email, expires)
	return err
}
func (s *Store) InvalidateEmailToken(ctx context.Context, hash []byte, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE customer_email_tokens SET consumed_at=$2 WHERE token_hash=$1 AND consumed_at IS NULL`, hash, now)
	return err
}

// ConsumeEmailToken must be called on a Store created from the caller's transaction.
func (s *Store) ConsumeEmailToken(ctx context.Context, hash []byte, purpose string, now time.Time) (EmailToken, error) {
	var token EmailToken
	if len(hash) != 32 {
		return token, ErrInvalidEmailToken
	}
	err := s.db.GetContext(ctx, &token, `UPDATE customer_email_tokens SET consumed_at=$3 WHERE token_hash=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at>$3 RETURNING user_id,purpose,email`, hash, purpose, now)
	if errors.Is(store.NotFound(err), store.ErrNotFound) {
		return EmailToken{}, ErrInvalidEmailToken
	}
	return token, err
}
func HashEmailToken(raw string) ([]byte, error) {
	if len(raw) != 43 {
		return nil, ErrInvalidEmailToken
	}
	proof, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil || len(proof) != 32 {
		return nil, ErrInvalidEmailToken
	}
	hash := sha256.Sum256(proof)
	return hash[:], nil
}
