-- +goose Up
ALTER TABLE orders DROP COLUMN payment_type;

-- +goose Down
ALTER TABLE orders ADD COLUMN payment_type text NOT NULL DEFAULT 'online' CHECK (payment_type IN ('cash', 'online'));
