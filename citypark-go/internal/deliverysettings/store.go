// Package deliverysettings stores informational delivery captions independently of town tariffs.
package deliverysettings

import (
	"context"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
)

type Settings struct {
	Cooking        string        `json:"cooking" db:"cooking"`
	OutskirtsLabel string        `json:"outskirtsLabel" db:"outskirts_label"`
	OutskirtsMin   money.Kopecks `json:"outskirtsMin" db:"outskirts_min_kopecks"`
	OutskirtsMax   money.Kopecks `json:"outskirtsMax" db:"outskirts_max_kopecks"`
}
type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store { return &Store{db: db} }
func (s *Store) Get(ctx context.Context) (Settings, time.Time, error) {
	var row struct {
		Settings
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := s.db.GetContext(ctx, &row, `SELECT cooking, outskirts_label, outskirts_min_kopecks, outskirts_max_kopecks, updated_at FROM delivery_settings WHERE id = 1`)
	return row.Settings, row.UpdatedAt, store.NotFound(err)
}
func (s *Store) Put(ctx context.Context, settings Settings) (time.Time, error) {
	var at time.Time
	err := s.db.GetContext(ctx, &at, `UPDATE delivery_settings SET cooking=$1, outskirts_label=$2, outskirts_min_kopecks=$3, outskirts_max_kopecks=$4, updated_at=now() WHERE id=1 RETURNING updated_at`, settings.Cooking, settings.OutskirtsLabel, settings.OutskirtsMin, settings.OutskirtsMax)
	return at, store.NotFound(err)
}
