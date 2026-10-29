package customer

import (
	"context"

	"github.com/drewp1ter/citypark-go/internal/store"
)

type Identity struct {
	UserID   int64  `db:"user_id"`
	Provider string `db:"provider"`
	Subject  string `db:"subject"`
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store { return &Store{db: db} }

func (s *Store) IdentityOwner(ctx context.Context, provider, subject string) (int64, error) {
	var id int64
	err := s.db.GetContext(ctx, &id, `SELECT user_id FROM customer_identities WHERE provider = $1 AND subject = $2`, provider, subject)
	if err != nil {
		return 0, store.NotFound(err)
	}
	return id, nil
}

func (s *Store) AddIdentity(ctx context.Context, identity Identity) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO customer_identities (user_id, provider, subject) VALUES ($1, $2, $3)`, identity.UserID, identity.Provider, identity.Subject)
	return err
}
