package hotelbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"time"

	"github.com/drewp1ter/citypark-go/internal/settings"
)

type ContactReader interface {
	Get(context.Context) (settings.Settings, time.Time, error)
}

type confirmationMailer interface {
	SendAlternative(ctx context.Context, to, subject, plainText, html string) error
}

type Worker struct {
	service        *Service
	mailer         confirmationMailer
	confirmation   *template.Template
	contacts       ContactReader
	log            *slog.Logger
	telegram       plainMessenger
	telegramChatID string
}

func NewWorker(service *Service, m confirmationMailer, contacts ContactReader, log *slog.Logger) *Worker {
	return &Worker{
		service:      service,
		mailer:       m,
		confirmation: template.Must(template.New("booking confirmation").Parse(confirmationHTML)),
		contacts:     contacts,
		log:          log,
	}
}
func (w *Worker) Run(ctx context.Context) {
	if w.telegram != nil {
		done := make(chan struct{})
		go func() {
			defer close(done)
			runBookingPoller(ctx, w.tickTelegram)
		}()
		defer func() { <-done }()
	}
	runBookingPoller(ctx, w.Tick)
}

func runBookingPoller(ctx context.Context, tick func(context.Context)) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Tick processes a bounded batch. Database claims survive crashes and prevent
// concurrent workers sending the same outbox row during its lease.
func (w *Worker) Tick(ctx context.Context) {
	for i := 0; i < 100 && ctx.Err() == nil; i++ {
		a, err := w.service.store.claimRecovery(ctx, w.service.clock.Now(), randomID())
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			w.log.Warn("hotel booking recovery storage unavailable")
			break
		}
		w.service.dispatch(ctx, a, true)
	}
	for i := 0; i < 100 && ctx.Err() == nil; i++ {
		claim := randomID()
		row, err := w.service.store.claimEmail(ctx, w.service.clock.Now(), claim)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			w.log.Warn("hotel booking email storage unavailable")
			break
		}
		sent := w.send(ctx, row) == nil
		if !sent {
			w.log.Warn("hotel booking confirmation email pending")
		}
		if err = w.service.store.emailResult(ctx, row.Key, claim, sent, w.service.clock.Now().Add(notificationBackoff(row.Attempts))); err != nil {
			w.log.Warn("hotel booking email acknowledgement unavailable")
		}
	}
}
func notificationBackoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 30 * time.Second
	case 2:
		return time.Minute
	case 3:
		return 5 * time.Minute
	default:
		return 30 * time.Minute
	}
}
func (w *Worker) send(ctx context.Context, row emailRow) error {
	var req CreateRequest
	var result Reservation
	if json.Unmarshal([]byte(row.Canonical), &req) != nil || json.Unmarshal(row.Result, &result) != nil {
		return ErrUnavailable
	}
	contact, _, err := w.contacts.Get(ctx)
	if err != nil {
		return ErrUnavailable
	}
	room, err := w.service.rooms.Get(ctx, result.RoomTypeID)
	if err != nil {
		return ErrUnavailable
	}
	if contact.Phones.Hotel.Number == "" {
		return ErrUnavailable
	}
	plainText, html, err := renderConfirmation(w.confirmation, result, room.Name, contact.Phones.Hotel.Number, w.service.cancellation.link(row.Key))
	if err != nil {
		return ErrUnavailable
	}
	return w.mailer.SendAlternative(ctx, req.Guest.Email, "Бронирование Сити Парк подтверждено", plainText, html)
}
