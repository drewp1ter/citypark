-- +goose Up
CREATE TABLE customer_telegram_proofs (
    proof_hash bytea PRIMARY KEY CHECK(octet_length(proof_hash)=32),
    expires_at timestamptz NOT NULL
);

-- +goose Down
DROP TABLE customer_telegram_proofs;
