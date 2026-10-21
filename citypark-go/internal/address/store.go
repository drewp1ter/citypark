package address

import (
	"context"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
)

const columns = `id, city, address, created_at, updated_at`

type Address struct {
	ID        int64     `db:"id" json:"id"`
	City      string    `db:"city" json:"city"`
	Address   string    `db:"address" json:"address"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// FindOrCreate returns the address with this city and address text, creating it when missing.
func (s *Store) FindOrCreate(ctx context.Context, city, address string) (Address, error) {
	var a Address
	err := s.db.GetContext(ctx, &a,
		`INSERT INTO addresses (city, address) VALUES ($1, $2)
		 ON CONFLICT (city, address) DO UPDATE SET updated_at = addresses.updated_at
		 RETURNING `+columns,
		city, address)
	if err != nil {
		return Address{}, err
	}
	return a, nil
}

// LinkUser attaches the address to the user. Linking twice is not an error.
func (s *Store) LinkUser(ctx context.Context, userID, addressID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_addresses (user_id, address_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, addressID)
	return err
}

// UnlinkUser removes the link. It is not an error when the link does not exist.
func (s *Store) UnlinkUser(ctx context.Context, userID, addressID int64) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM user_addresses WHERE user_id = $1 AND address_id = $2`,
		userID, addressID)
	return err
}

// ListByUser returns the user's addresses ordered by address id. Never returns nil.
func (s *Store) ListByUser(ctx context.Context, userID int64) ([]Address, error) {
	list := []Address{}
	err := s.db.SelectContext(ctx, &list,
		`SELECT a.id, a.city, a.address, a.created_at, a.updated_at
		 FROM addresses a
		 JOIN user_addresses ua ON ua.address_id = a.id
		 WHERE ua.user_id = $1
		 ORDER BY a.id`,
		userID)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ByID returns store.ErrNotFound when missing.
func (s *Store) ByID(ctx context.Context, id int64) (Address, error) {
	var a Address
	if err := s.db.GetContext(ctx, &a, `SELECT `+columns+` FROM addresses WHERE id = $1`, id); err != nil {
		return Address{}, store.NotFound(err)
	}
	return a, nil
}
