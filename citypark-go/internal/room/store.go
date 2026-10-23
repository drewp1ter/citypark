// Package room stores the five hotel room types: texts, price per night and photo.
package room

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
)

const columns = `id, sort, kind, balcony, name, hotel_name, numbers, area, area_detailed, capacity, summary, description,
	price_kopecks, photo_url, photo_alt, photo_position, gallery, hotel_note, updated_at`

// GalleryPhoto is one extra photo of a room.
type GalleryPhoto struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

// Gallery is the ordered list of extra photos, stored as jsonb.
type Gallery []GalleryPhoto

// Scan implements sql.Scanner.
func (g *Gallery) Scan(src any) error {
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	case nil:
		*g = Gallery{}
		return nil
	default:
		return fmt.Errorf("scan gallery: unsupported type %T", src)
	}
	var out Gallery
	if err := json.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("scan gallery: %w", err)
	}
	if out == nil {
		out = Gallery{}
	}
	*g = out
	return nil
}

// Value implements driver.Valuer. A nil gallery is written as `[]`, never `null`.
func (g Gallery) Value() (driver.Value, error) {
	if g == nil {
		g = Gallery{}
	}
	data, err := json.Marshal(g)
	if err != nil {
		return nil, fmt.Errorf("encode gallery: %w", err)
	}
	return string(data), nil
}

// Replaced is what Update overwrote: the old photo path and the old gallery.
type Replaced struct {
	PhotoURL string
	Gallery  Gallery
}

type Room struct {
	ID            string        `db:"id"`
	Sort          int           `db:"sort"`
	Kind          string        `db:"kind"`
	Balcony       bool          `db:"balcony"`
	Name          string        `db:"name"`
	HotelName     string        `db:"hotel_name"`
	Numbers       string        `db:"numbers"`
	Area          string        `db:"area"`
	AreaDetailed  string        `db:"area_detailed"`
	Capacity      string        `db:"capacity"`
	Summary       string        `db:"summary"`
	Description   string        `db:"description"`
	Price         money.Kopecks `db:"price_kopecks"`
	PhotoURL      string        `db:"photo_url"`
	PhotoAlt      string        `db:"photo_alt"`
	PhotoPosition string        `db:"photo_position"`
	Gallery       Gallery       `db:"gallery"`
	HotelNote     string        `db:"hotel_note"`
	UpdatedAt     time.Time     `db:"updated_at"`
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// List returns all rooms ordered by sort. Never returns nil.
func (s *Store) List(ctx context.Context) ([]Room, error) {
	list := []Room{}
	if err := s.db.SelectContext(ctx, &list, `SELECT `+columns+` FROM rooms ORDER BY sort`); err != nil {
		return nil, err
	}
	return list, nil
}

// Get returns store.ErrNotFound for an unknown id.
func (s *Store) Get(ctx context.Context, id string) (Room, error) {
	var r Room
	if err := s.db.GetContext(ctx, &r, `SELECT `+columns+` FROM rooms WHERE id = $1`, id); err != nil {
		return Room{}, store.NotFound(err)
	}
	return r, nil
}

// Update writes the editable fields of r (everything except ID, Sort, Kind, Balcony, UpdatedAt) and returns the
// stored row and what it replaced (the old photo path and gallery). store.ErrNotFound for an unknown id.
//
// The read of the old photo and gallery and the write run in one transaction. A Store built on a *sqlx.DB opens
// that transaction itself; a Store built on a *sqlx.Tx joins the caller's transaction.
func (s *Store) Update(ctx context.Context, r Room) (Room, Replaced, error) {
	db, ok := s.db.(*sqlx.DB)
	if !ok {
		return s.update(ctx, r)
	}
	var (
		saved Room
		old   Replaced
	)
	err := store.InTx(ctx, db, func(tx *sqlx.Tx) error {
		var err error
		saved, old, err = NewStore(tx).update(ctx, r)
		return err
	})
	if err != nil {
		return Room{}, Replaced{}, err
	}
	return saved, old, nil
}

func (s *Store) update(ctx context.Context, r Room) (Room, Replaced, error) {
	var prev struct {
		PhotoURL string  `db:"photo_url"`
		Gallery  Gallery `db:"gallery"`
	}
	if err := s.db.GetContext(ctx, &prev, `SELECT photo_url, gallery FROM rooms WHERE id = $1 FOR UPDATE`, r.ID); err != nil {
		return Room{}, Replaced{}, store.NotFound(err)
	}
	old := Replaced{PhotoURL: prev.PhotoURL, Gallery: prev.Gallery}
	var saved Room
	err := s.db.GetContext(ctx, &saved,
		`UPDATE rooms SET name = $2, hotel_name = $3, numbers = $4, area = $5, area_detailed = $6, capacity = $7,
		     description = $8, price_kopecks = $9, photo_url = $10, photo_alt = $11, photo_position = $12,
		     hotel_note = $13, summary = $14, gallery = $15, updated_at = now()
		 WHERE id = $1
		 RETURNING `+columns,
		r.ID, r.Name, r.HotelName, r.Numbers, r.Area, r.AreaDetailed, r.Capacity, r.Description,
		int64(r.Price), r.PhotoURL, r.PhotoAlt, r.PhotoPosition, r.HotelNote, r.Summary, r.Gallery)
	if err != nil {
		return Room{}, Replaced{}, store.NotFound(err)
	}
	return saved, old, nil
}
