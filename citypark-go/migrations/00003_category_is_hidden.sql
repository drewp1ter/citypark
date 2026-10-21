-- +goose Up
ALTER TABLE categories ADD COLUMN is_hidden boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE categories DROP COLUMN is_hidden;
