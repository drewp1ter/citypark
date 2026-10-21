package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/money"
)

const testToken = "123:secret-token"

func TestSendMessage(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if want := "/bot" + testToken + "/sendMessage"; r.URL.Path != want {
			t.Errorf("path = %s, want %s", r.URL.Path, want)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		want := map[string]string{"chat_id": "-100500", "parse_mode": "Markdown", "text": "hello"}
		if len(body) != len(want) {
			t.Errorf("body = %v, want %v", body, want)
		}
		for k, v := range want {
			if body[k] != v {
				t.Errorf("body[%s] = %q, want %q", k, body[k], v)
			}
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, testToken, nil)
	if err := c.SendMessage(context.Background(), "-100500", "hello"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if !called {
		t.Fatal("server was not called")
	}
}

func TestSendMessageClientError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"ok":false,"description":"chat not found"}`))
	}))
	defer srv.Close()

	err := NewClient(srv.URL, testToken, nil).SendMessage(context.Background(), "1", "x")
	if err == nil || err.Error() != "telegram: chat not found" {
		t.Fatalf("err = %v, want %q", err, "telegram: chat not found")
	}
}

func TestSendMessageClientErrorBadBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	err := NewClient(srv.URL, testToken, nil).SendMessage(context.Background(), "1", "x")
	if err == nil || err.Error() != "telegram: status 404" {
		t.Fatalf("err = %v, want %q", err, "telegram: status 404")
	}
}

func TestSendMessageServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	err := NewClient(srv.URL, testToken, nil).SendMessage(context.Background(), "1", "x")
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("err = %v, want it to contain %q", err, "status 502")
	}
}

func TestSendMessageNoTokenInError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing listens on url any more

	err := NewClient(url, testToken, nil).SendMessage(context.Background(), "1", "x")
	if err == nil {
		t.Fatal("expected an error from a closed server")
	}
	if strings.Contains(err.Error(), testToken) || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaks the token: %v", err)
	}
}

func TestSendMessageNoTokenInBuildError(t *testing.T) {
	// A control character in the URL makes http.NewRequestWithContext fail.
	c := NewClient("http://bad\x7fhost", testToken, nil)
	err := c.SendMessage(context.Background(), "1", "x")
	if err == nil {
		t.Fatal("expected a build error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaks the token: %v", err)
	}
}

func TestFormatOrderDelivery(t *testing.T) {
	got := FormatOrder(OrderMessage{
		ID:          15,
		Name:        "Иван",
		Phone:       "+79991234567",
		Address:     "Сочи, ул. Ленина, 1",
		PaymentType: "cash",
		Total:       money.FromRubles(640),
		Delivery:    money.FromRubles(150),
		Comment:     "Без лука",
		Items: []OrderItem{
			{Category: "Мангал", Name: "Шашлык", Amount: 2},
			{Category: "Пицца", Name: "Маргарита", Amount: 1},
		},
	})
	want := "```\n" +
		"Номер заказа:   15\n" +
		"Имя:            Иван\n" +
		"Телефон:        +79991234567\n" +
		"Доставка:       Да\n" +
		"Адрес доставки: Сочи, ул. Ленина, 1\n" +
		"Оплата:         наличными\n" +
		"Сумма заказа:   640\n" +
		"Сумма доставки: 150\n" +
		"Итого:          790\n" +
		"Комментарий:    Без лука\n" +
		"\n" +
		"Заказ:\n" +
		"Мангал: Шашлык * 2\n" +
		"Пицца: Маргарита * 1\n" +
		"```"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatOrderPickup(t *testing.T) {
	got := FormatOrder(OrderMessage{
		ID:          15,
		Name:        "Иван",
		Phone:       "+79991234567",
		PaymentType: "online",
		Total:       money.FromRubles(640),
		Delivery:    money.FromRubles(150), // ignored for pickup
		Items:       []OrderItem{{Category: "Мангал", Name: "Шашлык", Amount: 2}},
	})
	want := "```\n" +
		"Номер заказа:   15\n" +
		"Имя:            Иван\n" +
		"Телефон:        +79991234567\n" +
		"Доставка:       Нет\n" +
		"Оплата:         онлайн\n" +
		"Итого:          640\n" +
		"Комментарий:    \n" +
		"\n" +
		"Заказ:\n" +
		"Мангал: Шашлык * 2\n" +
		"```"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatOrderFractions(t *testing.T) {
	got := FormatOrder(OrderMessage{
		ID:       1,
		Address:  "Сочи, ул. Ленина, 1",
		Total:    59950,
		Delivery: 15050,
	})
	for _, want := range []string{
		"Сумма заказа:   599.50\n",
		"Сумма доставки: 150.50\n",
		"Итого:          750\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
}

func TestFormatOrderBackticks(t *testing.T) {
	got := FormatOrder(OrderMessage{
		ID:      1,
		Name:    "Ва`ня",
		Address: "ул. `Ленина`",
		Comment: "без `лука`",
		Items:   []OrderItem{{Category: "Кат`1", Name: "Блюдо`2", Amount: 1}},
	})
	for _, want := range []string{
		"Имя:            Ва'ня\n",
		"Адрес доставки: ул. 'Ленина'\n",
		"Комментарий:    без 'лука'\n",
		"Кат'1: Блюдо'2 * 1\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
	// Only the opening and closing fences may contain backticks.
	if n := strings.Count(got, "`"); n != 6 {
		t.Errorf("backtick count = %d, want 6 (two fences)", n)
	}
}

func TestFormatOrderUnknownPaymentType(t *testing.T) {
	got := FormatOrder(OrderMessage{ID: 1, PaymentType: "card"})
	if want := "Оплата:         card\n"; !strings.Contains(got, want) {
		t.Errorf("output does not contain %q:\n%s", want, got)
	}
}
