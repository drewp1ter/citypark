-- +goose Up
ALTER TABLE products ADD COLUMN full_description text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE products DROP COLUMN full_description;
