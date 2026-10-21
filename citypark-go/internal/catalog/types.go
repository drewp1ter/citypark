// Package catalog reads categories, products and their images.
package catalog

import (
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/salewindow"
)

// Image is a picture of a category or a product. ParentID is the owning
// category or product id; it is only used to attach images after loading.
type Image struct {
	ID       int64  `db:"id" json:"id"`
	URL      string `db:"url" json:"url"`
	Type     string `db:"type" json:"type"`
	ParentID int64  `db:"parent_id" json:"-"`
}

type Category struct {
	ID          int64   `db:"id"`
	Name        string  `db:"name"`
	Description string  `db:"description"`
	SaleStart   *int    `db:"sale_start"`
	SaleEnd     *int    `db:"sale_end"`
	SaleDays    string  `db:"sale_days"`
	IsHidden    bool    `db:"is_hidden"`
	Images      []Image `db:"-"`
}

type Product struct {
	ID            int64         `db:"id"`
	CategoryID    int64         `db:"category_id"`
	Name          string        `db:"name"`
	Description   string        `db:"description"`
	MeasureUnit   string        `db:"measure_unit"`
	Weight        float64       `db:"weight"`
	Proteins      float64       `db:"proteins"`
	Fats          float64       `db:"fats"`
	Carbohydrates float64       `db:"carbohydrates"`
	Calories      float64       `db:"calories"`
	Flags         int           `db:"flags"`
	Price         money.Kopecks `db:"price_kopecks"`
	SaleStart     *int          `db:"sale_start"`
	SaleEnd       *int          `db:"sale_end"`
	SaleDays      string        `db:"sale_days"`

	CategoryName      string `db:"category_name"`
	CategorySaleStart *int   `db:"category_sale_start"`
	CategorySaleEnd   *int   `db:"category_sale_end"`
	CategorySaleDays  string `db:"category_sale_days"`

	Images []Image `db:"-"`
}

// CategoryView is the JSON shape of a category.
type CategoryView struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	Description         string  `json:"description"`
	Images              []Image `json:"images"`
	IsAvailableForOrder bool    `json:"isAvailableForOrder"`
	SalesDays           string  `json:"salesDays"`
	StartTime           string  `json:"startTime"`
	EndTime             string  `json:"endTime"`
}

// ProductView is the JSON shape of a product.
type ProductView struct {
	ID                  int64         `json:"id"`
	CategoryID          int64         `json:"categoryId"`
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	MeasureUnit         string        `json:"measureUnit"`
	Weight              float64       `json:"weight"`
	Proteins            float64       `json:"proteins"`
	Fats                float64       `json:"fats"`
	Carbohydrates       float64       `json:"carbohydrates"`
	Calories            float64       `json:"calories"`
	Flags               int           `json:"flags"`
	Price               money.Kopecks `json:"price"`
	Images              []Image       `json:"images"`
	IsAvailableForOrder bool          `json:"isAvailableForOrder"`
	SalesDays           string        `json:"salesDays"`
	StartTime           string        `json:"startTime"`
	EndTime             string        `json:"endTime"`
}

func (c Category) Window() salewindow.Window {
	return salewindow.Window{Start: c.SaleStart, End: c.SaleEnd, Days: c.SaleDays}
}

func (p Product) Window() salewindow.Window {
	return salewindow.Window{Start: p.SaleStart, End: p.SaleEnd, Days: p.SaleDays}
}

// CategoryWindow is the sale window of the product's category.
func (p Product) CategoryWindow() salewindow.Window {
	return salewindow.Window{Start: p.CategorySaleStart, End: p.CategorySaleEnd, Days: p.CategorySaleDays}
}

// open reports whether the window allows an order at now.
func open(w salewindow.Window, now time.Time) bool {
	return w.OpenOnDay(now) && w.OpenAtTime(now)
}

// images returns imgs, or an empty slice when imgs is nil, so JSON shows [].
func images(imgs []Image) []Image {
	if imgs == nil {
		return []Image{}
	}
	return imgs
}

// View builds the JSON shape. now must already be in the restaurant's time zone.
func (c Category) View(now time.Time) CategoryView {
	w := c.Window()
	return CategoryView{
		ID:                  c.ID,
		Name:                c.Name,
		Description:         c.Description,
		Images:              images(c.Images),
		IsAvailableForOrder: open(w, now),
		SalesDays:           w.DaysString(),
		StartTime:           w.StartString(),
		EndTime:             w.EndString(),
	}
}

// View builds the JSON shape. now must already be in the restaurant's time zone.
// The product is available only when both its own window and its category's window are open.
func (p Product) View(now time.Time) ProductView {
	w := p.Window()
	return ProductView{
		ID:                  p.ID,
		CategoryID:          p.CategoryID,
		Name:                p.Name,
		Description:         p.Description,
		MeasureUnit:         p.MeasureUnit,
		Weight:              p.Weight,
		Proteins:            p.Proteins,
		Fats:                p.Fats,
		Carbohydrates:       p.Carbohydrates,
		Calories:            p.Calories,
		Flags:               p.Flags,
		Price:               p.Price,
		Images:              images(p.Images),
		IsAvailableForOrder: open(w, now) && open(p.CategoryWindow(), now),
		SalesDays:           w.DaysString(),
		StartTime:           w.StartString(),
		EndTime:             w.EndString(),
	}
}
