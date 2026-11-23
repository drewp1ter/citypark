-- +goose Up
CREATE TABLE hotel_booking_telegram_outbox (
    attempt_key uuid PRIMARY KEY REFERENCES hotel_booking_attempts(key),
    sent boolean NOT NULL DEFAULT false,
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL,
    claim uuid,
    lease_until timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX hotel_booking_telegram_pending
    ON hotel_booking_telegram_outbox(next_attempt_at) WHERE NOT sent;

-- +goose Down
DROP TABLE hotel_booking_telegram_outbox;
