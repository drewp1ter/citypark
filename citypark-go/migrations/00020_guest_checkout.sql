-- +goose Up
ALTER TABLE orders ADD COLUMN guest_checkout_hash text;
CREATE INDEX orders_guest_checkout_idx ON orders (guest_checkout_hash, id DESC)
    WHERE account_id IS NULL AND guest_checkout_hash IS NOT NULL;

-- +goose Down
DROP INDEX orders_guest_checkout_idx;
ALTER TABLE orders DROP COLUMN guest_checkout_hash;
