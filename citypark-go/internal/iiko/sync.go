package iiko

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/salewindow"
	"github.com/drewp1ter/citypark-go/internal/store"
)

// Source provides the menu. *Client implements it.
type Source interface {
	Fetch(ctx context.Context) (*Menu, error)
}

var _ Source = (*Client)(nil)

// Result reports how many categories and products the menu contributed.
type Result struct {
	TotalCategories int `json:"totalCategories"`
	TotalProducts   int `json:"totalProducts"`
}

// Syncer copies the iiko external menu into the menus, categories and
// products tables.
type Syncer struct {
	db  *sqlx.DB
	src Source
}

// NewSyncer creates a Syncer.
func NewSyncer(db *sqlx.DB, src Source) *Syncer {
	return &Syncer{db: db, src: src}
}

// Sync downloads the menu and applies it in one transaction.
func (s *Syncer) Sync(ctx context.Context) (Result, error) {
	menu, err := s.src.Fetch(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("fetch iiko menu: %w", err)
	}

	var res Result
	err = store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		menuID, err := syncMenu(ctx, tx, menu)
		if err != nil {
			return fmt.Errorf("sync menu: %w", err)
		}
		categoryIDs, err := syncCategories(ctx, tx, menuID, menu.ItemsGroups)
		if err != nil {
			return fmt.Errorf("sync categories: %w", err)
		}
		productIDs, err := syncProducts(ctx, tx, menu, categoryIDs)
		if err != nil {
			return fmt.Errorf("sync products: %w", err)
		}
		if err := syncImages(ctx, tx, menu, productIDs); err != nil {
			return fmt.Errorf("sync images: %w", err)
		}
		if err := clearTags(ctx, tx); err != nil {
			return fmt.Errorf("clear tags: %w", err)
		}
		res = Result{TotalCategories: len(categoryIDs), TotalProducts: len(productIDs)}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

// syncMenu upserts the menu row and returns its id.
func syncMenu(ctx context.Context, tx *sqlx.Tx, menu *Menu) (int64, error) {
	const upsert = `
INSERT INTO menus (iiko_id, name) VALUES ($1, $2)
ON CONFLICT (iiko_id) DO UPDATE SET name = EXCLUDED.name, updated_at = now()
RETURNING id`
	var id int64
	if err := tx.GetContext(ctx, &id, upsert, menu.ID, menu.Name); err != nil {
		return 0, fmt.Errorf("upsert menu %q: %w", menu.ID, err)
	}
	return id, nil
}

// syncCategories upserts one category per items group and soft-deletes iiko
// categories missing from the menu. It returns a map from iiko group id to
// categories.id.
func syncCategories(ctx context.Context, tx *sqlx.Tx, menuID int64, groups []ItemsGroup) (map[string]int64, error) {
	const upsert = `
INSERT INTO categories (iiko_id, menu_id, name, description, sort_order, is_deleted, sale_start, sale_end, sale_days)
VALUES ($1, $2, $3, '', $4, false, $5, $6, $7)
ON CONFLICT (iiko_id) DO UPDATE SET
    menu_id = EXCLUDED.menu_id, name = EXCLUDED.name, description = '', sort_order = EXCLUDED.sort_order,
    is_deleted = false, sale_start = EXCLUDED.sale_start, sale_end = EXCLUDED.sale_end,
    sale_days = EXCLUDED.sale_days, updated_at = now()
RETURNING id`

	w := salewindow.Window{}
	ids := make(map[string]int64, len(groups))
	keptIikoIDs := make([]string, 0, len(groups))
	for i, g := range groups {
		var id int64
		err := tx.GetContext(ctx, &id, upsert, g.ID, menuID, strings.TrimSpace(g.Name), i, w.Start, w.End, w.Days)
		if err != nil {
			return nil, fmt.Errorf("upsert category %q: %w", g.ID, err)
		}
		ids[g.ID] = id
		keptIikoIDs = append(keptIikoIDs, g.ID)
	}

	if err := softDeleteMissing(ctx, tx, "categories", keptIikoIDs); err != nil {
		return nil, err
	}
	return ids, nil
}

// pickSize chooses the size an item is sold at: the first default size, or
// else the first one. ok is false when the item has no sizes or the chosen
// size has no price, which means the item is not for sale.
func pickSize(sizes []ItemSizePrice) (size ItemSizePrice, ok bool) {
	if len(sizes) == 0 {
		return ItemSizePrice{}, false
	}
	size = sizes[0]
	for _, sz := range sizes {
		if sz.IsDefault {
			size = sz
			break
		}
	}
	return size, size.Price != nil
}

// syncProducts upserts the priced items of every group and soft-deletes iiko
// products missing from the menu. It returns a map from iiko product id to
// products.id.
func syncProducts(ctx context.Context, tx *sqlx.Tx, menu *Menu, categoryIDs map[string]int64) (map[string]int64, error) {
	const upsert = `
INSERT INTO products (iiko_id, category_id, name, description, measure_unit, weight, sort_order, is_deleted,
                      fats, proteins, carbohydrates, calories, price_kopecks, sale_start, sale_end, sale_days)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (iiko_id) DO UPDATE SET
    category_id = EXCLUDED.category_id, name = EXCLUDED.name, description = EXCLUDED.description,
    measure_unit = EXCLUDED.measure_unit, weight = EXCLUDED.weight, sort_order = EXCLUDED.sort_order,
    is_deleted = EXCLUDED.is_deleted, fats = EXCLUDED.fats, proteins = EXCLUDED.proteins,
    carbohydrates = EXCLUDED.carbohydrates, calories = EXCLUDED.calories,
    price_kopecks = EXCLUDED.price_kopecks, sale_start = EXCLUDED.sale_start,
    sale_end = EXCLUDED.sale_end, sale_days = EXCLUDED.sale_days, updated_at = now()
RETURNING id`

	measureUnits := make(map[string]string, len(menu.Products))
	for _, p := range menu.Products {
		measureUnits[p.ID] = p.MeasureUnit
	}

	w := salewindow.Window{}
	ids := make(map[string]int64)
	keptIikoIDs := make([]string, 0)
	for _, g := range menu.ItemsGroups {
		for j, item := range g.Items {
			size, ok := pickSize(item.SizePrices)
			if !ok {
				continue
			}
			price := money.FromRubles(*size.Price)

			var id int64
			err := tx.GetContext(ctx, &id, upsert,
				item.ProductID, categoryIDs[g.ID], item.Name, "", measureUnits[item.ProductID], size.Weight/1000, j, false,
				size.Nutritions.Fats, size.Nutritions.Proteins, size.Nutritions.Carbs, size.Nutritions.Energy,
				int64(price), w.Start, w.End, w.Days)
			if err != nil {
				return nil, fmt.Errorf("upsert product %q: %w", item.ProductID, err)
			}
			ids[item.ProductID] = id
			keptIikoIDs = append(keptIikoIDs, item.ProductID)
		}
	}

	if err := softDeleteMissing(ctx, tx, "products", keptIikoIDs); err != nil {
		return nil, err
	}
	return ids, nil
}

// softDeleteMissing marks every iiko row of table that is not in keptIikoIDs as
// deleted. Rows without iiko_id (created by hand) are never touched. table is
// always a literal from this package, never user input.
func softDeleteMissing(ctx context.Context, tx *sqlx.Tx, table string, keptIikoIDs []string) error {
	if keptIikoIDs == nil {
		keptIikoIDs = []string{} // a nil slice would be SQL NULL and match nothing
	}
	query := `UPDATE ` + table + ` SET is_deleted = true, updated_at = now()
WHERE iiko_id IS NOT NULL AND NOT is_deleted AND NOT (iiko_id = ANY($1))`
	if _, err := tx.ExecContext(ctx, query, keptIikoIDs); err != nil {
		return fmt.Errorf("soft-delete missing %s: %w", table, err)
	}
	return nil
}
