-- +goose Up
CREATE TABLE customer_oauth_attempts (
    state_hash bytea PRIMARY KEY CHECK (octet_length(state_hash) = 32),
    nonce_hash bytea NOT NULL CHECK (octet_length(nonce_hash) = 32),
    flow_hash bytea NOT NULL CHECK (octet_length(flow_hash) = 32),
    pkce_verifier text NOT NULL,
    intent text NOT NULL CHECK (intent IN ('login','link')),
    user_id bigint REFERENCES users(id),
    session_hash bytea,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CHECK ((intent='login' AND user_id IS NULL AND session_hash IS NULL)
        OR (intent='link' AND user_id IS NOT NULL AND session_hash IS NOT NULL))
);
CREATE INDEX customer_oauth_attempts_expires_idx ON customer_oauth_attempts(expires_at);

-- +goose Down
DROP TABLE customer_oauth_attempts;
