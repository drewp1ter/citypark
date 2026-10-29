-- +goose Up
-- users remains the legacy contact/staff table. account_id identifies the
-- authenticated owner independently of the phone-based customer_id.
ALTER TABLE orders
    ADD COLUMN account_id bigint REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN contact_name text,
    ADD COLUMN contact_phone text,
    ADD COLUMN delivery_city text,
    ADD COLUMN delivery_address text;

-- Only existing customer records with credentials retain ownership. A phone,
-- email, or name alone is never enough to claim historical orders.
UPDATE orders o
SET account_id = CASE
        WHEN u.role = 'user' AND (u.password_hash <> '' OR u.legacy_password_hash <> '')
        THEN u.id
    END,
    contact_name = u.name,
    contact_phone = u.phone,
    delivery_city = (SELECT a.city FROM addresses a WHERE a.id = o.address_id),
    delivery_address = (SELECT a.address FROM addresses a WHERE a.id = o.address_id),
    updated_at = now()
FROM users u
WHERE u.id = o.customer_id;

CREATE INDEX orders_account_id_idx ON orders (account_id, id DESC)
    WHERE account_id IS NOT NULL;

-- +goose Down
DROP INDEX orders_account_id_idx;
ALTER TABLE orders
    DROP COLUMN delivery_address,
    DROP COLUMN delivery_city,
    DROP COLUMN contact_phone,
    DROP COLUMN contact_name,
    DROP COLUMN account_id;
