-- +goose Up
ALTER TABLE hotel_booking_telegram_outbox
    ADD COLUMN kind text NOT NULL DEFAULT 'confirmation' CHECK (kind IN ('confirmation', 'cancellation')),
    DROP CONSTRAINT hotel_booking_telegram_outbox_pkey,
    ADD PRIMARY KEY (attempt_key, kind);

-- +goose Down
DELETE FROM hotel_booking_telegram_outbox WHERE kind = 'cancellation';
ALTER TABLE hotel_booking_telegram_outbox
    DROP CONSTRAINT hotel_booking_telegram_outbox_pkey,
    DROP COLUMN kind,
    ADD PRIMARY KEY (attempt_key);
