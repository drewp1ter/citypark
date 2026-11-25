-- +goose Up
ALTER TABLE hotel_booking_email_outbox
    ADD COLUMN channel text NOT NULL DEFAULT 'email' CHECK (channel IN ('email', 'sms'));

-- +goose Down
DELETE FROM hotel_booking_email_outbox WHERE channel = 'sms';
ALTER TABLE hotel_booking_email_outbox DROP COLUMN channel;
