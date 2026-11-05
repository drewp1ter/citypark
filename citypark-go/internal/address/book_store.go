package address

import (
	"context"
	"github.com/drewp1ter/citypark-go/internal/store"
)

const entryProjection = `SELECT ua.id, ua.address_id, ua.name, a.city, a.address,
 COALESCE(ua.street, '') AS street, ua.apartment, ua.entrance,
 ua.is_default, ua.street IS NULL AS needs_review, ua.created_at, ua.updated_at
 FROM user_addresses ua JOIN addresses a ON a.id=ua.address_id`

func (s *Store) ListEntries(ctx context.Context, userID int64) ([]Entry, error) {
	entries := []Entry{}
	if err := s.db.SelectContext(ctx, &entries, entryProjection+` WHERE ua.user_id=$1 ORDER BY ua.id`, userID); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Store) EntryByID(ctx context.Context, userID, entryID int64) (Entry, error) {
	var entry Entry
	if err := s.db.GetContext(ctx, &entry, entryProjection+` WHERE ua.user_id=$1 AND ua.id=$2`, userID, entryID); err != nil {
		return Entry{}, store.NotFound(err)
	}
	return entry, nil
}
