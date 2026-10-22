// Package content stores the editable copy of the site pages as content blocks addressed by a block key.
package content

import (
	"context"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
)

const columns = `key, page, section, sort, label, type, max_length, text_value, image_url, image_alt, image_position,
	updated_at`

// Block types.
const (
	TypeText     = "text"
	TypeLines    = "lines"
	TypeRichtext = "richtext"
	TypeImage    = "image"
)

type Block struct {
	Key           string    `db:"key"`
	Page          string    `db:"page"`
	Section       string    `db:"section"`
	Sort          int       `db:"sort"`
	Label         string    `db:"label"`
	Type          string    `db:"type"`
	MaxLength     int       `db:"max_length"`
	TextValue     string    `db:"text_value"`
	ImageURL      string    `db:"image_url"`
	ImageAlt      string    `db:"image_alt"`
	ImagePosition string    `db:"image_position"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// List returns the blocks ordered by page, sort and key. A non-empty page keeps only that page. Never returns nil.
func (s *Store) List(ctx context.Context, page string) ([]Block, error) {
	blocks := []Block{}
	query := `SELECT ` + columns + ` FROM content_blocks`
	args := []any{}
	if page != "" {
		query += ` WHERE page = $1`
		args = append(args, page)
	}
	query += ` ORDER BY page, sort, key`
	if err := s.db.SelectContext(ctx, &blocks, query, args...); err != nil {
		return nil, err
	}
	return blocks, nil
}

// Get returns one block or store.ErrNotFound.
func (s *Store) Get(ctx context.Context, key string) (Block, error) {
	var b Block
	err := s.db.GetContext(ctx, &b, `SELECT `+columns+` FROM content_blocks WHERE key = $1`, key)
	return b, store.NotFound(err)
}

// SetText stores text_value of a text, lines or richtext block and returns the block, or store.ErrNotFound.
func (s *Store) SetText(ctx context.Context, key, value string) (Block, error) {
	var b Block
	err := s.db.GetContext(ctx, &b, `UPDATE content_blocks SET text_value = $2, updated_at = now()
		WHERE key = $1 RETURNING `+columns, key, value)
	return b, store.NotFound(err)
}

// SetImage stores the photo of an image block and returns the block and the photo URL it had before,
// or store.ErrNotFound. Run it in a transaction: the row is locked while the old URL is read.
func (s *Store) SetImage(ctx context.Context, key, url, alt, position string) (Block, string, error) {
	var old string
	err := s.db.GetContext(ctx, &old, `SELECT image_url FROM content_blocks WHERE key = $1 FOR UPDATE`, key)
	if err != nil {
		return Block{}, "", store.NotFound(err)
	}
	var b Block
	err = s.db.GetContext(ctx, &b, `UPDATE content_blocks SET image_url = $2, image_alt = $3, image_position = $4,
		updated_at = now() WHERE key = $1 RETURNING `+columns, key, url, alt, position)
	return b, old, store.NotFound(err)
}
