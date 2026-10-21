// Package seed fills an empty development database with fixed sample data.
package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/order"
	"github.com/drewp1ter/citypark-go/internal/store"
)

// ErrNotEmpty is returned by Run when the categories table already has rows.
var ErrNotEmpty = errors.New("seed: database is not empty")

type Summary struct {
	Users, Categories, Products, DeliveryPrices, Orders int
}

const (
	userPassword = "password123"
	imageURLFmt  = "https://placehold.co/600x400?text=%d"
)

type productSeed struct {
	name   string
	price  money.Kopecks
	weight float64
	tags   []string
}

type categorySeed struct {
	name      string
	saleStart *int // minutes from midnight
	saleEnd   *int
	saleDays  string
	products  []productSeed
}

func minutes(v int) *int { return &v }

var deliveryPrices = []struct {
	name  string
	price money.Kopecks
}{
	{"Сочи", money.FromRubles(150)},
	{"Адлер", money.FromRubles(250)},
	{"Красная Поляна", money.FromRubles(500)},
}

var categories = []categorySeed{
	{
		name: "Грузинская кухня",
		products: []productSeed{
			{"Хачапури по-аджарски", money.FromRubles(550), 0.4, []string{"вегетарианское"}},
			{"Хинкали", money.FromRubles(450), 0.35, []string{"мясо"}},
			{"Харчо", money.FromRubles(350), 0.3, []string{"острое", "мясо"}},
			{"Лобио", money.FromRubles(300), 0.3, []string{"вегетарианское"}},
		},
	},
	{
		name: "Мангал",
		products: []productSeed{
			{"Шашлык из свинины", money.FromRubles(650), 0.3, []string{"мясо"}},
			{"Шашлык из баранины", money.FromRubles(1200), 0.3, []string{"мясо"}},
			{"Люля-кебаб", money.FromRubles(550), 0.25, []string{"мясо", "острое"}},
			{"Овощи на мангале", money.FromRubles(400), 0.4, []string{"вегетарианское"}},
		},
	},
	{
		name:      "Пицца",
		saleStart: minutes(12 * 60),
		saleEnd:   minutes(23 * 60),
		products: []productSeed{
			{"Маргарита", money.FromRubles(450), 0.45, []string{"вегетарианское"}},
			{"Пепперони", money.FromRubles(590), 0.5, []string{"мясо", "острое"}},
			{"Четыре сыра", money.FromRubles(620), 0.45, []string{"вегетарианское"}},
			{"Мясная", money.FromRubles(700), 0.5, []string{"мясо"}},
		},
	},
	{
		name:      "Завтраки",
		saleStart: minutes(9 * 60),
		saleEnd:   minutes(12 * 60),
		saleDays:  "1111100",
		products: []productSeed{
			{"Омлет с овощами", money.FromRubles(250), 0.25, []string{"вегетарианское"}},
			{"Сырники", money.FromRubles(220), 0.2, []string{"вегетарианское"}},
			{"Овсяная каша", money.FromRubles(150), 0.3, nil},
			{"Яичница с беконом", money.FromRubles(280), 0.25, []string{"мясо"}},
		},
	},
}

// Run inserts the sample data in one transaction.
// It returns ErrNotEmpty when the categories table already has rows.
func Run(ctx context.Context, db *sqlx.DB) (Summary, error) {
	var sum Summary
	err := store.InTx(ctx, db, func(tx *sqlx.Tx) error {
		var existing int
		if err := tx.GetContext(ctx, &existing, `SELECT count(*) FROM categories`); err != nil {
			return fmt.Errorf("count categories: %w", err)
		}
		if existing > 0 {
			return ErrNotEmpty
		}

		userID, err := insertUser(ctx, tx)
		if err != nil {
			return err
		}
		sum.Users = 1

		for _, dp := range deliveryPrices {
			_, err := tx.ExecContext(ctx,
				`INSERT INTO delivery_prices (name, price_kopecks) VALUES ($1, $2)`,
				dp.name, int64(dp.price))
			if err != nil {
				return fmt.Errorf("insert delivery price %q: %w", dp.name, err)
			}
			sum.DeliveryPrices++
		}

		// productIDs and prices keep insertion order, so orders can pick products by index.
		var productIDs []int64
		var prices []money.Kopecks
		tagIDs := map[string]int64{}
		for i, c := range categories {
			var categoryID int64
			err := tx.GetContext(ctx, &categoryID,
				`INSERT INTO categories (name, sort_order, sale_start, sale_end, sale_days)
				 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
				c.name, i, c.saleStart, c.saleEnd, c.saleDays)
			if err != nil {
				return fmt.Errorf("insert category %q: %w", c.name, err)
			}
			sum.Categories++

			for j, p := range c.products {
				productID, err := insertProduct(ctx, tx, categoryID, j, p, tagIDs)
				if err != nil {
					return err
				}
				productIDs = append(productIDs, productID)
				prices = append(prices, p.price)
				sum.Products++
			}
		}

		var addressID int64
		err = tx.GetContext(ctx, &addressID,
			`INSERT INTO addresses (city, address) VALUES ($1, $2) RETURNING id`,
			"Сочи", "ул. Навагинская, 1")
		if err != nil {
			return fmt.Errorf("insert address: %w", err)
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO user_addresses (user_id, address_id) VALUES ($1, $2)`, userID, addressID)
		if err != nil {
			return fmt.Errorf("link address: %w", err)
		}

		orders := order.NewStore(tx)
		delivery := []struct {
			addressID *int64
			price     money.Kopecks
			products  []int // indexes into productIDs
		}{
			{&addressID, deliveryPrices[0].price, []int{0, 4}},
			{nil, 0, []int{8}},
		}
		for _, o := range delivery {
			p := order.InsertParams{
				CustomerID:    userID,
				AddressID:     o.addressID,
				PaymentType:   order.PaymentCash,
				DeliveryPrice: o.price,
			}
			inputs := make([]order.ItemInput, 0, len(o.products))
			for _, idx := range o.products {
				inputs = append(inputs, order.ItemInput{ProductID: productIDs[idx], Amount: 1})
				p.Items = append(p.Items, order.InsertItem{
					ProductID: productIDs[idx], Amount: 1, OfferedPrice: prices[idx],
				})
			}
			p.ItemsHash = order.ItemsHash(inputs)
			if _, err := orders.Insert(ctx, p); err != nil {
				return fmt.Errorf("insert order: %w", err)
			}
			sum.Orders++
		}
		return nil
	})
	if err != nil {
		return Summary{}, err
	}
	return sum, nil
}

func insertUser(ctx context.Context, tx *sqlx.Tx) (int64, error) {
	hash, err := auth.HashPassword(userPassword)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}
	var id int64
	err = tx.GetContext(ctx, &id,
		`INSERT INTO users (name, phone, email, password_hash, role)
		 VALUES ($1, $2, $3, $4, 'user') RETURNING id`,
		"Тестовый пользователь", "+79990000001", "user@example.com", hash)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	return id, nil
}

// insertProduct writes the product, its image and its tag links. tagIDs caches tag ids by name.
func insertProduct(ctx context.Context, tx *sqlx.Tx, categoryID int64, sortOrder int, p productSeed, tagIDs map[string]int64) (int64, error) {
	var id int64
	err := tx.GetContext(ctx, &id,
		`INSERT INTO products (category_id, sort_order, name, measure_unit, weight, price_kopecks)
		 VALUES ($1, $2, $3, 'порц', $4, $5) RETURNING id`,
		categoryID, sortOrder, p.name, p.weight, int64(p.price))
	if err != nil {
		return 0, fmt.Errorf("insert product %q: %w", p.name, err)
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO product_images (product_id, url, type) VALUES ($1, $2, 'seed')`,
		id, fmt.Sprintf(imageURLFmt, id))
	if err != nil {
		return 0, fmt.Errorf("insert image for %q: %w", p.name, err)
	}
	for _, tag := range p.tags {
		tagID, ok := tagIDs[tag]
		if !ok {
			err := tx.GetContext(ctx, &tagID, `INSERT INTO tags (tag) VALUES ($1) RETURNING id`, tag)
			if err != nil {
				return 0, fmt.Errorf("insert tag %q: %w", tag, err)
			}
			tagIDs[tag] = tagID
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO product_tags (product_id, tag_id) VALUES ($1, $2)`, id, tagID)
		if err != nil {
			return 0, fmt.Errorf("link tag %q: %w", tag, err)
		}
	}
	return id, nil
}
