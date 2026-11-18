package iiko

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// syncImages saves the button image of every saved product and deletes the
// other iiko images. menu/v3 carries no category images, so those are all
// deleted. Images of another type are never touched.
func syncImages(ctx context.Context, tx *sqlx.Tx, menu *Menu, productIDs map[string]int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM category_images WHERE type = 'from_iiko'`); err != nil {
		return fmt.Errorf("delete stale category_images: %w", err)
	}

	// Keep row IDs rather than URLs: multiple dishes may share the same URL.
	keptIDs := make([]int64, 0)
	const upsert = `INSERT INTO product_images (product_id, url, type) VALUES ($1, $2, 'from_iiko')
ON CONFLICT (product_id, url) DO UPDATE SET type = 'from_iiko', updated_at = now()
RETURNING id`
	for _, g := range menu.ItemsGroups {
		for _, item := range g.Items {
			id, ok := productIDs[item.ProductID]
			if !ok {
				continue // not for sale, so it was not saved
			}
			for _, sizePrice := range item.SizePrices {
				if sizePrice.ButtonImageURL != "" {
					var imageID int64
					if err := tx.GetContext(ctx, &imageID, upsert, id, sizePrice.ButtonImageURL); err != nil {
						return fmt.Errorf("upsert product_images %q: %w", sizePrice.ButtonImageURL, err)
					}
					keptIDs = append(keptIDs, imageID)
					break
				}
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM product_images WHERE type = 'from_iiko' AND NOT (id = ANY($1))`, keptIDs); err != nil {
		return fmt.Errorf("delete stale product_images: %w", err)
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
