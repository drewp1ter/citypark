// Package hotel stores how many hotel rooms of each type are free.
package hotel

import (
	"context"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
)

const columns = `vip_free, single_balcony_free, single_free, double_twin_free, double_euro_free, updated_at`

// Rooms is the number of free rooms of each type.
type Rooms struct {
	VIP           int `db:"vip_free" json:"vip"`
	SingleBalcony int `db:"single_balcony_free" json:"singleBalcony"`
	Single        int `db:"single_free" json:"single"`
	DoubleTwin    int `db:"double_twin_free" json:"doubleTwin"`
	DoubleEuro    int `db:"double_euro_free" json:"doubleEuro"`
}

// Free is the total number of free rooms.
func (r Rooms) Free() int {
	return r.VIP + r.SingleBalcony + r.Single + r.DoubleTwin + r.DoubleEuro
}

// State is the stored row.
type State struct {
	Rooms
	UpdatedAt time.Time `db:"updated_at"`
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// Get returns store.ErrNotFound while the table is empty.
func (s *Store) Get(ctx context.Context) (State, error) {
	var st State
	if err := s.db.GetContext(ctx, &st, `SELECT `+columns+` FROM hotel_rooms WHERE id = 1`); err != nil {
		return State{}, store.NotFound(err)
	}
	return st, nil
}

// Set replaces all five counts and returns the stored row.
func (s *Store) Set(ctx context.Context, rooms Rooms) (State, error) {
	var st State
	err := s.db.GetContext(ctx, &st,
		`INSERT INTO hotel_rooms (id, vip_free, single_balcony_free, single_free, double_twin_free, double_euro_free)
		 VALUES (1, $1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE SET vip_free = EXCLUDED.vip_free, single_balcony_free = EXCLUDED.single_balcony_free,
		     single_free = EXCLUDED.single_free, double_twin_free = EXCLUDED.double_twin_free,
		     double_euro_free = EXCLUDED.double_euro_free, updated_at = now()
		 RETURNING `+columns,
		rooms.VIP, rooms.SingleBalcony, rooms.Single, rooms.DoubleTwin, rooms.DoubleEuro)
	if err != nil {
		return State{}, err
	}
	return st, nil
}
