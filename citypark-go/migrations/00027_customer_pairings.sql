-- +goose Up
CREATE TABLE customer_pairings (
    id text PRIMARY KEY,
    code_hash bytea NOT NULL UNIQUE CHECK (octet_length(code_hash) = 32),
    source_id bigint NOT NULL REFERENCES users(id),
    destination_id bigint REFERENCES users(id),
    source_session_hash bytea NOT NULL CHECK (octet_length(source_session_hash) = 32),
    destination_session_hash bytea CHECK (destination_session_hash IS NULL OR octet_length(destination_session_hash) = 32),
    expires_at timestamptz NOT NULL,
    redeemed_at timestamptz,
    source_confirmed_at timestamptz,
    destination_confirmed_at timestamptz,
    completed_at timestamptz,
    canceled_at timestamptz
);
CREATE INDEX customer_pairings_source_pending_idx ON customer_pairings(source_id) WHERE completed_at IS NULL AND canceled_at IS NULL;

-- +goose Down
DROP TABLE customer_pairings;
