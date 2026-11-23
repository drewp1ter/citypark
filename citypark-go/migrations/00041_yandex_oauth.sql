-- +goose Up
ALTER TABLE customer_oauth_attempts ADD COLUMN IF NOT EXISTS provider text NOT NULL DEFAULT 'google' CHECK (provider IN ('google', 'yandex'));
ALTER TABLE customer_identities DROP CONSTRAINT customer_identities_provider_check,
    ADD CONSTRAINT customer_identities_provider_check CHECK (provider IN ('email', 'google', 'telegram', 'yandex'));
ALTER TABLE customer_sessions DROP CONSTRAINT customer_sessions_method_check,
    ADD CONSTRAINT customer_sessions_method_check CHECK (method IN ('password', 'google', 'telegram', 'yandex'));

-- +goose Down
-- Refuse rollback while Yandex identities or sessions exist.
ALTER TABLE customer_identities DROP CONSTRAINT customer_identities_provider_check,
    ADD CONSTRAINT customer_identities_provider_check CHECK (provider IN ('email', 'google', 'telegram'));
ALTER TABLE customer_sessions DROP CONSTRAINT customer_sessions_method_check,
    ADD CONSTRAINT customer_sessions_method_check CHECK (method IN ('password', 'google', 'telegram'));
DELETE FROM customer_oauth_attempts WHERE provider = 'yandex';
ALTER TABLE customer_oauth_attempts DROP COLUMN provider;
