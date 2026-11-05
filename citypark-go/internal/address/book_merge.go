package address

import (
	"context"
	"errors"
)

// MergeEntries transfers associations inside the caller's transaction.
// The caller must hold both owner rows locked in ascending ID order.
func (s *Store) MergeEntries(ctx context.Context, sourceID, destinationID int64) error {
	if sourceID <= 0 || destinationID <= 0 || sourceID == destinationID {
		return errors.New("invalid address merge owners")
	}
	owners := []int64{}
	if err := s.db.SelectContext(ctx, &owners, `SELECT id FROM users WHERE id=ANY($1) ORDER BY id`, []int64{sourceID, destinationID}); err != nil {
		return err
	}
	if len(owners) != 2 {
		return errors.New("address merge owner missing")
	}
	source, err := s.ListEntries(ctx, sourceID)
	if err != nil {
		return err
	}
	destination, err := s.ListEntries(ctx, destinationID)
	if err != nil {
		return err
	}
	survivors := make(map[string]int64)
	destinationDefault := false
	for _, entry := range destination {
		key := DuplicateKey(entry.City, entry.Address)
		if _, exists := survivors[key]; !exists {
			survivors[key] = entry.ID
		}
		destinationDefault = destinationDefault || entry.IsDefault
	}
	var sourceDefault int64
	var defaultCount int64
	for _, entry := range source {
		if entry.IsDefault {
			defaultCount++
		}
	}
	if err := s.mergeMutation(ctx, defaultCount, `UPDATE user_addresses SET is_default=false,updated_at=now() WHERE user_id=$1 AND is_default`, sourceID); err != nil {
		return err
	}
	for _, entry := range source {
		key := DuplicateKey(entry.City, entry.Address)
		survivor, duplicate := survivors[key]
		if !duplicate {
			survivor = entry.ID
			survivors[key] = survivor
			err = s.mergeMutation(ctx, 1, `UPDATE user_addresses SET user_id=$2,updated_at=now() WHERE user_id=$1 AND id=$3`, sourceID, destinationID, entry.ID)
		} else {
			err = s.mergeMutation(ctx, 1, `DELETE FROM user_addresses WHERE user_id=$1 AND id=$2`, sourceID, entry.ID)
		}
		if err != nil {
			return err
		}
		if entry.IsDefault {
			sourceDefault = survivor
		}
	}
	if !destinationDefault && sourceDefault != 0 {
		return s.mergeMutation(ctx, 1, `UPDATE user_addresses SET is_default=true,updated_at=now() WHERE user_id=$1 AND id=$2`, destinationID, sourceDefault)
	}
	return nil
}

func (s *Store) mergeMutation(ctx context.Context, expected int64, query string, args ...any) error {
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != expected {
		return errors.New("address merge affected unexpected rows")
	}
	return nil
}
