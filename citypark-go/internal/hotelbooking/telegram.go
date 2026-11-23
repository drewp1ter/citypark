package hotelbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type plainMessenger interface {
	SendPlainMessage(context.Context, string, string) error
}

func (w *Worker) ConfigureTelegram(sender plainMessenger, chatID string) {
	w.telegram = sender
	w.telegramChatID = chatID
}

type telegramRow struct {
	Key      string          `db:"attempt_key"`
	Attempts int             `db:"attempts"`
	Result   json.RawMessage `db:"result"`
}

func (s *Store) claimTelegram(ctx context.Context, now time.Time, claim string) (telegramRow, error) {
	var row telegramRow
	err := s.db.GetContext(ctx, &row, `
		WITH claimed AS (
			UPDATE hotel_booking_telegram_outbox
			SET claim=$2, lease_until=$3, attempts=attempts+1, updated_at=now()
			WHERE attempt_key=(
				SELECT attempt_key FROM hotel_booking_telegram_outbox
				WHERE NOT sent AND next_attempt_at<=$1 AND (lease_until IS NULL OR lease_until<=$1)
				ORDER BY next_attempt_at, attempt_key FOR UPDATE SKIP LOCKED LIMIT 1
			)
			RETURNING attempt_key, attempts
		)
		SELECT c.attempt_key, c.attempts, a.result
		FROM claimed c JOIN hotel_booking_attempts a ON a.key=c.attempt_key`, now, claim, now.Add(time.Minute))
	return row, err
}

func (s *Store) telegramResult(ctx context.Context, key, claim string, sent bool, next time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE hotel_booking_telegram_outbox
		SET sent=$3, next_attempt_at=$4, claim=NULL, lease_until=NULL, updated_at=now()
		WHERE attempt_key=$1 AND claim=$2`, key, claim, sent, next)
	return err
}

func (w *Worker) tickTelegram(ctx context.Context) {
	for i := 0; i < 100 && ctx.Err() == nil; i++ {
		claim := randomID()
		row, err := w.service.store.claimTelegram(ctx, w.service.clock.Now(), claim)
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		if err != nil {
			w.log.Warn("hotel booking Telegram storage unavailable")
			return
		}
		sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		sent := w.sendTelegram(sendCtx, row) == nil
		cancel()
		if !sent {
			w.log.Warn("hotel booking Telegram notification pending")
		}
		next := w.service.clock.Now().Add(notificationBackoff(row.Attempts))
		if err := w.service.store.telegramResult(ctx, row.Key, claim, sent, next); err != nil {
			w.log.Warn("hotel booking Telegram acknowledgement unavailable")
		}
	}
}

func (w *Worker) sendTelegram(ctx context.Context, row telegramRow) error {
	var result Reservation
	if err := json.Unmarshal(row.Result, &result); err != nil {
		return ErrUnavailable
	}
	reference := result.ReservationNumber
	if reference == "" {
		reference = result.ReservationID
	}
	roomName := result.RoomTypeID
	if room, err := w.service.rooms.Get(ctx, result.RoomTypeID); err == nil && room.Name != "" {
		roomName = room.Name
	}
	body := fmt.Sprintf("Новое бронирование гостиницы\nБронирование: %s\nТип номера: %s\nЗаезд: %s\nВыезд: %s\nГостей: %d\nСтоимость: %s ₽",
		reference, roomName, result.CheckIn, result.CheckOut, result.Guests, result.Price.TotalKopecks.String())
	return w.telegram.SendPlainMessage(ctx, w.telegramChatID, body)
}
