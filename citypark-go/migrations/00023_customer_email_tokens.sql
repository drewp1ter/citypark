-- +goose Up
CREATE TABLE customer_email_tokens (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose text NOT NULL CHECK (purpose IN ('verify','reset','change-email')),
    email text NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz
);
CREATE INDEX customer_email_tokens_user_idx ON customer_email_tokens(user_id,purpose);

-- +goose Down
DROP TABLE customer_email_tokens;
