package address

import (
	"context"
	"errors"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
	"github.com/jmoiron/sqlx"
)

func (s *Service) SetDefault(ctx context.Context, userID, entryID int64) (Entry, error) {
	var entry Entry
	err := store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		book := NewStore(tx)
		if err := book.lockOwner(ctx, userID); err != nil {
			return err
		}
		if _, err := book.EntryByID(ctx, userID, entryID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_addresses SET is_default=false,updated_at=now() WHERE user_id=$1 AND is_default AND id<>$2`, userID, entryID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE user_addresses SET is_default=true,updated_at=now() WHERE user_id=$1 AND id=$2 AND NOT is_default`, userID, entryID); err != nil {
			return err
		}
		var err error
		entry, err = book.EntryByID(ctx, userID, entryID)
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	return entry, nil
}
func (s *Service) DeleteEntry(ctx context.Context, userID, entryID int64) error {
	return store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		book := NewStore(tx)
		if err := book.lockOwner(ctx, userID); err != nil {
			return err
		}
		if _, err := book.EntryByID(ctx, userID, entryID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM user_addresses WHERE user_id=$1 AND id=$2`, userID, entryID)
		return err
	})
}
func lockLegacyOwner(ctx context.Context, tx *sqlx.Tx, userID int64) error {
	var id int64
	err := tx.GetContext(ctx, &id, `SELECT id FROM users WHERE id=$1 AND merged_into_user_id IS NULL FOR UPDATE`, userID)
	if errors.Is(store.NotFound(err), store.ErrNotFound) {
		return ErrOwnerUnavailable
	}
	return err
}
func (s *Service) CreateLegacy(ctx context.Context, userID int64, city, address string) (Address, error) {
	in := Input{City: strings.TrimSpace(city), Address: strings.TrimSpace(address)}
	if err := validate.Struct(in); err != nil {
		return Address{}, err
	}
	var result Address
	err := store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		if err := lockLegacyOwner(ctx, tx, userID); err != nil {
			return err
		}
		book := NewStore(tx)
		entries, err := book.ListEntries(ctx, userID)
		if err != nil {
			return err
		}
		result, err = book.FindOrCreate(ctx, in.City, in.Address)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO user_addresses(user_id,address_id,is_default) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, userID, result.ID, len(entries) == 0)
		return err
	})
	if err != nil {
		return Address{}, err
	}
	return result, nil
}
func (s *Service) DeleteLegacy(ctx context.Context, userID, addressID int64) error {
	return store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		if err := lockLegacyOwner(ctx, tx, userID); err != nil {
			return err
		}
		return NewStore(tx).UnlinkUser(ctx, userID, addressID)
	})
}
