package customer

import (
	"context"
	"errors"
	"time"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

var ErrMergeConflict = errors.New("customer merge conflict")
var ErrMergeForbidden = errors.New("customer merge forbidden")

type mergeUser struct {
	ID         int64   `db:"id"`
	Role       string  `db:"role"`
	MergedInto *int64  `db:"merged_into_user_id"`
	Email      *string `db:"email"`
	Password   string  `db:"password_hash"`
	Legacy     string  `db:"legacy_password_hash"`
	Confirmed  bool    `db:"is_email_confirmed"`
}

// MergeAccounts moves authenticated ownership into the website account.
// The caller must establish consent from both parties before calling it.
func (s *Service) MergeAccounts(ctx context.Context, sourceID, destinationID int64) error {
	return store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		return s.MergeAccountsInTx(ctx, tx, sourceID, destinationID)
	})
}

// MergeAccountsInTx participates in the caller's transaction, including pairing consumption.
func (s *Service) MergeAccountsInTx(ctx context.Context, tx *sqlx.Tx, sourceID, destinationID int64) error {
	if sourceID <= 0 || destinationID <= 0 || sourceID == destinationID {
		return ErrMergeForbidden
	}
	users := []mergeUser{}
	if err := tx.SelectContext(ctx, &users, `SELECT id,role,merged_into_user_id,email,password_hash,legacy_password_hash,is_email_confirmed FROM users WHERE id=ANY($1) ORDER BY id FOR UPDATE`, []int64{sourceID, destinationID}); err != nil {
		return err
	}
	if len(users) != 2 {
		return ErrMergeForbidden
	}
	var source, destination mergeUser
	for _, u := range users {
		if u.ID == sourceID {
			source = u
		} else {
			destination = u
		}
	}
	if source.Role != "user" || destination.Role != "user" || source.MergedInto != nil || destination.MergedInto != nil {
		return ErrMergeForbidden
	}
	// Child rows are locked in one order after the two account rows.
	for _, query := range []string{
		`SELECT user_id FROM customer_identities WHERE user_id=ANY($1) ORDER BY user_id,provider FOR UPDATE`,
		`SELECT user_id FROM customer_email_tokens WHERE user_id=ANY($1) ORDER BY user_id,token_hash FOR UPDATE`,
		`SELECT user_id FROM customer_oauth_attempts WHERE user_id=ANY($1) ORDER BY user_id,state_hash FOR UPDATE`,
		`SELECT user_id FROM customer_sessions WHERE user_id=ANY($1) ORDER BY user_id,token_hash FOR UPDATE`,
	} {
		rows, err := tx.QueryxContext(ctx, query, []int64{sourceID, destinationID})
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	identities := []Identity{}
	if err := tx.SelectContext(ctx, &identities, `SELECT user_id,provider,subject FROM customer_identities WHERE user_id=ANY($1)`, []int64{sourceID, destinationID}); err != nil {
		return err
	}
	providers := map[string]int64{}
	for _, identity := range identities {
		if owner, exists := providers[identity.Provider]; exists && owner != identity.UserID {
			return ErrMergeConflict
		}
		providers[identity.Provider] = identity.UserID
	}
	if (source.Password != "" || source.Legacy != "") && (destination.Password != "" || destination.Legacy != "") {
		return ErrMergeConflict
	}
	if (source.Password != "" || source.Legacy != "") && source.Email == nil {
		return ErrMergeConflict
	}
	if source.Email != nil && destination.Email != nil {
		return ErrMergeConflict
	}
	// Release the unique email reservation before assigning it to the destination.
	if _, err := tx.ExecContext(ctx, `UPDATE users SET merged_into_user_id=$2,email=NULL,password_hash='',legacy_password_hash='',is_email_confirmed=false,updated_at=now() WHERE id=$1`, sourceID, destinationID); err != nil {
		return err
	}
	if source.Email != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET email=$2,password_hash=$3,legacy_password_hash=$4,is_email_confirmed=$5,updated_at=now() WHERE id=$1`, destinationID, *source.Email, source.Password, source.Legacy, source.Confirmed); err != nil {
			if store.IsUniqueViolation(err) {
				return ErrMergeConflict
			}
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_identities SET user_id=$2 WHERE user_id=$1`, sourceID, destinationID); err != nil {
		if store.IsUniqueViolation(err) {
			return ErrMergeConflict
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE orders SET account_id=$2,updated_at=now() WHERE account_id=$1`, sourceID, destinationID); err != nil {
		return err
	}
	if err := address.NewStore(tx).MergeEntries(ctx, sourceID, destinationID); err != nil {
		return err
	}

	now := s.clock.Now()
	return revokeMergedProofs(ctx, tx, sourceID, now)
}

func revokeMergedProofs(ctx context.Context, tx *sqlx.Tx, sourceID int64, now time.Time) error {
	for _, query := range []string{
		`UPDATE customer_email_tokens SET consumed_at=$2 WHERE user_id=$1 AND consumed_at IS NULL`,
		`UPDATE customer_oauth_attempts SET consumed_at=$2,pkce_verifier='' WHERE user_id=$1 AND consumed_at IS NULL`,
		`UPDATE customer_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL`,
	} {
		if _, err := tx.ExecContext(ctx, query, sourceID, now); err != nil {
			return err
		}
	}
	return nil
}
