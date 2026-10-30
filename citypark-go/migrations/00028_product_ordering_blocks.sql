-- +goose Up
ALTER TABLE products ADD COLUMN is_order_blocked boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE products DROP COLUMN is_order_blocked;
