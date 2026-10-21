package telegram

import (
	"fmt"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/money"
)

// OrderMessage is everything the text needs. It is filled by the caller.
type OrderMessage struct {
	ID          int64
	Name        string // customer name
	Phone       string
	Address     string        // "city, street"; "" means pickup
	PaymentType string        // "cash" or "online"
	Total       money.Kopecks // sum of items, without delivery
	Delivery    money.Kopecks
	Comment     string
	Items       []OrderItem
}

// OrderItem is one line of the order.
type OrderItem struct {
	Category string
	Name     string
	Amount   int
}

// labelWidth is the width of the label column, in characters.
const labelWidth = 16

// FormatOrder builds the Markdown code block that staff receive.
func FormatOrder(m OrderMessage) string {
	delivery := m.Address != ""

	var b strings.Builder
	b.WriteString("```\n")
	line := func(label, value string) {
		// %-16s counts runes, so Cyrillic labels are padded correctly.
		fmt.Fprintf(&b, "%-*s%s\n", labelWidth, label+":", value)
	}

	line("Номер заказа", fmt.Sprint(m.ID))
	line("Имя", safe(m.Name))
	line("Телефон", m.Phone)
	if delivery {
		line("Доставка", "Да")
		line("Адрес доставки", safe(m.Address))
	} else {
		line("Доставка", "Нет")
	}
	line("Оплата", paymentText(m.PaymentType))

	total := m.Total
	if delivery {
		line("Сумма заказа", m.Total.String())
		line("Сумма доставки", m.Delivery.String())
		total += m.Delivery
	}
	line("Итого", total.String())
	line("Комментарий", safe(m.Comment))

	b.WriteString("\nЗаказ:\n")
	for _, it := range m.Items {
		fmt.Fprintf(&b, "%s: %s * %d\n", safe(it.Category), safe(it.Name), it.Amount)
	}
	b.WriteString("```")
	return b.String()
}

func paymentText(t string) string {
	switch t {
	case "cash":
		return "наличными"
	case "online":
		return "онлайн"
	}
	return t
}

// safe replaces backticks so user text cannot close the Markdown code block.
func safe(s string) string {
	return strings.ReplaceAll(s, "`", "'")
}
