package hotelbooking

import (
	_ "embed"
	"html/template"
	"strconv"
	"strings"
)

//go:embed confirmation.html
var confirmationHTML string

type confirmationDetail struct {
	Label string
	Value string
}

func renderConfirmation(t *template.Template, booking Reservation, roomName, phone string, links ...string) (plainText, html string, err error) {
	const title = "Бронирование подтверждено"
	const paymentNote = "Предоплата не требуется."

	details := make([]confirmationDetail, 0, 7)
	if booking.ReservationNumber != "" {
		details = append(details, confirmationDetail{"Номер бронирования", booking.ReservationNumber})
	}
	details = append(details,
		confirmationDetail{"Тип номера", roomName},
		confirmationDetail{"Заезд", booking.CheckIn},
		confirmationDetail{"Выезд", booking.CheckOut},
		confirmationDetail{"Гостей", strconv.Itoa(booking.Guests)},
		confirmationDetail{"Стоимость", booking.Price.TotalKopecks.String() + " RUB"},
		confirmationDetail{"Телефон гостиницы", phone},
	)

	var plain strings.Builder
	plain.WriteString(title + "\n")
	for _, detail := range details {
		plain.WriteString(detail.Label + ": " + detail.Value + "\n")
	}
	plain.WriteString(paymentNote)

	var cancellationURL string
	if len(links) > 0 && links[0] != "" {
		cancellationURL = links[0]
		plain.WriteString("\nОтменить бронирование: " + cancellationURL)
	}
	var styled strings.Builder
	if err := t.Execute(&styled, struct {
		Title           string
		Details         []confirmationDetail
		PaymentNote     string
		CancellationURL string
	}{title, details, paymentNote, cancellationURL}); err != nil {
		return "", "", err
	}
	return plain.String(), styled.String(), nil
}
