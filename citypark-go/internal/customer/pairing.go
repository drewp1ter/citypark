package customer

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

var ErrPairingInvalid = errors.New("invalid or expired pairing")
var ErrPairingNotFound = errors.New("pairing not found")
var ErrPairingSurface = errors.New("wrong pairing surface")
var ErrPairingSameAccount = errors.New("pairing same account")
var ErrPairingStale = errors.New("pairing session is stale")

type PairingStatus struct {
	ID                   string    `json:"id"`
	Status               string    `json:"status"`
	SourceConfirmed      bool      `json:"sourceConfirmed"`
	DestinationConfirmed bool      `json:"destinationConfirmed"`
	OtherName            string    `json:"otherName"`
	ExpiresAt            time.Time `json:"expiresAt"`
}

type PairingCreated struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type pairingRow struct {
	ID                     string     `db:"id"`
	SourceID               int64      `db:"source_id"`
	DestinationID          *int64     `db:"destination_id"`
	SourceHash             []byte     `db:"source_session_hash"`
	DestinationHash        []byte     `db:"destination_session_hash"`
	ExpiresAt              time.Time  `db:"expires_at"`
	RedeemedAt             *time.Time `db:"redeemed_at"`
	SourceConfirmedAt      *time.Time `db:"source_confirmed_at"`
	DestinationConfirmedAt *time.Time `db:"destination_confirmed_at"`
	CompletedAt            *time.Time `db:"completed_at"`
	CanceledAt             *time.Time `db:"canceled_at"`
}

func (s *Service) randomPairingValue() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(s.random, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) pairingSession(ctx context.Context, tx *sqlx.Tx, session Session, recent bool) error {
	now := s.clock.Now()
	var stored Session
	err := tx.GetContext(ctx, &stored, `SELECT user_id,token_hash,authenticated_at,expires_at,method,surface FROM customer_sessions WHERE token_hash=$1 AND user_id=$2 AND revoked_at IS NULL AND expires_at>$3`, session.TokenHash, session.UserID, now)
	if errors.Is(store.NotFound(err), store.ErrNotFound) {
		return ErrInvalidSession
	}
	if err != nil {
		return err
	}
	if stored.Surface != session.Surface || stored.Method != session.Method || !stored.AuthenticatedAt.Equal(session.AuthenticatedAt) {
		return ErrInvalidSession
	}
	if recent && (now.Before(stored.AuthenticatedAt) || now.Sub(stored.AuthenticatedAt) > 5*time.Minute) {
		return ErrPairingStale
	}
	return nil
}

func lockPairingUsers(ctx context.Context, tx *sqlx.Tx, sourceID int64, destinationID *int64) error {
	ids := []int64{sourceID}
	if destinationID != nil && *destinationID != sourceID {
		ids = append(ids, *destinationID)
	}
	rows, err := tx.QueryxContext(ctx, `SELECT id FROM users WHERE id=ANY($1) ORDER BY id FOR UPDATE`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Service) CreatePairing(ctx context.Context, session Session) (PairingCreated, error) {
	if session.Surface != "mini" || session.Method != "telegram" {
		return PairingCreated{}, ErrPairingSurface
	}
	id, err := s.randomPairingValue()
	if err != nil {
		return PairingCreated{}, err
	}
	code, err := s.randomPairingValue()
	if err != nil {
		return PairingCreated{}, err
	}
	now := s.clock.Now()
	expires := now.Add(5 * time.Minute)
	hash := sha256.Sum256([]byte(code))
	err = store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		if err := lockPairingUsers(ctx, tx, session.UserID, nil); err != nil {
			return err
		}
		if err := s.pairingSession(ctx, tx, session, true); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE customer_pairings SET canceled_at=$2 WHERE source_id=$1 AND completed_at IS NULL AND canceled_at IS NULL`, session.UserID, now); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO customer_pairings(id,code_hash,source_id,source_session_hash,expires_at) VALUES($1,$2,$3,$4,$5)`, id, hash[:], session.UserID, session.TokenHash, expires)
		return err
	})
	return PairingCreated{ID: id, Code: code, ExpiresAt: expires}, err
}

func readPairing(ctx context.Context, db store.DBTX, id string, locked bool) (pairingRow, error) {
	var row pairingRow
	query := `SELECT id,source_id,destination_id,source_session_hash,destination_session_hash,expires_at,redeemed_at,source_confirmed_at,destination_confirmed_at,completed_at,canceled_at FROM customer_pairings WHERE id=$1`
	if locked {
		query += ` FOR UPDATE`
	}
	err := db.GetContext(ctx, &row, query, id)
	if errors.Is(store.NotFound(err), store.ErrNotFound) {
		return row, ErrPairingNotFound
	}
	return row, err
}

func pairingSide(row pairingRow, session Session) (string, error) {
	if row.SourceID == session.UserID && string(row.SourceHash) == string(session.TokenHash) {
		if session.Surface != "mini" {
			return "", ErrPairingSurface
		}
		return "source", nil
	}
	if row.DestinationID != nil && *row.DestinationID == session.UserID && string(row.DestinationHash) == string(session.TokenHash) {
		if session.Surface != "web" {
			return "", ErrPairingSurface
		}
		return "destination", nil
	}
	return "", ErrPairingNotFound
}

func (s *Service) pairingStatus(ctx context.Context, tx *sqlx.Tx, row pairingRow, side string) (PairingStatus, error) {
	status := "pending"
	if row.CompletedAt != nil {
		status = "completed"
	} else if row.CanceledAt != nil {
		status = "canceled"
	}
	otherID := row.SourceID
	if side == "source" {
		if row.DestinationID == nil {
			return PairingStatus{ID: row.ID, Status: status, SourceConfirmed: row.SourceConfirmedAt != nil, DestinationConfirmed: row.DestinationConfirmedAt != nil, ExpiresAt: row.ExpiresAt}, nil
		}
		otherID = *row.DestinationID
	}
	var name string
	if err := tx.GetContext(ctx, &name, `SELECT name FROM users WHERE id=$1`, otherID); err != nil {
		return PairingStatus{}, err
	}
	return PairingStatus{ID: row.ID, Status: status, SourceConfirmed: row.SourceConfirmedAt != nil, DestinationConfirmed: row.DestinationConfirmedAt != nil, OtherName: name, ExpiresAt: row.ExpiresAt}, nil
}

func (s *Service) RedeemPairing(ctx context.Context, code string, session Session) (PairingStatus, error) {
	if session.Surface != "web" {
		return PairingStatus{}, ErrPairingSurface
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(code)
	if err != nil || len(decoded) != 16 {
		return PairingStatus{}, ErrPairingInvalid
	}
	hash := sha256.Sum256([]byte(code))
	var result PairingStatus
	err = store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		var id string
		var sourceID int64
		err := tx.QueryRowxContext(ctx, `SELECT id,source_id FROM customer_pairings WHERE code_hash=$1`, hash[:]).Scan(&id, &sourceID)
		if errors.Is(store.NotFound(err), store.ErrNotFound) {
			return ErrPairingInvalid
		}
		if err != nil {
			return err
		}
		if err := lockPairingUsers(ctx, tx, sourceID, &session.UserID); err != nil {
			return err
		}
		row, err := readPairing(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if row.RedeemedAt != nil || row.CanceledAt != nil || !row.ExpiresAt.After(s.clock.Now()) {
			return ErrPairingInvalid
		}
		if err := s.pairingSession(ctx, tx, session, true); err != nil {
			return err
		}
		if row.SourceID == session.UserID {
			return ErrPairingSameAccount
		}
		if err := s.boundSession(ctx, tx, row.SourceHash, row.SourceID); err != nil {
			return ErrPairingInvalid
		}
		now := s.clock.Now()
		_, err = tx.ExecContext(ctx, `UPDATE customer_pairings SET destination_id=$2,destination_session_hash=$3,redeemed_at=$4 WHERE id=$1 AND redeemed_at IS NULL AND expires_at>$4 AND canceled_at IS NULL`, id, session.UserID, session.TokenHash, now)
		if err != nil {
			return err
		}
		row.DestinationID, row.DestinationHash, row.RedeemedAt = &session.UserID, session.TokenHash, &now
		result, err = s.pairingStatus(ctx, tx, row, "destination")
		return err
	})
	return result, err
}

func (s *Service) boundSession(ctx context.Context, tx *sqlx.Tx, hash []byte, userID int64) error {
	var session Session
	err := tx.GetContext(ctx, &session, `SELECT user_id,token_hash,authenticated_at,expires_at,method,surface FROM customer_sessions WHERE token_hash=$1 AND user_id=$2 AND revoked_at IS NULL AND expires_at>$3`, hash, userID, s.clock.Now())
	if err != nil {
		return err
	}
	return s.pairingSession(ctx, tx, session, true)
}

func (s *Service) inspectPairing(ctx context.Context, id string, session Session, action string) (PairingStatus, error) {
	var result PairingStatus
	err := store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		pre, err := readPairing(ctx, tx, id, false)
		if err != nil {
			return err
		}
		if err := lockPairingUsers(ctx, tx, pre.SourceID, pre.DestinationID); err != nil {
			return err
		}
		row, err := readPairing(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if (pre.DestinationID == nil) != (row.DestinationID == nil) || pre.DestinationID != nil && row.DestinationID != nil && *pre.DestinationID != *row.DestinationID {
			return ErrPairingInvalid
		}
		side, err := pairingSide(row, session)
		if err != nil {
			return err
		}
		if action != "cancel" && row.CompletedAt == nil && row.CanceledAt == nil && !row.ExpiresAt.After(s.clock.Now()) {
			return ErrPairingInvalid
		}
		if err := s.pairingSession(ctx, tx, session, action != "cancel"); err != nil {
			return err
		}
		if action == "confirm" && row.CompletedAt == nil {
			if row.CanceledAt != nil || row.RedeemedAt == nil || row.DestinationID == nil {
				return ErrPairingInvalid
			}
			if err := s.boundSession(ctx, tx, row.SourceHash, row.SourceID); err != nil {
				return ErrPairingInvalid
			}
			if err := s.boundSession(ctx, tx, row.DestinationHash, *row.DestinationID); err != nil {
				return ErrPairingInvalid
			}
			now := s.clock.Now()
			if side == "source" && row.SourceConfirmedAt == nil {
				if _, err := tx.ExecContext(ctx, `UPDATE customer_pairings SET source_confirmed_at=$2 WHERE id=$1 AND source_confirmed_at IS NULL`, id, now); err != nil {
					return err
				}
				row.SourceConfirmedAt = &now
			}
			if side == "destination" && row.DestinationConfirmedAt == nil {
				if _, err := tx.ExecContext(ctx, `UPDATE customer_pairings SET destination_confirmed_at=$2 WHERE id=$1 AND destination_confirmed_at IS NULL`, id, now); err != nil {
					return err
				}
				row.DestinationConfirmedAt = &now
			}
			if row.SourceConfirmedAt != nil && row.DestinationConfirmedAt != nil {
				if err := s.MergeAccountsInTx(ctx, tx, row.SourceID, *row.DestinationID); err != nil {
					return err
				}
				if _, err := tx.ExecContext(ctx, `UPDATE customer_pairings SET completed_at=$2 WHERE id=$1`, id, now); err != nil {
					return err
				}
				row.CompletedAt = &now
			}
		}
		if action == "cancel" && row.CompletedAt == nil && row.CanceledAt == nil {
			now := s.clock.Now()
			if _, err := tx.ExecContext(ctx, `UPDATE customer_pairings SET canceled_at=$2 WHERE id=$1 AND completed_at IS NULL AND canceled_at IS NULL`, id, now); err != nil {
				return err
			}
			row.CanceledAt = &now
		}
		result, err = s.pairingStatus(ctx, tx, row, side)
		return err
	})
	return result, err
}

func (s *Service) GetPairing(ctx context.Context, id string, session Session) (PairingStatus, error) {
	return s.inspectPairing(ctx, id, session, "get")
}
func (s *Service) ConfirmPairing(ctx context.Context, pairingID string, session Session) (PairingStatus, error) {
	return s.inspectPairing(ctx, pairingID, session, "confirm")
}
func (s *Service) CancelPairing(ctx context.Context, id string, session Session) error {
	_, err := s.inspectPairing(ctx, id, session, "cancel")
	return err
}
