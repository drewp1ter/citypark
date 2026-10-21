package catalog

import (
	"context"

	"github.com/drewp1ter/citypark-go/internal/store"
)

const categoryColumns = `id, name, description, sale_start, sale_end, sale_days, is_hidden`

// productSelect is the base query of every product read. Callers append conditions with AND.
const productSelect = `
SELECT p.id, p.category_id, p.name, p.description, p.measure_unit, p.weight, p.proteins, p.fats,
       p.carbohydrates, p.calories, p.flags, p.price_kopecks, p.sale_start, p.sale_end, p.sale_days,
       c.name AS category_name, c.sale_start AS category_sale_start,
       c.sale_end AS category_sale_end, c.sale_days AS category_sale_days
FROM products p JOIN categories c ON c.id = p.category_id
WHERE NOT p.is_deleted AND NOT c.is_deleted`

// searchCondition matches a product when any word matches its name, its category name or a tag.
const searchCondition = `
  AND (
       p.search @@ ANY (ARRAY(SELECT plainto_tsquery('russian', w) FROM unnest($1::text[]) AS w))
    OR c.search @@ ANY (ARRAY(SELECT plainto_tsquery('russian', w) FROM unnest($1::text[]) AS w))
    OR EXISTS (SELECT 1 FROM product_tags pt JOIN tags t ON t.id = pt.tag_id
               WHERE pt.product_id = p.id AND t.tag = ANY($1::text[]))
  )`

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

// ListCategories returns the categories that are not deleted, ordered by sort_order, id.
// Images are loaded. Never returns nil.
func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	list := []Category{}
	err := s.db.SelectContext(ctx, &list,
		`SELECT `+categoryColumns+` FROM categories WHERE NOT is_deleted AND NOT is_hidden ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	if err := s.attachCategoryImages(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}

// CategoryByID returns store.ErrNotFound when the category is missing or deleted. Images are loaded.
func (s *Store) CategoryByID(ctx context.Context, id int64) (Category, error) {
	var c Category
	err := s.db.GetContext(ctx, &c,
		`SELECT `+categoryColumns+` FROM categories WHERE id = $1 AND NOT is_deleted`, id)
	if err != nil {
		return Category{}, store.NotFound(err)
	}
	one := []Category{c}
	if err := s.attachCategoryImages(ctx, one); err != nil {
		return Category{}, err
	}
	return one[0], nil
}

// ProductsByCategory returns the products of a category, ordered by sort_order, id.
// Deleted products and products of a deleted category are skipped. Images are loaded.
func (s *Store) ProductsByCategory(ctx context.Context, categoryID int64) ([]Product, error) {
	list := []Product{}
	err := s.db.SelectContext(ctx, &list,
		productSelect+` AND p.category_id = $1 ORDER BY p.sort_order, p.id`, categoryID)
	if err != nil {
		return nil, err
	}
	if err := s.attachProductImages(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}

// SearchProducts returns the products matching any of words (see searchCondition),
// or all products when words is empty. Images are loaded.
func (s *Store) SearchProducts(ctx context.Context, words []string, limit, offset int) ([]Product, error) {
	list := []Product{}
	var err error
	if len(words) == 0 {
		err = s.db.SelectContext(ctx, &list,
			productSelect+` ORDER BY p.sort_order, p.id LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		err = s.db.SelectContext(ctx, &list,
			productSelect+searchCondition+` ORDER BY p.sort_order, p.id LIMIT $2 OFFSET $3`,
			words, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	if err := s.attachProductImages(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}

// ProductsByIDs returns the requested products that are not deleted, in no particular order.
// Images are not loaded.
func (s *Store) ProductsByIDs(ctx context.Context, ids []int64) ([]Product, error) {
	list := []Product{}
	if len(ids) == 0 {
		return list, nil
	}
	err := s.db.SelectContext(ctx, &list, productSelect+` AND p.id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Store) attachCategoryImages(ctx context.Context, list []Category) error {
	ids := make([]int64, len(list))
	for i, c := range list {
		ids[i] = c.ID
	}
	images, err := s.loadImages(ctx, "category_images", "category_id", ids)
	if err != nil {
		return err
	}
	for i := range list {
		list[i].Images = images[list[i].ID]
	}
	return nil
}

func (s *Store) attachProductImages(ctx context.Context, list []Product) error {
	ids := make([]int64, len(list))
	for i, p := range list {
		ids[i] = p.ID
	}
	images, err := s.loadImages(ctx, "product_images", "product_id", ids)
	if err != nil {
		return err
	}
	for i := range list {
		list[i].Images = images[list[i].ID]
	}
	return nil
}

// loadImages reads the images of the given parents with one query and groups them by parent id.
// table and parentColumn are constants of this package, never request data.
func (s *Store) loadImages(ctx context.Context, table, parentColumn string, ids []int64) (map[int64][]Image, error) {
	byParent := map[int64][]Image{}
	if len(ids) == 0 {
		return byParent, nil
	}
	var rows []Image
	err := s.db.SelectContext(ctx, &rows,
		`SELECT id, url, type, `+parentColumn+` AS parent_id FROM `+table+
			` WHERE `+parentColumn+` = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return nil, err
	}
	for _, img := range rows {
		byParent[img.ParentID] = append(byParent[img.ParentID], img)
	}
	return byParent, nil
}
