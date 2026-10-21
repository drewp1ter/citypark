// Package deliveryprice stores the delivery price of every town.
package deliveryprice

import (
	"context"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
)

const columns = `id, name, value, price_kopecks, created_at, updated_at`

type DeliveryPrice struct {
	ID        int64         `db:"id" json:"id"`
	Name      string        `db:"name" json:"name"`
	Value     string        `db:"value" json:"value"`
	Price     money.Kopecks `db:"price_kopecks" json:"price"`
	CreatedAt time.Time     `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time     `db:"updated_at" json:"updatedAt"`
}

type Input struct {
	Name  string        `json:"name" validate:"required,max=255"`
	Value string        `json:"value" validate:"max=255"`
	Price money.Kopecks `json:"price" validate:"gt=0"`
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// Create inserts a delivery price. A taken name is a unique violation.
func (s *Store) Create(ctx context.Context, in Input) (DeliveryPrice, error) {
	var d DeliveryPrice
	err := s.db.GetContext(ctx, &d,
		`INSERT INTO delivery_prices (name, value, price_kopecks) VALUES ($1, $2, $3)
		 RETURNING `+columns,
		in.Name, in.Value, int64(in.Price))
	if err != nil {
		return DeliveryPrice{}, err
	}
	return d, nil
}

// List returns all delivery prices ordered by id. Never returns nil.
func (s *Store) List(ctx context.Context) ([]DeliveryPrice, error) {
	list := []DeliveryPrice{}
	err := s.db.SelectContext(ctx, &list, `SELECT `+columns+` FROM delivery_prices ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ByName matches the name exactly. It returns store.ErrNotFound when missing.
func (s *Store) ByName(ctx context.Context, name string) (DeliveryPrice, error) {
	var d DeliveryPrice
	if err := s.db.GetContext(ctx, &d, `SELECT `+columns+` FROM delivery_prices WHERE name = $1`, name); err != nil {
		return DeliveryPrice{}, store.NotFound(err)
	}
	return d, nil
}

// Update replaces name, value and price. It returns store.ErrNotFound when the row is missing.
func (s *Store) Update(ctx context.Context, id int64, in Input) (DeliveryPrice, error) {
	var d DeliveryPrice
	err := s.db.GetContext(ctx, &d,
		`UPDATE delivery_prices SET name = $2, value = $3, price_kopecks = $4, updated_at = now()
		 WHERE id = $1
		 RETURNING `+columns,
		id, in.Name, in.Value, int64(in.Price))
	if err != nil {
		return DeliveryPrice{}, store.NotFound(err)
	}
	return d, nil
}

// Delete removes the row. It is not an error when the row does not exist.
func (s *Store) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM delivery_prices WHERE id = $1`, id)
	return err
}
