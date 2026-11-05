package address

import (
	"context"
	"errors"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

type Service struct{ db *sqlx.DB }

func NewService(db *sqlx.DB) *Service { return &Service{db: db} }

type InvalidFieldsError struct{ Fields map[string]string }

func (e *InvalidFieldsError) Error() string { return "invalid address fields" }

var ErrDuplicate = errors.New("saved address already exists")
var ErrOwnerUnavailable = errors.New("address owner unavailable")

func (s *Service) CreateEntry(ctx context.Context, userID int64, in Fields) (Entry, bool, error) {
	in = CleanFields(in)
	if fields := FieldErrors(in); len(fields) > 0 {
		return Entry{}, false, &InvalidFieldsError{Fields: fields}
	}
	var entry Entry
	created := false
	err := store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		book := NewStore(tx)
		if err := book.lockOwner(ctx, userID); err != nil {
			return err
		}
		entries, err := book.ListEntries(ctx, userID)
		if err != nil {
			return err
		}
		key := DuplicateKey(in.City, FormatAddress(in))
		for _, existing := range entries {
			if DuplicateKey(existing.City, existing.Address) == key {
				entry = existing
				return nil
			}
		}
		a, err := book.FindOrCreate(ctx, in.City, FormatAddress(in))
		if err != nil {
			return err
		}
		id, err := book.insertEntry(ctx, userID, a.ID, in, len(entries) == 0)
		if err != nil {
			return err
		}
		entry, err = book.EntryByID(ctx, userID, id)
		created = true
		return err
	})
	if err != nil {
		return Entry{}, false, err
	}
	return entry, created, nil
}

func (s *Service) UpdateEntry(ctx context.Context, userID, entryID int64, in Fields) (Entry, error) {
	in = CleanFields(in)
	if fields := FieldErrors(in); len(fields) > 0 {
		return Entry{}, &InvalidFieldsError{Fields: fields}
	}
	var entry Entry
	err := store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		book := NewStore(tx)
		if err := book.lockOwner(ctx, userID); err != nil {
			return err
		}
		if _, err := book.EntryByID(ctx, userID, entryID); err != nil {
			return err
		}
		entries, err := book.ListEntries(ctx, userID)
		if err != nil {
			return err
		}
		key := DuplicateKey(in.City, FormatAddress(in))
		for _, existing := range entries {
			if existing.ID != entryID && DuplicateKey(existing.City, existing.Address) == key {
				return ErrDuplicate
			}
		}
		a, err := book.FindOrCreate(ctx, in.City, FormatAddress(in))
		if err != nil {
			return err
		}
		if err := book.updateEntry(ctx, userID, entryID, a.ID, in); err != nil {
			return err
		}
		entry, err = book.EntryByID(ctx, userID, entryID)
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	return entry, nil
}
