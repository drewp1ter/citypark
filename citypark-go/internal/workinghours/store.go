// Package workinghours stores the hours when orders are accepted as one row.
package workinghours

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/salewindow"
	"github.com/drewp1ter/citypark-go/internal/store"
)

// Hours are "HH:MM" with a leading zero, start before end, the same day.
type Hours struct {
	Start string `json:"start" validate:"required,hhmm"`
	End   string `json:"end" validate:"required,hhmm"`
}

// Contains reports whether hhmm ("18:30") is within [Start, End], both ends included (used for deliveryTime).
// Both sides are zero-padded, so comparing the strings compares the times.
func (h Hours) Contains(hhmm string) bool {
	return hhmm >= h.Start && hhmm <= h.End
}

// OpenAt reports whether t (in its own location) is within [Start, End): the order window, end excluded.
func (h Hours) OpenAt(t time.Time) bool {
	s, okStart := salewindow.ParseClock(h.Start)
	e, okEnd := salewindow.ParseClock(h.End)
	if !okStart || !okEnd {
		return false
	}
	return salewindow.Window{Start: &s, End: &e}.OpenAtTime(t)
}

// Display drops a leading zero: "09:00" -> "9:00", "22:00" -> "22:00".
func Display(hhmm string) string {
	if strings.HasPrefix(hhmm, "0") && len(hhmm) > 1 {
		return hhmm[1:]
	}
	return hhmm
}

type Store struct{ db *sqlx.DB }

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// Get returns the hours and the time of the last change. It returns store.ErrNotFound if the row is missing.
func (s *Store) Get(ctx context.Context) (Hours, time.Time, error) {
	var row struct {
		Start     string    `db:"start_time"`
		End       string    `db:"end_time"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := s.db.GetContext(ctx, &row, `SELECT start_time, end_time, updated_at FROM working_hours WHERE id = 1`)
	if err != nil {
		return Hours{}, time.Time{}, store.NotFound(err)
	}
	return Hours{Start: row.Start, End: row.End}, row.UpdatedAt, nil
}

// Put replaces both times and returns the new update time.
func (s *Store) Put(ctx context.Context, h Hours) (time.Time, error) {
	var at time.Time
	err := s.db.GetContext(ctx, &at,
		`UPDATE working_hours SET start_time = $1, end_time = $2, updated_at = now() WHERE id = 1 RETURNING updated_at`,
		h.Start, h.End)
	if err != nil {
		return time.Time{}, store.NotFound(err)
	}
	return at, nil
}
