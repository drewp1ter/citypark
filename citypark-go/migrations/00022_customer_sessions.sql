-- +goose Up
CREATE TABLE customer_sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    authenticated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    method text NOT NULL CHECK (method IN ('password','google','telegram')),
    surface text NOT NULL CHECK (surface IN ('web','mini')),
    revoked_at timestamptz
);
CREATE INDEX customer_sessions_user_id_idx ON customer_sessions(user_id);

-- +goose Down
DROP TABLE customer_sessions;
