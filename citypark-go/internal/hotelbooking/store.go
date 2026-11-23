package hotelbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

type Store struct {
	db            store.DBTX
	queueTelegram bool
}

func NewStore(db store.DBTX) *Store             { return &Store{db: db} }
func NewStoreWithTelegram(db store.DBTX) *Store { return &Store{db: db, queueTelegram: true} }

func (s *Store) transaction(ctx context.Context, fn func(*Store) error) error {
	if db, ok := s.db.(*sqlx.DB); ok {
		return store.InTx(ctx, db, func(tx *sqlx.Tx) error { return fn(NewStore(tx)) })
	}
	return fn(s)
}

type quoteRow struct {
	ID          string        `db:"id"`
	ContextHash string        `db:"context_hash"`
	RoomTypeID  string        `db:"room_type_id"`
	CheckIn     time.Time     `db:"check_in"`
	CheckOut    time.Time     `db:"check_out"`
	Guests      int           `db:"guests"`
	Nightly     money.Kopecks `db:"nightly_price_kopecks"`
	Total       money.Kopecks `db:"total_kopecks"`
	ExpiresAt   time.Time     `db:"expires_at"`
}

func (q quoteRow) request(key string, guest Guest) CreateRequest {
	nights := int((q.CheckOut.Unix() - q.CheckIn.Unix()) / 86400)
	return CreateRequest{key, q.RoomTypeID, q.CheckIn.Format("2006-01-02"), q.CheckOut.Format("2006-01-02"), q.Guests, guest, Price{"RUB", nights, q.Nightly, q.Total}}
}
func (s *Store) saveQuote(ctx context.Context, q QuoteResponse, hash string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO hotel_booking_quotes(id,context_hash,room_type_id,check_in,check_out,guests,nightly_price_kopecks,total_kopecks,expires_at,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, q.QuoteID, hash, q.RoomTypeID, q.CheckIn, q.CheckOut, q.Guests, int64(q.Price.PricePerNight), int64(q.Price.Total), q.ExpiresAt, now)
	return err
}
func (s *Store) quote(ctx context.Context, id, hash string) (quoteRow, error) {
	var q quoteRow
	err := s.db.GetContext(ctx, &q, `SELECT id,context_hash,room_type_id,check_in,check_out,guests,nightly_price_kopecks,total_kopecks,expires_at FROM hotel_booking_quotes WHERE id=$1 AND context_hash=$2`, id, hash)
	return q, err
}

type attemptRow struct {
	Key         string         `db:"key"`
	QuoteID     string         `db:"quote_id"`
	ContextHash string         `db:"context_hash"`
	Canonical   string         `db:"canonical_request"`
	State       string         `db:"state"`
	Result      []byte         `db:"result"`
	Rejection   string         `db:"rejection_code"`
	EmailState  string         `db:"email_state"`
	Claim       sql.NullString `db:"claim"`
}

const attemptColumns = `a.key,a.quote_id,a.context_hash,a.canonical_request,a.state,a.result,a.rejection_code,a.claim,COALESCE(o.state,'pending') AS email_state`

func (s *Store) attempt(ctx context.Context, key string) (attemptRow, error) {
	var a attemptRow
	err := s.db.GetContext(ctx, &a, `SELECT `+attemptColumns+` FROM hotel_booking_attempts a LEFT JOIN hotel_booking_email_outbox o ON o.attempt_key=a.key WHERE a.key=$1`, key)
	return a, err
}

// begin persists identity and both creation quotas atomically. ON CONFLICT waits
// for a competing insert before returning the saved attempt; no HTTP runs in a transaction.
func (s *Store) begin(ctx context.Context, req PublicCreateRequest, canonical, hash, ip, contact, claim string, now time.Time) (attemptRow, bool, error) {
	var a attemptRow
	fresh := false
	err := s.transaction(ctx, func(tx *Store) error {
		res, err := tx.db.ExecContext(ctx, `INSERT INTO hotel_booking_attempts(key,quote_id,context_hash,canonical_request,state,claim,lease_until,next_attempt_at,created_at) VALUES($1,$2,$3,$4,'pending',$5,$6,$7,$8) ON CONFLICT(key) DO NOTHING`, req.IdempotencyKey, req.QuoteID, hash, canonical, claim, now.Add(time.Minute), now.Add(30*time.Second), now)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		fresh = n == 1
		if fresh {
			if err = tx.limit(ctx, "create-ip", ip, now, time.Hour, 5); err != nil {
				return err
			}
			if err = tx.limit(ctx, "create-contact", contact, now, time.Hour, 3); err != nil {
				return err
			}
		}
		a, err = tx.attempt(ctx, req.IdempotencyKey)
		return err
	})
	return a, fresh, err
}
func (s *Store) limit(ctx context.Context, scope, hash string, now time.Time, window time.Duration, max int) error {
	var count int
	err := s.db.GetContext(ctx, &count, `INSERT INTO hotel_booking_rate_limits(scope,hash,window_start,counter) VALUES($1,$2,$3,1) ON CONFLICT(scope,hash,window_start) DO UPDATE SET counter=hotel_booking_rate_limits.counter+1,updated_at=now() WHERE hotel_booking_rate_limits.counter<$4 RETURNING counter`, scope, hash, now.Truncate(window), max)
	if errors.Is(err, sql.ErrNoRows) {
		return failure("RATE_LIMITED")
	}
	return err
}

// finish makes confirmation and its uniquely keyed outbox entry one local commit.
func (s *Store) finish(ctx context.Context, a attemptRow, r Reservation, now time.Time) error {
	raw, _ := json.Marshal(r)
	return s.transaction(ctx, func(tx *Store) error {
		var key string
		err := tx.db.GetContext(ctx, &key, `UPDATE hotel_booking_attempts SET state='confirmed',result=$3,claim=NULL,lease_until=NULL,updated_at=now() WHERE key=$1 AND state='pending' AND claim=$2 RETURNING key`, a.Key, a.Claim.String, string(raw))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = tx.db.ExecContext(ctx, `INSERT INTO hotel_booking_email_outbox(attempt_key,next_attempt_at) VALUES($1,$2) ON CONFLICT(attempt_key) DO NOTHING`, key, now)
		if err != nil || !s.queueTelegram {
			return err
		}
		_, err = tx.db.ExecContext(ctx, `INSERT INTO hotel_booking_telegram_outbox(attempt_key,next_attempt_at) VALUES($1,$2) ON CONFLICT(attempt_key) DO NOTHING`, key, now)
		return err
	})
}
func (s *Store) reject(ctx context.Context, a attemptRow, code string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE hotel_booking_attempts SET state='rejected',rejection_code=$3,claim=NULL,lease_until=NULL,updated_at=now() WHERE key=$1 AND state='pending' AND claim=$2`, a.Key, a.Claim.String, code)
	return err
}
func (s *Store) release(ctx context.Context, a attemptRow, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE hotel_booking_attempts SET claim=NULL,lease_until=NULL,next_attempt_at=$3,updated_at=now() WHERE key=$1 AND state='pending' AND claim=$2`, a.Key, a.Claim.String, now.Add(30*time.Second))
	return err
}

// Claim uses SKIP LOCKED and a fencing token so different processes cannot
// acknowledge one another's expired leases.
func (s *Store) claimRecovery(ctx context.Context, now time.Time, claim string) (attemptRow, error) {
	var key string
	err := s.db.GetContext(ctx, &key, `UPDATE hotel_booking_attempts SET claim=$2,lease_until=$3,updated_at=now() WHERE key=(SELECT key FROM hotel_booking_attempts WHERE state='pending' AND next_attempt_at<=$1 AND (lease_until IS NULL OR lease_until<=$1) ORDER BY next_attempt_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING key`, now, claim, now.Add(time.Minute))
	if err != nil {
		return attemptRow{}, err
	}
	return s.attempt(ctx, key)
}

type emailRow struct {
	Key       string `db:"attempt_key"`
	Attempts  int    `db:"attempts"`
	Canonical string `db:"canonical_request"`
	Result    []byte `db:"result"`
}

func (s *Store) claimEmail(ctx context.Context, now time.Time, claim string) (emailRow, error) {
	var row emailRow
	err := s.db.GetContext(ctx, &row, `WITH claimed AS (UPDATE hotel_booking_email_outbox SET claim=$2,lease_until=$3,attempts=attempts+1,updated_at=now() WHERE attempt_key=(SELECT attempt_key FROM hotel_booking_email_outbox WHERE state='pending' AND next_attempt_at<=$1 AND (lease_until IS NULL OR lease_until<=$1) ORDER BY next_attempt_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING attempt_key,attempts) SELECT c.attempt_key,c.attempts,a.canonical_request,a.result FROM claimed c JOIN hotel_booking_attempts a ON a.key=c.attempt_key`, now, claim, now.Add(time.Minute))
	return row, err
}
func (s *Store) emailResult(ctx context.Context, key, claim string, sent bool, next time.Time) error {
	state := "pending"
	if sent {
		state = "sent"
	}
	_, err := s.db.ExecContext(ctx, `UPDATE hotel_booking_email_outbox SET state=$3,next_attempt_at=$4,claim=NULL,lease_until=NULL,updated_at=now() WHERE attempt_key=$1 AND claim=$2`, key, claim, state, next)
	return err
}
