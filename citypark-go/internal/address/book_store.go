package address

import (
	"context"
	"errors"
	"github.com/drewp1ter/citypark-go/internal/store"
)

const entryProjection = `SELECT ua.id, ua.address_id, ua.name, ua.recipient_name, ua.recipient_phone, a.city, a.address,
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

func (s *Store) lockOwner(ctx context.Context, userID int64) error {
	var id int64
	err := s.db.GetContext(ctx, &id, `SELECT id FROM users WHERE id=$1 AND role='user' AND merged_into_user_id IS NULL FOR UPDATE`, userID)
	if errors.Is(store.NotFound(err), store.ErrNotFound) {
		return ErrOwnerUnavailable
	}
	return err
}

func (s *Store) insertEntry(ctx context.Context, userID, addressID int64, in Fields, isDefault bool) (int64, error) {
	var id int64
	err := s.db.GetContext(ctx, &id, `INSERT INTO user_addresses(user_id,address_id,name,street,apartment,entrance,is_default,recipient_name,recipient_phone)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, userID, addressID, in.Name, in.Street, in.Apartment, in.Entrance, isDefault, in.RecipientName, in.RecipientPhone)
	return id, err
}

func (s *Store) updateEntry(ctx context.Context, userID, entryID, addressID int64, in Fields) error {
	_, err := s.db.ExecContext(ctx, `UPDATE user_addresses SET address_id=$3,name=$4,street=$5,apartment=$6,entrance=$7,recipient_name=$8,recipient_phone=$9,updated_at=now() WHERE user_id=$1 AND id=$2`, userID, entryID, addressID, in.Name, in.Street, in.Apartment, in.Entrance, in.RecipientName, in.RecipientPhone)
	return err
}
