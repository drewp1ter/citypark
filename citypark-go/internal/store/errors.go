package store

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned by stores when a row does not exist.
var ErrNotFound = errors.New("not found")

// NotFound converts sql.ErrNoRows to ErrNotFound and returns every other error unchanged.
// Stores wrap single-row reads with it: return store.NotFound(err).
func NotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func IsUniqueViolation(err error) bool { return hasPgCode(err, "23505") }
func IsFKViolation(err error) bool     { return hasPgCode(err, "23503") }
func IsCheckViolation(err error) bool  { return hasPgCode(err, "23514") }

func hasPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
