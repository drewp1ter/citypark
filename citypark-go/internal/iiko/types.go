package iiko

// Menu is an external menu from menu/v3/by_id.
type Menu struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	ItemsGroups []ItemsGroup  `json:"itemsGroups"`
	Products    []MenuProduct `json:"products"`
}

type ItemsGroup struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	IsHidden bool       `json:"isHidden"`
	Items    []MenuItem `json:"items"`
}

type MenuItem struct {
	ProductID  string          `json:"productId"`
	Name       string          `json:"name"`
	SizePrices []ItemSizePrice `json:"sizePrices"`
}

// ItemSizePrice: Price is nil when iiko sends no price; such an item is not for sale.
type ItemSizePrice struct {
	SizeName       string     `json:"sizeName"`
	IsDefault      bool       `json:"isDefault"`
	Price          *float64   `json:"price"`
	ButtonImageURL string     `json:"buttonImageUrl"`
	Weight         float64    `json:"weight"`
	Nutritions     Nutritions `json:"nutritions"`
	SKU            string     `json:"sku"`
	MenuItemSizeID string     `json:"menuItemSizeId"`
}

// Nutritions are zero when iiko sends none (null or missing).
type Nutritions struct {
	Fats     float64 `json:"fats"`
	Proteins float64 `json:"proteins"`
	Carbs    float64 `json:"carbs"`
	Energy   float64 `json:"energy"`
}

type MenuProduct struct {
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	MeasureUnit string `json:"measureUnit"`
}
