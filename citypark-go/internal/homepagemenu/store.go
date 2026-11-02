// Package homepagemenu stores homepage selections independently of the iiko catalog.
package homepagemenu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

type Category struct {
	CategoryID int64   `json:"categoryId"`
	ProductIDs []int64 `json:"productIds"`
}

type Configuration struct {
	Categories []Category `json:"categories"`
}

type Store struct{ db *sqlx.DB }

func NewStore(db *sqlx.DB) *Store { return &Store{db: db} }

func (s *Store) Get(ctx context.Context) (Configuration, error) {
	var data []byte
	if err := s.db.GetContext(ctx, &data, `SELECT data FROM homepage_menu WHERE id=1`); err != nil {
		return Configuration{}, store.NotFound(err)
	}
	var out Configuration
	err := json.Unmarshal(data, &out)
	return out, err
}

const validationMessage = "Ошибка валидации, пожалуйста проверьте введенные вами данные"

func invalid(field, message string) error {
	e := httpx.NewError(http.StatusBadRequest, validationMessage)
	e.Fields = map[string]string{field: message}
	return e
}

// Put validates catalog membership and replaces the configuration in one transaction.
// Row locks keep a concurrent sync from changing membership between validation and save.
func (s *Store) Put(ctx context.Context, in Configuration) error {
	if err := check(in); err != nil {
		return err
	}
	return store.InTx(ctx, s.db, func(tx *sqlx.Tx) error {
		for i, category := range in.Categories {
			prefix := fmt.Sprintf("categories[%d]", i)
			var id int64
			err := tx.GetContext(ctx, &id, `SELECT id FROM categories WHERE id=$1 AND NOT is_deleted AND NOT is_hidden FOR SHARE`, category.CategoryID)
			if errors.Is(store.NotFound(err), store.ErrNotFound) {
				return invalid(prefix+".categoryId", "Категория недоступна")
			}
			if err != nil {
				return err
			}
			for _, productID := range category.ProductIDs {
				err := tx.GetContext(ctx, &id, `SELECT id FROM products WHERE id=$1 AND category_id=$2 AND NOT is_deleted FOR SHARE`, productID, category.CategoryID)
				if errors.Is(store.NotFound(err), store.ErrNotFound) {
					return invalid(prefix+".productIds", "Блюдо недоступно или относится к другой категории")
				}
				if err != nil {
					return err
				}
			}
		}
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		var id int
		err = tx.GetContext(ctx, &id, `UPDATE homepage_menu SET data=$1, updated_at=now() WHERE id=1 RETURNING id`, data)
		return store.NotFound(err)
	})
}

func check(in Configuration) error {
	if in.Categories == nil {
		return invalid("categories", "Укажите список категорий")
	}
	seen := map[int64]bool{}
	for i, category := range in.Categories {
		prefix := fmt.Sprintf("categories[%d]", i)
		if category.CategoryID <= 0 || seen[category.CategoryID] {
			return invalid(prefix+".categoryId", "Укажите уникальную категорию")
		}
		seen[category.CategoryID] = true
		if category.ProductIDs == nil || len(category.ProductIDs) > 3 {
			return invalid(prefix+".productIds", "Укажите список из не более трёх блюд")
		}
		products := map[int64]bool{}
		for _, id := range category.ProductIDs {
			if id <= 0 || products[id] {
				return invalid(prefix+".productIds", "Укажите уникальные блюда")
			}
			products[id] = true
		}
	}
	return nil
}
