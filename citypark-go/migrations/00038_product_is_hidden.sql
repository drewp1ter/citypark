-- +goose Up
ALTER TABLE products ADD COLUMN is_hidden boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE products DROP COLUMN is_hidden;
