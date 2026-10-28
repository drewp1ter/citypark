-- +goose Up
ALTER TABLE orders ADD COLUMN iiko_order_id text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE orders DROP COLUMN iiko_order_id;
