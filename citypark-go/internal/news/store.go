// Package news stores the news items of the site: text, one optional photo, and a slug for the page address.
package news

import (
	"context"
	"fmt"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
)

const (
	columns = `id, slug, date, title, short_title, excerpt, body, photo_url, photo_alt, photo_position, updated_at`
	// listColumns leaves the body out: it can be long and the list does not show it.
	listColumns = `id, slug, date, title, short_title, excerpt, '' AS body, photo_url, photo_alt, photo_position,
	updated_at`
)

type Item struct {
	ID            int64     `db:"id"`
	Slug          string    `db:"slug"`
	Date          time.Time `db:"date"`
	Title         string    `db:"title"`
	ShortTitle    string    `db:"short_title"`
	Excerpt       string    `db:"excerpt"`
	Body          string    `db:"body"`
	PhotoURL      string    `db:"photo_url"`
	PhotoAlt      string    `db:"photo_alt"`
	PhotoPosition string    `db:"photo_position"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// Input is what an admin can write. An empty Slug means "make it from the title".
type Input struct {
	Slug          string
	Date          time.Time
	Title         string
	ShortTitle    string
	Excerpt       string
	Body          string
	PhotoURL      string
	PhotoAlt      string
	PhotoPosition string
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// List returns one page of items, newest first, without Body, and the total number of items. Never returns nil.
func (s *Store) List(ctx context.Context, limit, offset int) ([]Item, int, error) {
	items := []Item{}
	err := s.db.SelectContext(ctx, &items,
		`SELECT `+listColumns+` FROM news ORDER BY date DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT count(*) FROM news`); err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

// GetBySlug returns store.ErrNotFound for an unknown slug.
func (s *Store) GetBySlug(ctx context.Context, slug string) (Item, error) {
	var it Item
	if err := s.db.GetContext(ctx, &it, `SELECT `+columns+` FROM news WHERE slug = $1`, slug); err != nil {
		return Item{}, store.NotFound(err)
	}
	return it, nil
}

// Create inserts an item. When in.Slug is empty the slug is made from the title and gets "-2", "-3", ... until it
// is free. An explicit slug that exists returns an error for which store.IsUniqueViolation is true.
// Call it inside a transaction so the free-slug check and the insert see the same data.
func (s *Store) Create(ctx context.Context, in Input) (Item, error) {
	if in.Slug == "" {
		slug, err := s.freeSlug(ctx, Slugify(in.Title), 0)
		if err != nil {
			return Item{}, err
		}
		in.Slug = slug
	}
	var it Item
	err := s.db.GetContext(ctx, &it,
		`INSERT INTO news (slug, date, title, short_title, excerpt, body, photo_url, photo_alt, photo_position)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING `+columns,
		in.Slug, in.Date, in.Title, in.ShortTitle, in.Excerpt, in.Body, in.PhotoURL, in.PhotoAlt, in.PhotoPosition)
	if err != nil {
		return Item{}, err
	}
	return it, nil
}

// Update replaces all fields of item id and returns the stored row and the photo path it replaced.
// store.ErrNotFound for an unknown id. The slug rules are those of Create; the item's own slug is not a conflict.
func (s *Store) Update(ctx context.Context, id int64, in Input) (Item, string, error) {
	var old string
	if err := s.db.GetContext(ctx, &old, `SELECT photo_url FROM news WHERE id = $1 FOR UPDATE`, id); err != nil {
		return Item{}, "", store.NotFound(err)
	}
	if in.Slug == "" {
		slug, err := s.freeSlug(ctx, Slugify(in.Title), id)
		if err != nil {
			return Item{}, "", err
		}
		in.Slug = slug
	}
	var it Item
	err := s.db.GetContext(ctx, &it,
		`UPDATE news SET slug = $2, date = $3, title = $4, short_title = $5, excerpt = $6, body = $7,
		     photo_url = $8, photo_alt = $9, photo_position = $10, updated_at = now()
		 WHERE id = $1
		 RETURNING `+columns,
		id, in.Slug, in.Date, in.Title, in.ShortTitle, in.Excerpt, in.Body, in.PhotoURL, in.PhotoAlt, in.PhotoPosition)
	if err != nil {
		return Item{}, "", err
	}
	return it, old, nil
}

// Delete removes item id and returns its photo path. store.ErrNotFound for an unknown id.
func (s *Store) Delete(ctx context.Context, id int64) (string, error) {
	var photo string
	if err := s.db.GetContext(ctx, &photo, `DELETE FROM news WHERE id = $1 RETURNING photo_url`, id); err != nil {
		return "", store.NotFound(err)
	}
	return photo, nil
}

// freeSlug returns base, or base-2, base-3, ... the first one no other item (id != ignoreID) uses.
func (s *Store) freeSlug(ctx context.Context, base string, ignoreID int64) (string, error) {
	slug := base
	for n := 2; ; n++ {
		var taken bool
		err := s.db.GetContext(ctx, &taken,
			`SELECT EXISTS (SELECT 1 FROM news WHERE slug = $1 AND id <> $2)`, slug, ignoreID)
		if err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, n)
	}
}
