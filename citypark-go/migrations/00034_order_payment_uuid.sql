-- +goose Up
ALTER TABLE orders ADD COLUMN payment_uuid uuid NOT NULL DEFAULT gen_random_uuid();
CREATE UNIQUE INDEX orders_payment_uuid_idx ON orders (payment_uuid);

-- +goose Down
DROP INDEX orders_payment_uuid_idx;
ALTER TABLE orders DROP COLUMN payment_uuid;
