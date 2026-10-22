// Package settings stores the site's contact details as one JSON object.
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
)

type Phone struct {
	Label      string `json:"label" validate:"required,max=60"`
	ShortLabel string `json:"shortLabel" validate:"required,max=30"`
	Display    string `json:"display" validate:"required,max=40"`
	Number     string `json:"number" validate:"required,max=20"` // "+79183119791" or "88001002441"
	AltDisplay string `json:"altDisplay" validate:"max=40"`      // second (landline) number, optional
	AltNumber  string `json:"altNumber" validate:"max=20"`
}

type Social struct {
	Name    string `json:"name" validate:"required,max=40"`    // "ВКонтакте"
	Label   string `json:"label" validate:"required,max=10"`   // "VK"
	URL     string `json:"url" validate:"required,max=200"`    // https only
	Display string `json:"display" validate:"required,max=60"` // "vk.com/cityparkvip"
}

type Settings struct {
	Phones struct {
		Restaurant Phone `json:"restaurant"`
		Hotel      Phone `json:"hotel"`
		Free       Phone `json:"free"`
		Landline   Phone `json:"landline"`
	} `json:"phones"`
	Address struct {
		Postcode string `json:"postcode" validate:"required,len=6,numeric"`
		City     string `json:"city" validate:"required,max=60"`
		Street   string `json:"street" validate:"required,max=100"`
	} `json:"address"`
	Email   string `json:"email" validate:"required,email,max=100"`
	Company struct {
		Name string `json:"name" validate:"required,max=100"`
		INN  string `json:"inn" validate:"required,numeric,min=10,max=12"`
		OGRN string `json:"ogrn" validate:"required,numeric,min=13,max=15"`
	} `json:"company"`
	Social []Social `json:"social" validate:"max=10,dive"`
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// Get returns the settings and the time of the last change. It returns store.ErrNotFound if the row is missing.
func (s *Store) Get(ctx context.Context) (Settings, time.Time, error) {
	var row struct {
		Data      []byte    `db:"data"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	if err := s.db.GetContext(ctx, &row, `SELECT data, updated_at FROM site_settings WHERE id = 1`); err != nil {
		return Settings{}, time.Time{}, store.NotFound(err)
	}
	var st Settings
	if err := json.Unmarshal(row.Data, &st); err != nil {
		return Settings{}, time.Time{}, fmt.Errorf("decode site settings: %w", err)
	}
	if st.Social == nil {
		st.Social = []Social{}
	}
	return st, row.UpdatedAt, nil
}

// Put replaces the settings and returns the new update time.
func (s *Store) Put(ctx context.Context, st Settings) (time.Time, error) {
	if st.Social == nil {
		st.Social = []Social{}
	}
	data, err := json.Marshal(st)
	if err != nil {
		return time.Time{}, fmt.Errorf("encode site settings: %w", err)
	}
	var at time.Time
	err = s.db.GetContext(ctx, &at,
		`UPDATE site_settings SET data = $1, updated_at = now() WHERE id = 1 RETURNING updated_at`, data)
	if err != nil {
		return time.Time{}, store.NotFound(err)
	}
	return at, nil
}
