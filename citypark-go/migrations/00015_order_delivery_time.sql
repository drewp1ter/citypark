-- +goose Up
ALTER TABLE orders ADD COLUMN delivery_time text NOT NULL DEFAULT ''
    CHECK (delivery_time ~ '^(([01][0-9]|2[0-3]):[0-5][0-9])?$');

-- +goose Down
ALTER TABLE orders DROP COLUMN delivery_time;
