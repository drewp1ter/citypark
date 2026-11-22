-- +goose Up
CREATE TABLE hotel_booking_quotes (
    id uuid PRIMARY KEY,
    context_hash text NOT NULL,
    room_type_id text NOT NULL CHECK (room_type_id IN ('single','single-balcony','double-twin','double-euro','vip')),
    check_in date NOT NULL,
    check_out date NOT NULL CHECK (check_out > check_in),
    guests integer NOT NULL CHECK (guests > 0),
    nightly_price_kopecks bigint NOT NULL CHECK (nightly_price_kopecks > 0),
    total_kopecks bigint NOT NULL CHECK (total_kopecks > 0),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL
);
CREATE TABLE hotel_booking_attempts (
    key uuid PRIMARY KEY,
    quote_id uuid NOT NULL REFERENCES hotel_booking_quotes(id),
    context_hash text NOT NULL,
    canonical_request text NOT NULL,
    state text NOT NULL CHECK (state IN ('pending','confirmed','rejected')),
    result jsonb,
    rejection_code text NOT NULL DEFAULT '',
    claim uuid,
    lease_until timestamptz,
    next_attempt_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX hotel_booking_pending ON hotel_booking_attempts(next_attempt_at) WHERE state = 'pending';
CREATE TABLE hotel_booking_email_outbox (
    attempt_key uuid PRIMARY KEY REFERENCES hotel_booking_attempts(key),
    state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','sent')),
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL,
    claim uuid,
    lease_until timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX hotel_booking_email_pending ON hotel_booking_email_outbox(next_attempt_at) WHERE state = 'pending';
CREATE TABLE hotel_booking_rate_limits (
    scope text NOT NULL,
    hash text NOT NULL,
    window_start timestamptz NOT NULL,
    counter integer NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (scope, hash, window_start)
);

-- +goose Down
DROP TABLE hotel_booking_rate_limits;
DROP TABLE hotel_booking_email_outbox;
DROP TABLE hotel_booking_attempts;
DROP TABLE hotel_booking_quotes;
