-- +goose Up
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;
UPDATE users SET phone = NULL, updated_at = now() WHERE phone = '';

CREATE TABLE customer_identities (
    user_id bigint NOT NULL REFERENCES users(id),
    provider text NOT NULL CHECK (provider IN ('email', 'google', 'telegram')),
    subject text NOT NULL CHECK (subject <> ''),
    PRIMARY KEY (provider, subject),
    UNIQUE (user_id, provider)
);

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM users
        WHERE role = 'user' AND NULLIF(btrim(email), '') IS NOT NULL
          AND (password_hash <> '' OR legacy_password_hash <> '')
        GROUP BY lower(btrim(email)) HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'normalized customer email collision; resolve duplicate addresses before migration';
    END IF;
END $$;
-- +goose StatementEnd

INSERT INTO customer_identities (user_id, provider, subject)
SELECT id, 'email', lower(btrim(email)) FROM users
WHERE role = 'user' AND NULLIF(btrim(email), '') IS NOT NULL
  AND (password_hash <> '' OR legacy_password_hash <> '');

-- +goose Down
DROP TABLE customer_identities;
UPDATE users SET phone = 'unassigned:' || id, updated_at = now() WHERE phone IS NULL;
ALTER TABLE users ALTER COLUMN phone SET NOT NULL;
