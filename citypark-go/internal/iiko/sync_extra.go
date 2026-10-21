package iiko

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// imageSet maps image urls to the id of their owner. A url that appears twice
// keeps the last owner, because the url column is unique.
type imageSet struct {
	owners map[string]int64
	urls   []string // in first-seen order, so the SQL runs in a stable order
}

func newImageSet() *imageSet {
	return &imageSet{owners: make(map[string]int64), urls: make([]string, 0)}
}

func (s *imageSet) add(url string, owner int64) {
	if _, seen := s.owners[url]; !seen {
		s.urls = append(s.urls, url)
	}
	s.owners[url] = owner
}

// syncImages saves the button image of every saved product and deletes the
// other iiko images. menu/v3 carries no category images, so those are all
// deleted. Images of another type are never touched.
func syncImages(ctx context.Context, tx *sqlx.Tx, menu *Menu, productIDs map[string]int64) error {
	if err := replaceImages(ctx, tx, "category_images", "category_id", newImageSet()); err != nil {
		return err
	}

	products := newImageSet()
	for _, g := range menu.ItemsGroups {
		for _, item := range g.Items {
			id, ok := productIDs[item.ProductID]
			if !ok {
				continue // not for sale, so it was not saved
			}
			for _, sizePrice := range item.SizePrices {
				if sizePrice.ButtonImageURL != "" {
					products.add(sizePrice.ButtonImageURL, id)
					break
				}
			}
		}
	}
	return replaceImages(ctx, tx, "product_images", "product_id", products)
}

// replaceImages upserts every url of set and deletes the other iiko images.
// table and ownerColumn are literals from this package, never user input.
func replaceImages(ctx context.Context, tx *sqlx.Tx, table, ownerColumn string, set *imageSet) error {
	upsert := `INSERT INTO ` + table + ` (` + ownerColumn + `, url, type) VALUES ($1, $2, 'from_iiko')
ON CONFLICT (url) DO UPDATE SET ` + ownerColumn + ` = EXCLUDED.` + ownerColumn + `, type = 'from_iiko', updated_at = now()`
	for _, url := range set.urls {
		if _, err := tx.ExecContext(ctx, upsert, set.owners[url], url); err != nil {
			return fmt.Errorf("upsert %s %q: %w", table, url, err)
		}
	}

	del := `DELETE FROM ` + table + ` WHERE type = 'from_iiko' AND NOT (url = ANY($1))`
	if _, err := tx.ExecContext(ctx, del, set.urls); err != nil {
		return fmt.Errorf("delete stale %s: %w", table, err)
	}
	return nil
}

// clearTags removes every product tag (menu/v3 carries none) and the tags that
// no product uses any more.
func clearTags(ctx context.Context, tx *sqlx.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM product_tags`); err != nil {
		return fmt.Errorf("clear product tags: %w", err)
	}

	const unused = `DELETE FROM tags WHERE NOT EXISTS (SELECT 1 FROM product_tags pt WHERE pt.tag_id = tags.id)`
	if _, err := tx.ExecContext(ctx, unused); err != nil {
		return fmt.Errorf("delete unused tags: %w", err)
	}
	return nil
}
