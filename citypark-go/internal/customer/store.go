package customer

import (
	"context"
	"errors"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
)

var ErrMergedAccount = errors.New("merged customer account")

type Identity struct {
	UserID   int64  `db:"user_id"`
	Provider string `db:"provider"`
	Subject  string `db:"subject"`
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store { return &Store{db: db} }

// LockAccount serializes all account credential and session mutations with merges.
func (s *Store) LockAccount(ctx context.Context, id int64) error {
	var merged bool
	err := s.db.GetContext(ctx, &merged, `SELECT merged_into_user_id IS NOT NULL FROM users WHERE id=$1 FOR UPDATE`, id)
	if err != nil {
		return store.NotFound(err)
	}
	if merged {
		return ErrMergedAccount
	}
	return nil
}

func (s *Store) IdentityOwner(ctx context.Context, provider, subject string) (int64, error) {
	var id int64
	err := s.db.GetContext(ctx, &id, `SELECT user_id FROM customer_identities WHERE provider = $1 AND subject = $2`, provider, subject)
	if err != nil {
		return 0, store.NotFound(err)
	}
	return id, nil
}

func (s *Store) AddIdentity(ctx context.Context, identity Identity) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO customer_identities (user_id, provider, subject) VALUES ($1, $2, $3)`, identity.UserID, identity.Provider, identity.Subject)
	return err
}

func (s *Store) VerifiedEmailOwner(ctx context.Context, email string) (int64, error) {
	return s.IdentityOwner(ctx, "email", email)
}
func (s *Store) LockedEmailUser(ctx context.Context, id int64) (string, bool, string, bool, error) {
	var v struct {
		Email     *string `db:"email"`
		Confirmed bool    `db:"is_email_confirmed"`
		Role      string  `db:"role"`
		Password  bool    `db:"has_password"`
		Merged    bool    `db:"merged"`
	}
	err := s.db.GetContext(ctx, &v, `SELECT email,is_email_confirmed,role,(password_hash<>'' OR legacy_password_hash<>'') AS has_password,merged_into_user_id IS NOT NULL AS merged FROM users WHERE id=$1 FOR UPDATE`, id)
	if err != nil {
		return "", false, "", false, store.NotFound(err)
	}
	if v.Merged {
		return "", false, "", false, ErrMergedAccount
	}
	if v.Email == nil {
		return "", v.Confirmed, v.Role, v.Password, nil
	}
	return *v.Email, v.Confirmed, v.Role, v.Password, nil
}
func (s *Store) ConfirmEmail(ctx context.Context, id int64, email string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE users SET is_email_confirmed=true,updated_at=now() WHERE id=$1 AND lower(btrim(email))=$2`, id, email)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInvalidEmailToken
	}
	return nil
}
func (s *Store) ChangeEmail(ctx context.Context, id int64, email string, now time.Time, keepHash []byte) error {
	result, err := s.db.ExecContext(ctx, `UPDATE customer_identities SET subject=$2 WHERE user_id=$1 AND provider='email'`, id, email)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInvalidEmailToken
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE users SET email=$2,is_email_confirmed=true,updated_at=now() WHERE id=$1`, id, email); err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE customer_email_tokens SET consumed_at=$2 WHERE user_id=$1 AND consumed_at IS NULL`, id, now); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE customer_sessions SET revoked_at=$3 WHERE user_id=$1 AND token_hash<>$2 AND revoked_at IS NULL`, id, keepHash, now)
	return err
}
