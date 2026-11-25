-- +goose Up
ALTER TABLE products ADD COLUMN is_on_stop_list boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE products DROP COLUMN is_on_stop_list;
