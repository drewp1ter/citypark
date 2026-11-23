package hotelbooking

import (
	"html"
	"strings"
	"testing"
)

func TestRenderConfirmation(t *testing.T) {
	worker := NewWorker(nil, nil, nil, nil)
	reservation := Reservation{
		CheckIn:  "2026-12-01",
		CheckOut: "2026-12-03",
		Guests:   2,
	}
	reservation.Price.TotalKopecks = 800050
	const room = "Люкс & <вид>"
	const phone = "+7 900 123-45-67"
	plain, styled, err := renderConfirmation(worker.confirmation, reservation, room, phone)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Бронирование подтверждено", "Тип номера", "2026-12-01", "2026-12-03", "Гостей", "8000.50 RUB", phone, "Предоплата не требуется."} {
		if !strings.Contains(plain, want) || !strings.Contains(html.UnescapeString(styled), want) {
			t.Errorf("expected %q in both versions", want)
		}
	}
	if strings.Contains(plain, "Номер бронирования") || strings.Contains(styled, "Номер бронирования") {
		t.Error("missing reservation number should omit its label")
	}
	if !strings.Contains(plain, room) || !strings.Contains(styled, "Люкс &amp; &lt;вид&gt;") || strings.Contains(styled, room) {
		t.Error("room name should be escaped in the HTML version")
	}
}
